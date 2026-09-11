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

function stepScript(name) {
  const run = step(action, name).match(/\n      run: \|\n([\s\S]*)/);
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

const deriveScript = stepScript('Derive pull request status');

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

console.log('scanner action context tests passed');
