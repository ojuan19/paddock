// Downloads the matching paddock binary from GitHub Releases and places it at bin/paddock.
// Zero npm deps — uses node stdlib + the host's `tar` for extraction.
// Archive naming MUST match .goreleaser.yaml's name_template:
//   paddock_{OS}_{ARCH}.tar.gz   where OS is Darwin|Linux (capitalized) and ARCH is amd64|arm64
//
// BINARY_VERSION is the Go release tag to download, NOT this npm package's version.
// Decoupled so we can ship npm-wrapper-only patches without cutting a new Go release.
// Bump this when shipping a new Go binary.
const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

const BINARY_VERSION = '0.1.0';
const REPO = 'ojuan19/paddock';

const OS_MAP = { darwin: 'Darwin', linux: 'Linux' };
const ARCH_MAP = { x64: 'amd64', arm64: 'arm64' };

const OS_NAME = OS_MAP[process.platform];
const ARCH = ARCH_MAP[process.arch];

if (!OS_NAME || !ARCH) {
  console.error(`paddockcli: unsupported platform ${process.platform}/${process.arch}`);
  process.exit(0);
}

const ASSET = `paddock_${OS_NAME}_${ARCH}.tar.gz`;
const URL = `https://github.com/${REPO}/releases/download/v${BINARY_VERSION}/${ASSET}`;
const BIN_DIR = path.join(__dirname, 'bin');
const FINAL_BIN = path.join(BIN_DIR, 'paddock');

fs.mkdirSync(BIN_DIR, { recursive: true });

// Extract into a sibling temp dir on the same filesystem so the final rename is atomic.
const TMP_DIR = fs.mkdtempSync(path.join(BIN_DIR, '.tmp-'));
const TGZ_PATH = path.join(TMP_DIR, ASSET);

function download(url, dest, redirectsLeft = 5) {
  return new Promise((resolve, reject) => {
    https.get(url, (res) => {
      if ([301, 302, 303, 307, 308].includes(res.statusCode)) {
        if (redirectsLeft === 0) return reject(new Error('too many redirects'));
        return resolve(download(res.headers.location, dest, redirectsLeft - 1));
      }
      if (res.statusCode !== 200) return reject(new Error(`HTTP ${res.statusCode} for ${url}`));
      const file = fs.createWriteStream(dest);
      res.pipe(file);
      file.on('finish', () => file.close(resolve));
      file.on('error', reject);
    }).on('error', reject);
  });
}

(async () => {
  try {
    console.log(`paddockcli: downloading ${ASSET} from ${URL}`);
    await download(URL, TGZ_PATH);
    execSync(`tar -xzf "${TGZ_PATH}" -C "${TMP_DIR}"`);
    const extractedBin = path.join(TMP_DIR, 'paddock');
    fs.chmodSync(extractedBin, 0o755);
    // Atomic swap: rename over the placeholder. The bin symlink keeps resolving.
    fs.renameSync(extractedBin, FINAL_BIN);
    fs.rmSync(TMP_DIR, { recursive: true, force: true });
    console.log(`paddockcli: installed paddock v${BINARY_VERSION}`);
  } catch (err) {
    fs.rmSync(TMP_DIR, { recursive: true, force: true });
    console.error(`paddockcli: install failed — ${err.message}`);
    console.error(`paddockcli: the 'paddock' command will print an install-failure message until you reinstall.`);
    // Exit 0: the placeholder remains and self-reports on invocation. Avoids
    // breaking parent automation (CI, monorepo bootstraps) for a recoverable failure.
    process.exit(0);
  }
})();
