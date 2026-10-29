# nvm installs Node.js for you only, in ~/.nvm; no root needed.
command -v curl >/dev/null 2>&1 || pkg curl
curl -fsSL https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
export NVM_DIR="$HOME/.nvm"
. "$NVM_DIR/nvm.sh"
nvm install --lts
