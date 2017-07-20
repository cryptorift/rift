#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
riftdir="$workspace/src/github.com/cryptorift"
if [ ! -L "$riftdir/riftcore" ]; then
    mkdir -p "$riftdir"
    cd "$riftdir"
    ln -s ../../../../../. riftcore
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$riftdir/riftcore"
PWD="$riftdir/riftcore"

# Launch the arguments with the configured environment.
exec "$@"
