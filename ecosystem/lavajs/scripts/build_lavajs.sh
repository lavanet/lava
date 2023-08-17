#!/bin/bash


# Check if the 'telescope' binary is available
if ! command -v telescope &> /dev/null; then
    echo "Please install 'telescope' and try again."
    echo "npm install -g @cosmology/telescope"
    exit 1
fi

if ! command -v expect &> /dev/null; then
    echo "Please install 'expect' and try again."
    echo "sudo apt-get install expect"
    exit 1
fi

# Continue with other actions if 'telescope' is available
echo "Telescope is available."
# Add your other actions here


rm -rf ./proto/lavanet
cp -r ../../proto/lavanet ./proto/.

./scripts/transpile.sh

npm run build

echo "Script completed."