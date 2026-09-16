#!/usr/bin/env node

const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const action = fs.readFileSync(path.join(__dirname, '..', 'diff', 'action.yml'), 'utf8');
const scanAction = fs.readFileSync(path.join(__dirname, '..', 'scan', 'action.yml'), 'utf8');

// The steps are matched out of the YAML rather than parsed: the workflow runs
// this without installing node_modules, so there is no YAML parser.
function step(text, name) {
  const match = text.match(new RegExp(`\\n    - name: ${name}\\n([\\s\\S]*?)(?=\\n    (?:#|- name: ))|\\n    - name: ${name}\\n([\\s\\S]*)$`));
  assert(match, `${name} step is required`);
  return match[1] || match[2];
}

function stepScript(text, name) {
  const run = step(text, name).match(/\n      run: \|\n([\s\S]*)/);
  assert(run, `${name} must have a run block`);
  return run[1].replace(/^ {8}/gm, '');
}

// Asserts before indexing: a missing name would otherwise die with a TypeError
// naming neither the step nor the variable.
function envValue(text, name) {
  const match = text.match(new RegExp(`^ +${name}: (.*)$`, 'm'));
  assert(match, `${name} must be set`);
  return match[1];
}

// Runs the action's own bash under a fixture environment, so a change to the
// shell that the cases below do not expect fails here.
function deriveStatus(env) {
  const outputFile = fs.mkdtempSync(path.join(os.tmpdir(), 'scanner-action-')) + '/output';
  fs.writeFileSync(outputFile, '');

  let failed = false;
  try {
    execFileSync('bash', ['-c', deriveScript], {
      env: { PATH: process.env.PATH, GITHUB_OUTPUT: outputFile, ...env },
      stdio: 'pipe',
    });
  } catch {
    failed = true;
  }

  const outputs = {};
  for (const line of fs.readFileSync(outputFile, 'utf8').split('\n')) {
    const [name, ...value] = line.split('=');
    if (name) outputs[name] = value.join('=');
  }
  return { failed, ...outputs };
}

const deriveScript = stepScript(action, 'Derive pull request status');

const fixture = {
  INPUT_PR_STATUS: '',
  INPUT_BASE_PATH: 'base',
  INPUT_HEAD_PATH: 'head',
  EVENT_NAME: 'pull_request',
  EVENT_ACTION: 'opened',
  EVENT_PR_MERGED: '',
};

for (const test of [
  { name: 'opened PR scans and updates status', env: {}, pr_status: 'OPEN', scan: 'true' },
  { name: 'synchronized PR scans and updates status', env: { EVENT_ACTION: 'synchronize' }, pr_status: 'OPEN', scan: 'true' },
  { name: 'closed PR only updates status', env: { EVENT_ACTION: 'closed', EVENT_PR_MERGED: 'false' }, pr_status: 'CLOSED', scan: 'false' },
  { name: 'merged PR only updates status', env: { EVENT_ACTION: 'closed', EVENT_PR_MERGED: 'true' }, pr_status: 'MERGED', scan: 'false' },
  { name: 'push does not scan or update a PR', env: { EVENT_NAME: 'push', EVENT_ACTION: '', INPUT_BASE_PATH: '', INPUT_HEAD_PATH: '' }, pr_status: '', scan: 'false' },
  {
    name: 'explicit status without checkouts only updates status',
    env: { INPUT_PR_STATUS: 'OPEN', INPUT_BASE_PATH: '', INPUT_HEAD_PATH: '', EVENT_NAME: 'workflow_dispatch', EVENT_ACTION: '' },
    pr_status: 'OPEN',
    scan: 'false',
  },
  {
    name: 'explicit status with checkouts still scans',
    env: { INPUT_PR_STATUS: 'OPEN', EVENT_NAME: 'workflow_dispatch', EVENT_ACTION: '' },
    pr_status: 'OPEN',
    scan: 'true',
  },
  {
    name: 'explicit merged status still scans when both paths are set',
    env: { INPUT_PR_STATUS: 'MERGED', EVENT_ACTION: 'closed', EVENT_PR_MERGED: 'false' },
    pr_status: 'MERGED',
    scan: 'true',
  },
  {
    name: 'pull_request_target with checkouts still scans',
    env: { EVENT_NAME: 'pull_request_target', EVENT_ACTION: 'synchronize' },
    pr_status: '',
    scan: 'true',
  },
  {
    name: 'explicit status with one checkout only updates status',
    env: { INPUT_PR_STATUS: 'MERGED', INPUT_HEAD_PATH: '' },
    pr_status: 'MERGED',
    scan: 'false',
  },
  { name: 'one checkout does not scan', env: { INPUT_HEAD_PATH: '' }, pr_status: 'OPEN', scan: 'false' },
]) {
  const got = deriveStatus({ ...fixture, ...test.env });
  assert.equal(got.failed, false, test.name);
  assert.equal(got.pr_status, test.pr_status, test.name);
  assert.equal(got.scan, test.scan, test.name);
}

// An injected line must not become an output for the later steps to read.
const injected = deriveStatus({ ...fixture, INPUT_PR_STATUS: 'OPEN\nscan=true' });
assert.equal(injected.failed, true, 'an invalid pr-status must fail the step');
assert.equal(injected.scan, undefined, 'a rejected pr-status must write no outputs');

const scanner = step(action, 'Run scanner');
const status = step(action, 'Update PR status');

// Matches a write, not the word: the action explains in a comment why it does
// not use GITHUB_ENV.
assert(!/>>\s*"?\$\{?GITHUB_ENV/.test(action), 'the VCS context must not leak into the caller job through GITHUB_ENV');
assert(scanner.includes("if: steps.context.outputs.scan == 'true'"));
assert(status.includes("if: steps.context.outputs.pr_status != ''"));
assert(!scanner.includes('--pr-number'), 'the action must use the VCS environment contract');
assert(!status.includes('--pr-number'), 'status must use the same VCS environment contract');

// Both steps build the dashboard's PR key, so their inputs must be identical.
for (const name of ['INFRACOST_VCS_PROVIDER', 'INFRACOST_VCS_REPOSITORY_URL', 'INFRACOST_VCS_PULL_REQUEST_ID', 'INFRACOST_VCS_PULL_REQUEST_URL']) {
  assert.equal(envValue(scanner, name), envValue(status, name), `${name} must match in both steps`);
}

// scan derives these from the path it scans, so an inherited value must not
// reach it: the env: block a caller writes for diff is inherited here too.
const scanActionStep = step(scanAction, 'Run scanner');
for (const name of [
  'INFRACOST_VCS_BRANCH',
  'INFRACOST_VCS_COMMIT_SHA',
  'INFRACOST_VCS_COMMIT_MESSAGE',
  'INFRACOST_VCS_COMMIT_AUTHOR_NAME',
  'INFRACOST_VCS_COMMIT_AUTHOR_EMAIL',
  'INFRACOST_VCS_COMMIT_TIMESTAMP',
]) {
  assert.equal(envValue(scanActionStep, name), "''", `scan must blank inherited ${name}`);
}

// The download step is byte-identical in both actions: it has been kept in
// sync by hand and nothing checked it until now.
const download = step(action, 'Download scanner');
assert.equal(download, step(scanAction, 'Download scanner'), 'the download step must be identical in diff and scan');
assert(!/GH_TOKEN|github-token|github\.token/.test(download), 'the download must stay anonymous');
assert(!/INFRACOST_SCANNER_BASE_URL/.test(download), 'the base URL must not be overridable from the caller job env');

const downloadScript = stepScript(action, 'Download scanner');

// uname and curl are stubbed on PATH: the step then runs offline and the archive
// name does not depend on the host running the tests. sha256sum remains real so
// these tests verify the archive against its advertised digest.
const stubs = {
  uname: `#!/usr/bin/env bash
case "\${1:-}" in
  -s) printf '%s\\n' "$UNAME_S" ;;
  -m) printf '%s\\n' "$UNAME_M" ;;
esac
`,
  curl: `#!/usr/bin/env bash
set -u
url=""; out=""; prev=""
for a in "$@"; do
  case "$a" in https://*) url="$a" ;; esac
  if [ "$prev" = "-o" ]; then out="$a"; fi
  prev="$a"
done
printf '%s\\n' "$url" >> "$CURL_LOG"
if [ -n "\${CURL_404:-}" ]; then
  case "$url" in *"$CURL_404"*) exit 22 ;; esac
fi
case "$url" in
  */releases/latest) printf '%s' "$LATEST_REDIRECT" ;;
  *checksums.txt) cp "$CHECKSUMS_SRC" "$out" ;;
  *.tar.gz) cp "$ARCHIVE_SRC" "$out" ;;
  *) exit 22 ;;
esac
`,
};

function downloadScanner({
  version = 'latest',
  unameS = 'Linux',
  unameM = 'x86_64',
  latestRedirect = 'https://github.com/infracost/ci/releases/tag/v1.2.3',
  notFound = '',
  checksumName = null,
  corruptArchive = false,
} = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'scanner-download-'));
  const [bin, runnerTemp, workspace, payload] = ['bin', 'runner-temp', 'workspace', 'payload']
    .map((name) => { const p = path.join(dir, name); fs.mkdirSync(p); return p; });

  for (const [name, body] of Object.entries(stubs)) {
    fs.writeFileSync(path.join(bin, name), body, { mode: 0o755 });
  }

  // A real tarball, so the step's tar has something to extract.
  fs.writeFileSync(path.join(payload, 'infracost-scanner'), '#!/bin/sh\n');
  const archiveSrc = path.join(dir, 'archive.tar.gz');
  execFileSync('tar', ['-czf', archiveSrc, '-C', payload, 'infracost-scanner']);
  const corruptArchiveSrc = path.join(dir, 'corrupt-archive.tar.gz');
  fs.writeFileSync(path.join(payload, 'infracost-scanner'), '#!/bin/sh\necho corrupt\n');
  execFileSync('tar', ['-czf', corruptArchiveSrc, '-C', payload, 'infracost-scanner']);

  const archive = `infracost-scanner_${unameS.toLowerCase()}_${unameM === 'x86_64' ? 'amd64' : 'arm64'}.tar.gz`;
  const checksumsSrc = path.join(dir, 'checksums.txt');
  const checksumCommand = process.platform === 'darwin' ? ['shasum', '-a', '256'] : ['sha256sum'];
  const checksum = execFileSync(checksumCommand[0], [...checksumCommand.slice(1), archiveSrc]).toString().split(/\s+/)[0];
  fs.writeFileSync(checksumsSrc, `${checksum}  ${checksumName === null ? archive : checksumName}\n`);

  const curlLog = path.join(dir, 'curl.log');
  const githubPath = path.join(dir, 'github-path');
  fs.writeFileSync(curlLog, '');
  fs.writeFileSync(githubPath, '');

  let failed = false;
  let output = '';
  try {
    output = execFileSync('bash', ['-c', downloadScript], {
      cwd: workspace,
      env: {
        PATH: `${bin}:${process.env.PATH}`,
        RUNNER_TEMP: runnerTemp,
        GITHUB_PATH: githubPath,
        VERSION: version,
        CURL_LOG: curlLog,
        LATEST_REDIRECT: latestRedirect,
        CURL_404: notFound,
        ARCHIVE_SRC: corruptArchive ? corruptArchiveSrc : archiveSrc,
        CHECKSUMS_SRC: checksumsSrc,
        UNAME_S: unameS,
        UNAME_M: unameM,
      },
      stdio: 'pipe',
    }).toString();
  } catch (e) {
    failed = true;
    output = `${e.stdout}${e.stderr}`;
  }

  return {
    failed,
    output,
    archive,
    urls: fs.readFileSync(curlLog, 'utf8').split('\n').filter(Boolean),
    temp: fs.readdirSync(runnerTemp).sort(),
    workspace: fs.readdirSync(workspace),
    githubPath: fs.readFileSync(githubPath, 'utf8').trim(),
    runnerTemp,
  };
}

const RELEASES = 'https://github.com/infracost/ci/releases';

const pinned = downloadScanner({ version: '1.2.3' });
assert.equal(pinned.failed, false, 'a pinned version must download');
assert.deepEqual(pinned.urls, [
  `${RELEASES}/download/v1.2.3/${pinned.archive}`,
  `${RELEASES}/download/v1.2.3/checksums.txt`,
], 'a pinned version must not ask for the latest release');
assert.deepEqual(pinned.temp, ['checksums.txt', 'infracost-scanner', pinned.archive].sort(), 'every file must land in RUNNER_TEMP');
assert.deepEqual(pinned.workspace, [], 'the working directory must be untouched');
assert.equal(pinned.githubPath, pinned.runnerTemp, 'RUNNER_TEMP must be added to GITHUB_PATH');

const latest = downloadScanner({ version: 'latest' });
assert.equal(latest.failed, false, 'latest must download');
assert.deepEqual(latest.urls, [
  `${RELEASES}/latest`,
  `${RELEASES}/download/v1.2.3/${latest.archive}`,
  `${RELEASES}/download/v1.2.3/checksums.txt`,
], 'latest must resolve one tag and pin both assets to it');

// A repo with no releases redirects /releases/latest to the releases list.
const noReleases = downloadScanner({ version: 'latest', latestRedirect: `${RELEASES}` });
assert.equal(noReleases.failed, true, 'an unresolvable latest must fail the step');
assert(noReleases.output.includes('Could not resolve the latest'), 'an unresolvable latest must say so');
assert.deepEqual(noReleases.urls, [`${RELEASES}/latest`], 'an unresolvable latest must not build a download URL');

const inaccessibleLatest = downloadScanner({ version: 'latest', notFound: '/releases/latest' });
assert.equal(inaccessibleLatest.failed, true, 'an inaccessible latest release must fail the step');
assert(inaccessibleLatest.output.includes('Could not resolve the latest'), 'an inaccessible latest release must say so');

const mac = downloadScanner({ version: '1.2.3', unameS: 'Darwin', unameM: 'arm64' });
assert.equal(mac.failed, false, 'darwin/arm64 must download');
assert(mac.urls[0].endsWith('/infracost-scanner_darwin_arm64.tar.gz'), 'uname must map to the asset name');

const unresolvable = downloadScanner({ version: '0.2.9', notFound: 'download/v0.2.9' });
assert.equal(unresolvable.failed, true, 'a version that does not exist must fail the step');
assert(unresolvable.output.includes('Could not download scanner archive'), 'a failed archive download must identify the failed URL');
assert.deepEqual(unresolvable.workspace, [], 'a failed download must leave the working directory untouched');

const invalidVersion = downloadScanner({ version: '0.0.0/../../evilorg/evilrepo' });
assert.equal(invalidVersion.failed, true, 'a version containing a path must fail the step');
assert(invalidVersion.output.includes('version must be a semantic version'), 'an invalid version must explain why it failed');
assert.deepEqual(invalidVersion.urls, [], 'an invalid version must not make a download request');

const leadingV = downloadScanner({ version: 'v1.2.3' });
assert.equal(leadingV.failed, false, 'a leading v must be accepted');
assert.equal(leadingV.urls[0], `${RELEASES}/download/v1.2.3/${leadingV.archive}`, 'a leading v must not be doubled in the tag');

const unsupportedPlatform = downloadScanner({ unameM: 'i686' });
assert.equal(unsupportedPlatform.failed, true, 'an unsupported architecture must fail the step');
assert(unsupportedPlatform.output.includes('Unsupported scanner architecture'), 'an unsupported architecture must say so');
assert.deepEqual(unsupportedPlatform.urls, [], 'an unsupported architecture must not request a nonexistent asset');

const unlisted = downloadScanner({ version: '1.2.3', checksumName: 'infracost-scanner_other.tar.gz' });
assert.equal(unlisted.failed, true, 'an archive absent from checksums.txt must fail the step');
assert(!unlisted.temp.includes('infracost-scanner'), 'an unverified archive must never reach tar');

const corrupted = downloadScanner({ version: '1.2.3', corruptArchive: true });
assert.equal(corrupted.failed, true, 'an archive with a mismatched checksum must fail the step');
assert(!corrupted.temp.includes('infracost-scanner'), 'a mismatched archive must never reach tar');

console.log('scanner action context tests passed');
