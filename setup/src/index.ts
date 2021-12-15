import * as path from 'path';
import * as core from '@actions/core';
import * as tc from '@actions/tool-cache';
import * as io from '@actions/io';
import * as os from 'os';
import * as semver from 'semver';
import * as exec from '@actions/exec';
import * as github from '@actions/github';

// arch in [arm, x32, x64...] (https://nodejs.org/api/os.html#os_os_arch)
// return value in [amd64, 386, arm]
function mapArch(arch: string): string {
  const mappings: { [s: string]: string } = {
    x64: 'amd64',
  };
  return mappings[arch] || arch;
}

// os in [darwin, linux, win32...] (https://nodejs.org/api/os.html#os_os_platform)
// return value in [darwin, linux, windows]
function mapOS(os: string): string {
  const mappings: { [s: string]: string } = {
    win32: 'windows',
  };
  return mappings[os] || os;
}

function getDownloadObject(version: string): {
  url: string;
  binaryName: string;
} {
  let path = `releases/download/v${version}`;
  if (version === 'latest') {
    path = `releases/latest/download`;
  }

  const platform = os.platform();
  const filename = `infracost-${mapOS(platform)}-${mapArch(os.arch())}`;
  const binaryName = platform === 'win32' ? 'infracost.exe' : filename;
  const url = `https://github.com/infracost/infracost/${path}/${filename}.tar.gz`;
  return {
    url,
    binaryName,
  };
}

// Rename infracost-<platform>-<arch> to infracost
async function renameBinary(
  pathToCLI: string,
  binaryName: string
): Promise<void> {
  if (!binaryName.endsWith('.exe')) {
    const source = path.join(pathToCLI, binaryName);
    const target = path.join(pathToCLI, 'infracost');
    core.debug(`Moving ${source} to ${target}.`);
    try {
      await io.mv(source, target);
    } catch (e) {
      core.error(`Unable to move ${source} to ${target}.`);
      throw e;
    }
  }
}

async function getVersion(): Promise<string> {
  const version = core.getInput('version');
  if (semver.valid(version)) {
    return semver.clean(version) || version;
  }

  if (semver.validRange(version)) {
    const max = semver.maxSatisfying(await getAllVersions(), version);
    if (max) {
      return semver.clean(max) || version;
    }
    core.warning(`${version} did not match any release version.`);
  } else {
    core.warning(`${version} is not a valid version or range.`);
  }
  return version;
}

async function getAllVersions(): Promise<string[]> {
  const githubToken = core.getInput('github-token', { required: true });
  const octokit = github.getOctokit(githubToken);

  const allVersions: string[] = [];
  for await (const response of octokit.paginate.iterator(
    octokit.rest.repos.listReleases,
    { owner: 'infracost', repo: 'infracost' }
  )) {
    for (const release of response.data) {
      if (release.name) {
        allVersions.push(release.name);
      }
    }
  }

  return allVersions;
}

function exportEnvVars(): void {
  core.exportVariable('INFRACOST_PARALLELISM', 1) // TODO: remove this once we have fixed race conditions https://github.com/infracost/infracost/issues/1202
  core.exportVariable('INFRACOST_GITHUB_ACTION', true);
  core.exportVariable('INFRACOST_SKIP_UPDATE_CHECK', true);

  const repoUrl =
    github.context.payload.repository?.html_url ||
    `${github.context.serverUrl}/${github.context.repo.owner}/${github.context.repo.repo}`;
  const pullRequestUrl = github.context.payload.pull_request?.html_url;

  if (repoUrl) {
    core.exportVariable('INFRACOST_VCS_REPOSITORY_URL', repoUrl);
  }

  if (pullRequestUrl) {
    core.exportVariable('INFRACOST_VCS_PULL_REQUEST_URL', pullRequestUrl);
  }

  core.exportVariable('INFRACOST_LOG_LEVEL', core.isDebug() ? 'debug' : 'info');
}

async function setup(): Promise<void> {
  try {
    // Set Infracost environment variables
    exportEnvVars();

    // Get version of tool to be installed
    const version = await getVersion();

    // Download the specific version of the tool, e.g. as a tarball/zipball
    const download = getDownloadObject(version);
    const pathToTarball = await tc.downloadTool(download.url);

    // Extract the tarball onto host runner
    const pathToCLI = await tc.extractTar(pathToTarball);

    // Rename the platform/architecture specific binary to 'infracost'
    await renameBinary(pathToCLI, download.binaryName);

    // Expose the tool by adding it to the PATH
    core.addPath(pathToCLI);

    core.notice(`Setup Infracost CLI version ${version}`);

    // Set configure options
    const apiKey = core.getInput('api-key');
    if (apiKey) {
      const returnCode = await exec.exec('infracost', [
        'configure',
        'set',
        'api_key',
        apiKey,
      ]);
      if (returnCode !== 0) {
        throw new Error(
          `Error running infracost configure set api_key: ${returnCode}`
        );
      }
    }

    const currency = core.getInput('currency');
    if (currency) {
      const returnCode = await exec.exec('infracost', [
        'configure',
        'set',
        'currency',
        currency,
      ]);
      if (returnCode !== 0) {
        throw new Error(
          `Error running infracost configure set currency: ${returnCode}`
        );
      }
    }

    const pricingApiEndpoint = core.getInput('pricing-api-endpoint');
    if (pricingApiEndpoint) {
      const returnCode = await exec.exec('infracost', [
        'configure',
        'set',
        'pricing_api_endpoint',
        pricingApiEndpoint,
      ]);
      if (returnCode !== 0) {
        throw new Error(
          `Error running infracost configure set pricing_api_endpoint: ${returnCode}`
        );
      }
    }
  } catch (e) {
    core.setFailed(e as string | Error);
  }
}

if (require.main === module) {
  // eslint-disable-next-line no-void
  void setup();
}
