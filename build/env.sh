#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
ubetcoindir="$workspace/src/github.com/ubetprotocol"
if [ ! -L "$ubetcoindir/go-ubetprotocol" ]; then
    mkdir -p "$ubetcoindir"
    cd "$ubetcoindir"
    ln -s ../../../../../. go-ubetprotocol
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$ubetcoindir/go-ubetprotocol"
PWD="$ubetcoindir/go-ubetprotocol"

# Launch the arguments with the configured environment.
exec "$@"
