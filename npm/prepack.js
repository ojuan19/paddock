// Runs before `npm pack` / `npm publish`. Writes the placeholder bin/paddock
// into the tarball-staging tree so npm creates the bin symlink at install
// time. postinstall.js overwrites this placeholder with the real binary.
const fs = require('fs');
const path = require('path');

const BIN_DIR = path.join(__dirname, 'bin');
const BIN_PATH = path.join(BIN_DIR, 'paddock');

const PLACEHOLDER = `#!/bin/sh
echo "paddock: binary was not installed (postinstall did not run or failed)." >&2
echo "Reinstall with network access: npm install -g paddockcli --force" >&2
exit 1
`;

fs.mkdirSync(BIN_DIR, { recursive: true });
fs.writeFileSync(BIN_PATH, PLACEHOLDER, { mode: 0o755 });
console.log(`prepack: wrote placeholder ${BIN_PATH}`);
