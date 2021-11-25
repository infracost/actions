#!/usr/bin/env node
const { spawn } = require('child_process');
require('dotenv').config()

const args = process.argv.slice(2);
const update = args.length > 0 && args[0] === 'true';

let command = `act \
-W .github/workflows/examples_test.yml \
-s GITHUB_TOKEN=$GITHUB_TOKEN \
-s INFRACOST_API_KEY=$(infracost configure get api_key) \
-s TFC_TOKEN=$TFC_TOKEN \
--artifact-server-path=.act/artifacts`;

if (update) {
  command += ` --env UPDATE_GOLDEN_FILES=true -b`;
}

console.log(`Running ${command}`);

const child = spawn('bash', ['-c', command], { env: process.env });

child.stdout.on('data', (data) => {
  process.stdout.write(data.toString()); 
});

child.stderr.on('data', (data) => {
  process.stderr.write(data.toString()); 
});
