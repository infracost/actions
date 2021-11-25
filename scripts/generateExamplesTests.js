#!/usr/bin/env node

// This file generates a GitHub action to test the examples by extracting the
// examples from each README file, modifying them slightly and then writing
// them to a GitHub action.

const fs = require('fs');
const yaml = require('js-yaml');

const examplesTestWorkflowPath = './.github/workflows/examples_test.yml';
const examplesDir = 'examples';
const exampleRegex =
  /\[\/\/\]: <> \(BEGIN EXAMPLE\)\n```.*\n((.|\n)*?)```\n\[\/\/\]: <> \(END EXAMPLE\)/gm;

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
        if (step.uses && step.uses.startsWith('infracost/actions/comment')) {
          const path = step.with.path;
          const goldenFilePath = `./testdata/${jobKey}_comment_golden.md`;

          steps.push(
            {
              name: 'Generate Infracost comment',
              run: `infracost output --path=${path} --format=github-comment --out-file=/tmp/infracost_comment.md`,
            },
            {
              name: 'Check the comment',
              run: `diff /tmp/infracost_comment.md ${goldenFilePath}`,
              if: `env.UPDATE_GOLDEN_FILES != 'true'`,
            },
            {
              name: 'Update the golden comment file',
              run: `cp /tmp/infracost_comment.md ${goldenFilePath}`,
              if: `env.UPDATE_GOLDEN_FILES == 'true'`,
            }
          );
        } else {
          // Replace infracost/actions steps with the local path
          steps.push({
            ...step,
            uses:
              step.uses &&
              step.uses.replace(/infracost\/actions\/(\w+)(@\w+)?/, './$1'),
          });
        }
      }

      job.steps = steps;
    }
  }

  return examples;
}

// Generate the workflow YAML from the examples
function generateWorkflow(examples) {
  const workflow = { ...workflowTemplate };

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
