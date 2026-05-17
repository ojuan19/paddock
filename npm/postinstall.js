// Downloads the matching paddock binary from GitHub Releases and places it at bin/paddock.
// Zero npm deps — uses node stdlib + the host's `tar` for extraction.
// Archive naming MUST match .goreleaser.yaml's name_template:
//   paddock_{OS}_{ARCH}.tar.gz   where OS is Darwin|Linux (capitalized) and ARCH is amd64|arm64
const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

const VERSION = process.env.npm_package_version;
const REPO = 'ojuan19/paddock';

const OS_MAP = { darwin: 'Darwin', linux: 'Linux' };
const ARCH_MAP = { x64: 'amd64', arm64: 'arm64' };

const OS = OS_MAP[process.platform];
const ARCH = ARCH_MAP[process.arch];

if (!OS || !ARCH) {
  console.error(`paddockcli: unsupported platform ${process.platform}/${process.arch}`);
  process.exit(1);
}

const ASSET = `paddock_${OS}_${ARCH}.tar.gz`;
const URL = `https://github.com/${REPO}/releases/download/v${VERSION}/${ASSET}`;
const BIN_DIR = path.join(__dirname, 'bin');
const TGZ_PATH = path.join(BIN_DIR, ASSET);

fs.mkdirSync(BIN_DIR, { recursive: true });

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
    execSync(`tar -xzf "${TGZ_PATH}" -C "${BIN_DIR}"`);
    fs.unlinkSync(TGZ_PATH);
    fs.chmodSync(path.join(BIN_DIR, 'paddock'), 0o755);
    console.log(`paddockcli: installed paddock v${VERSION}`);
  } catch (err) {
    console.error(`paddockcli: install failed — ${err.message}`);
    process.exit(1);
  }
})();
