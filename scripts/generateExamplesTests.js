#!/usr/bin/env node

// This file generates a GitHub action to test the examples by extracting the
// examples from each README file, modifying them slightly and then writing
// them to a GitHub action.

const fs = require('fs');
const yaml = require('js-yaml');
const {env} = require('process');

const examplesTestWorkflowPath = './.github/workflows/examples_test.yml';
const examplesDir = 'examples';
const exampleRegex =
  /\[\/\/\]: <> \(BEGIN EXAMPLE\)\n```.*\n((.|\n)*?)```\n\[\/\/\]: <> \(END EXAMPLE\)/gm;

const localSkipJobs = [
  // These jobs are skipped locally until https://github.com/nektos/act/issues/769 is fixed
  'multi-project-matrix',
  'multi-project-matrix-merge',
]

const workflowTemplate = {
  name: 'Run examples',
  on: {
    push: {
      branches: ['master'],
    },
    pull_request: {},
  },

  defaults: {
    run: {
      shell: 'bash',
    },
  },

  jobs: {},
};

// Finds all the examples in a file
function extractExamples(file) {
  const content = fs.readFileSync(file, 'utf8');
  const matches = [...content.matchAll(exampleRegex)];
  return matches.map((match) => yaml.load(match[1]));
}

// Extracts all the examples from a directory by reading all the README files
function extractAllExamples(examplesDir) {
  const examples = [];

  for (const dir of fs.readdirSync(examplesDir)) {
    if (!fs.statSync(`${examplesDir}/${dir}`).isDirectory()) {
      continue;
    }

    console.log(
      `Generating GitHub Actions workflow job for ${examplesDir}/${dir}`
    );

    const filename = `${examplesDir}/${dir}/README.md`;

    try {
      if (!fs.existsSync(filename)) {
        console.error(`Skipping ${dir} since no README.md file was found`);
        continue;
      }

      examples.push(...extractExamples(filename));
    } catch (err) {
      console.error(`Error reading YAML file ${filename}: ${err}`);
      continue;
    }
  }

  return examples;
}

// Modifies the examples by:
// 1. Replacing any infracost/actions steps with the local path
// 2. Replacing the infracost/actions/comment step with a step that comment contents
function fixupExamples(examples) {
  for (const example of examples) {
    for (const jobEntry of Object.entries(example.jobs)) {
      const [jobKey, job] = jobEntry;

      const steps = [];
      for (const step of job.steps) {
        if (step.name && step.name.toLowerCase() === 'checkout pr branch') {
          steps.push(
            step,
            {
              name: 'Replace m5 instance',
              run: `find examples -type f  -name '*.tf' -o -name '*.hcl'  | xargs sed -i 's/m5\.4xlarge/m5\.8xlarge/g'`
            },
            {
              name: 'Replace t2 instance',
              run: `find examples -type f  -name '*.tf' -o -name '*.hcl'  | xargs sed -i 's/t2\.micro/t2\.medium/g'`
            }
          )

          continue;
        }

        if (step.name && step.name.toLowerCase() === 'post infracost comment') {
          const goldenFilePath = `./testdata/${jobKey}_comment_golden.md`;
          const commentArgs = step.run
            .replace(/\\/g, '')
            .replace(/--pull-request \$\{\{github\.event\.pull_request\.number\}\}/g, '--pull-request 1')
            .split('\n')
            .map(s => s.trim())
            .filter(e => !e.startsWith('#') && e !== '')

          commentArgs.push('--dry-run true', '> /tmp/infracost_comment.md')
          step.run = commentArgs.join(' \\\n');

          steps.push(
            step,
            {
              run: `diff -y ${goldenFilePath} /tmp/infracost_comment.md`,
              name: 'Check the comment',
              if: `env.UPDATE_GOLDEN_FILES != 'true'`,
            },
            {
              name: 'Update the golden comment file',
              run: `cp /tmp/infracost_comment.md ${goldenFilePath}`,
              if: `env.UPDATE_GOLDEN_FILES == 'true'`,
            }
          );

          continue;
        }

        if (step.uses && step.uses.startsWith('slackapi/slack-github-action')) {
          // Assume this path for now. If we add our own Slack action we can get this easier from an input
          const path = '/tmp/infracost.json';
          const goldenFilePath = `./testdata/${jobKey}_slack_message_golden.json`;

          steps.push(
            {
              name: 'Generate Slack message',
              run: `infracost output --path=${path} --format=slack-message --show-skipped --out-file=/tmp/infracost_slack_message.json`,
            },
            {
              name: 'Check the Slack message',
              run: `diff -y <(jq --sort-keys . ${goldenFilePath}) <(jq --sort-keys . /tmp/infracost_slack_message.json)`,
              if: `env.UPDATE_GOLDEN_FILES != 'true'`,
            },
            {
              name: 'Update the golden Slack message file',
              run: `jq --sort-keys . /tmp/infracost_slack_message.json > ${goldenFilePath}`,
              if: `env.UPDATE_GOLDEN_FILES == 'true'`,
            }
          );

          continue;
        }

        // Replace infracost/actions steps with the local path
        steps.push({
          ...step,
          uses:
            step.uses &&
            step.uses.replace(/infracost\/actions\/(\w+)(@\w+)?/, './$1'),
        });
      }

      job.steps = steps;

      if (localSkipJobs.includes(jobKey)) {
        job.if = 'github.actor != \'nektos/act\'';
      }
    }
  }

  return examples;
}

// Generate the workflow YAML from the examples
function generateWorkflow(examples) {
  const workflow = {...workflowTemplate};

  for (const example of examples) {
    workflow.jobs = {
      ...workflow.jobs,
      ...example.jobs,
    };
  }

  return workflow;
}

// Write the generated workflow to a file
function writeWorkflow(workflow, target) {
  try {
    fs.writeFileSync(target, yaml.dump(workflow));
  } catch (err) {
    console.error(`Error writing YAML file: ${err}`);
  }
}

let examples = extractAllExamples(examplesDir);
examples = fixupExamples(examples);
const workflow = generateWorkflow(examples);
writeWorkflow(workflow, examplesTestWorkflowPath);

console.log('DONE');
