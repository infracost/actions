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

// Scoped to one input: a bare default: reads whichever input is declared first.
function inputValue(text, input, name) {
  const block = text.match(new RegExp(`\\n  ${input}:\\n([\\s\\S]*?)(?=\\n  [a-z-]+:\\n|\\nruns:)`));
  assert(block, `${input} input is required`);
  return envValue(block[1], name);
}

// Reads both GITHUB_OUTPUT forms: name=value and the name<<delimiter heredoc.
function readOutputs(file) {
  const outputs = {};
  const lines = fs.readFileSync(file, 'utf8').split('\n');
  for (let i = 0; i < lines.length; i++) {
    const heredoc = lines[i].match(/^([^=<]+)<<(.+)$/);
    if (heredoc) {
      const end = lines.indexOf(heredoc[2], i + 1);
      assert(end > i, `unterminated heredoc for ${heredoc[1]}`);
      outputs[heredoc[1]] = lines.slice(i + 1, end).join('\n');
      i = end;
      continue;
    }
    const [name, ...value] = lines[i].split('=');
    if (name) outputs[name] = value.join('=');
  }
  return outputs;
}

// Runs the action's own bash under a fixture environment, so a change to the
// shell that the cases below do not expect fails here.
function runStep(script, env) {
  const outputFile = fs.mkdtempSync(path.join(os.tmpdir(), 'scanner-action-')) + '/output';
  fs.writeFileSync(outputFile, '');

  let failed = false;
  try {
    execFileSync('bash', ['-c', script], {
      env: { PATH: process.env.PATH, GITHUB_OUTPUT: outputFile, ...env },
      stdio: 'pipe',
    });
  } catch {
    failed = true;
  }

  return { failed, ...readOutputs(outputFile) };
}

// Runs a step against a docker shim, so the assertions see the argv the bash
// builds rather than the source it is built from.
function dockerArgv(script, env) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'scanner-docker-'));
  const argvFile = path.join(dir, 'argv');
  const shim = path.join(dir, 'docker');
  fs.writeFileSync(shim, `#!/bin/sh\nprintf '%s\\n' "$@" > '${argvFile}'\n`);
  fs.chmodSync(shim, 0o755);

  const workspace = fs.mkdtempSync(path.join(os.tmpdir(), 'scanner-ws-'));
  fs.mkdirSync(path.join(workspace, 'base'));
  fs.mkdirSync(path.join(workspace, 'head'));

  const got = runStep(script, {
    PATH: `${dir}:${process.env.PATH}`,
    GITHUB_WORKSPACE: workspace,
    ...env,
  });

  const argv = fs.existsSync(argvFile)
    ? fs.readFileSync(argvFile, 'utf8').split('\n').filter(Boolean)
    : [];
  return { ...got, argv, workspace };
}

// The names each bare -e forwards.
function forwarded(argv) {
  return argv.filter((arg, i) => argv[i - 1] === '-e');
}

const deriveScript = stepScript(action, 'Derive pull request status');
const deriveStatus = (env) => runStep(deriveScript, env);

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

// The tag resolution and the 2.0.0 floor are the same bash in both actions.
for (const [file, text] of [['diff', action], ['scan', scanAction]]) {
  const resolveScript = stepScript(text, 'Resolve image tag');
  const resolve = (version) => runStep(resolveScript, { VERSION: version });

  for (const test of [
    { version: 'latest', ref: 'ghcr.io/infracost/ci:latest' },
    { version: '2', ref: 'ghcr.io/infracost/ci:2' },
    { version: '2.0', ref: 'ghcr.io/infracost/ci:2.0' },
    { version: '2.0.1', ref: 'ghcr.io/infracost/ci:2.0.1' },
    { version: '3.1.0', ref: 'ghcr.io/infracost/ci:3.1.0' },
    { version: 'sha256:0123abcd', ref: 'ghcr.io/infracost/ci@sha256:0123abcd' },
  ]) {
    const got = resolve(test.version);
    assert.equal(got.failed, false, `${file}: version ${test.version} must resolve`);
    assert.equal(got.ref, test.ref, `${file}: version ${test.version}`);
  }

  // A tag that is neither numeric nor a known name reaches docker as a 404
  // rather than as this step's error, so it is rejected here.
  for (const version of [
    '', '0.1.0', '0.1.6', '0.2.10', '1', '1.9.9', '0', '00.1', '01.2.3',
    'v0.1.6', 'v2.0.1', ' 1', '+1', '2.0.1 ', 'evil:latest',
  ]) {
    const got = resolve(version);
    assert.equal(got.failed, true, `${file}: version ${JSON.stringify(version)} must fail`);
    assert.equal(got.ref, undefined, `${file}: version ${JSON.stringify(version)} must write no outputs`);
  }

  // A major too long for [ must still resolve rather than error on the compare.
  const long = resolve('99999999999999999999');
  assert.equal(long.failed, false, `${file}: a 20-digit major must resolve`);
  assert.equal(long.ref, 'ghcr.io/infracost/ci:99999999999999999999', `${file}: a 20-digit major`);

  // A version holding a newline must not become a second output line.
  const forged = resolve('2\nref=evil');
  assert.equal(forged.failed, true, `${file}: a multi-line version must fail`);
  assert.equal(forged.ref, undefined, `${file}: a multi-line version must write no outputs`);

  // The regression test for FIX-726: the scanner ships from infracost/ci. The
  // ban is on the release source, so an error may still name a v4 uses: value.
  assert(text.includes('ghcr.io/infracost/ci'), `${file} must run ghcr.io/infracost/ci`);
  assert(!/--repo "?infracost\/actions/.test(text), `${file} must not resolve the scanner from infracost/actions`);
  assert(!text.includes('gh release'), `${file} must not download a release`);
  assert.equal(inputValue(text, 'version', 'default'), '"2"', `${file}: version must default to the 2 series`);

  const run = step(text, 'Run scanner');
  assert(run.includes('docker run'), `${file}: Run scanner must use docker run`);
  assert(/--user "\$\(id -u\):\$\(id -g\)"/.test(run), `${file}: docker run must not run as root`);
  assert(run.includes('-e HOME=/tmp'), `${file}: a non-root user needs a writable HOME`);
  assert(run.includes('-v "${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}"'), `${file}: docker run must mount the workspace`);
  assert(run.includes('-w "${GITHUB_WORKSPACE}"'), `${file}: relative paths must resolve against the workspace`);

  // The argv, not the source: the forwarding is a loop over the step's own env.
  const runScript = stepScript(text, 'Run scanner');
  const inputs = file === 'diff'
    ? { INPUT_BASE_PATH: 'base', INPUT_HEAD_PATH: 'head' }
    : { INPUT_PATH: 'base' };
  const scanEnv = {
    ...inputs,
    IMAGE_REF: 'ghcr.io/infracost/ci:2',
    GITHUB_TOKEN: 'token',
    INFRACOST_CLI_AUTHENTICATION_TOKEN: 'key',
    INFRACOST_VCS_BRANCH: 'feature',
    INFRACOST_CLI_PLUGIN_AUTO_UPDATE: 'true',
    TF_VAR_instance_type: 'm5.4xlarge',
    AWS_ACCESS_KEY_ID: 'AKIA',
    HTTPS_PROXY: 'http://proxy:3128',
  };

  const invocation = dockerArgv(runScript, scanEnv);
  assert.equal(invocation.failed, false, `${file}: Run scanner must invoke docker`);
  assert(invocation.argv.includes('--pull'), `${file}: a mutable tag must not be served from the image cache`);
  assert.equal(invocation.argv[invocation.argv.indexOf('--pull') + 1], 'always', `${file}: --pull always`);

  const names = forwarded(invocation.argv);
  for (const name of ['INFRACOST_CLI_AUTHENTICATION_TOKEN', 'INFRACOST_VCS_BRANCH', 'TF_VAR_instance_type', 'AWS_ACCESS_KEY_ID', 'HTTPS_PROXY']) {
    assert(names.includes(name), `${file}: ${name} must reach the container`);
  }
  // The image bakes the plugin cache in a root-owned directory.
  assert(!names.includes('INFRACOST_CLI_PLUGIN_AUTO_UPDATE'), `${file}: a caller must not override the baked plugin settings`);

  // Only the workspace is mounted, so an outside path must name the mount.
  const outside = dockerArgv(runScript, { ...scanEnv, ...(file === 'diff' ? { INPUT_BASE_PATH: os.tmpdir() } : { INPUT_PATH: os.tmpdir() }) });
  assert.equal(outside.failed, true, `${file}: a path outside the workspace must fail the step`);
  assert.deepEqual(outside.argv, [], `${file}: a path outside the workspace must not reach docker`);

  // A runner without docker must fail on the missing binary, not on its output.
  const noDocker = dockerArgv(runScript, { ...scanEnv, PATH: fs.mkdtempSync(path.join(os.tmpdir(), 'scanner-nodocker-')) });
  assert.equal(noDocker.failed, true, `${file}: a runner without docker must fail the step`);
}

assert(!scanAction.includes('github-token'), 'scan needs no token: the image is public and it posts nothing');

const scanner = step(action, 'Run scanner');
const status = step(action, 'Update PR status');

// Matches a write, not the word: the action explains in a comment why it does
// not use GITHUB_ENV.
assert(!/>>\s*"?\$\{?GITHUB_ENV/.test(action), 'the VCS context must not leak into the caller job through GITHUB_ENV');
assert(scanner.includes("if: steps.context.outputs.scan == 'true'"));
assert(status.includes("if: steps.context.outputs.pr_status != ''"));
assert(!scanner.includes('--pr-number'), 'the action must use the VCS environment contract');
assert(!status.includes('--pr-number'), 'status must use the same VCS environment contract');
assert(scanner.includes('-e GITHUB_TOKEN'), 'diff must forward the comment token to the container');
assert(status.includes('docker run') && status.includes('"${!INFRACOST_@}"'), 'status must run in the container with the VCS contract');

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

console.log('scanner action context tests passed');
