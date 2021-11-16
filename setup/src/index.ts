import path from 'path';
import * as core from '@actions/core';
import * as tc from '@actions/tool-cache';
import * as io from '@actions/io';
import os from 'os';
import * as exec from '@actions/exec';

// arch in [arm, x32, x64...] (https://nodejs.org/api/os.html#os_os_arch)
// return value in [amd64, 386, arm]
function mapArch(arch) {
  const mappings = {
    x64: 'amd64',
  };
  return mappings[arch] || arch;
}

// os in [darwin, linux, win32...] (https://nodejs.org/api/os.html#os_os_platform)
// return value in [darwin, linux, windows]
function mapOS(os) {
  const mappings = {
    win32: 'windows',
  };
  return mappings[os] || os;
}

function getDownloadObject(version): { url: string; binaryName: string } {
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
async function renameBinary(pathToCLI, binaryName) {
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

async function setup() {
  try {
    // Get version of tool to be installed
    const version = core.getInput('version');

    // Download the specific version of the tool, e.g. as a tarball/zipball
    const download = getDownloadObject(version);
    const pathToTarball = await tc.downloadTool(download.url);

    // Extract the tarball onto host runner
    const pathToCLI = await tc.extractTar(pathToTarball);

    // Rename the platform/architecture specific binary to 'infracost'
    await renameBinary(pathToCLI, download.binaryName);

    // Expose the tool by adding it to the PATH
    core.addPath(pathToCLI);

    // Set configure options
    const apiKey = core.getInput('api_key');
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

    const pricingApiEndpoint = core.getInput('pricing_api_endpoint');
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
  setup();
}
