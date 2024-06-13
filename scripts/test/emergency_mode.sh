#!/bin/bash

# Specify timeout for emergency mode
timeout="$1s"

# Specify the path and config file
path="$HOME/.lava/config/"
config='config.toml'

# Determine OS
os_name=$(uname)
case "$(uname)" in
  Darwin)
    SED_INLINE="-i ''" ;;
  Linux)
    SED_INLINE="-i" ;;
  *)
    echo "unknown system: $(uname)"
    exit 1 ;;
esac

# Edit config.toml file
sed $SED_INLINE -e "s/timeout_commit = \".*/timeout_commit = \"$timeout\"/" "$path$config"

# Start node
lavad start --pruning=nothing