#!/bin/bash

function prepare() {
    echo "🔹make sure to run 'go mod tidy' from the lava repo before trying to run this file"
    
    use_sudo=$1
    if [ "$use_sudo" = true ]; then
        SUDO=sudo
    else
        SUDO=''
    fi

    file_path="../../go.mod"
    expected_lines=(
        "github.com/gogo/googleapis v1.4.1 // indirect"
        "github.com/cosmos/cosmos-sdk v0.47.13"
        "github.com/cosmos/gogoproto v1.4.10"
        "github.com/cosmos/cosmos-proto v1.0.0-beta.5"
    )

    missing_lines=()

    for line in "${expected_lines[@]}"; do
        if ! grep -qF "$line" "$file_path"; then
            missing_lines+=("$line")
        fi
    done

    if [[ ${#missing_lines[@]} -eq 0 ]]; then
        echo "✨ All expected lines are present in the $file_path file."
    else
        echo "Some or all expected lines are missing in the $file_path file."
        echo "Missing lines:"
        for missing_line in "${missing_lines[@]}"; do
            echo "🔹$missing_line"
        done
        exit 1
    fi


    if [[ -z "$GOPATH" ]]; then
        echo "Error: GOPATH is not set. Set the GOPATH environment variable to your Go workspace directory." >&2
        exit 1
    fi

    if [[ ! -d "$GOPATH" ]]; then
        echo "Error: The directory specified in GOPATH ('$GOPATH') does not exist." >&2
        exit 1
    fi

    specific_dir="$GOPATH/pkg/mod/github.com/lavanet/cosmos-sdk@v0.47.13-lava-cosmos"

    if [[ ! -d "$specific_dir" ]]; then
        echo "Error: The cosmos-sdk directory ('$specific_dir') does not exist under '$GOPATH/pkg/mod'." >&2
        echo "make sure you ran 'go mod tidy' in the lava main repo"
        exit 1
    fi

    gogodir="$GOPATH/pkg/mod/github.com/cosmos/gogoproto@v1.4.10"

    if [[ ! -d "$gogodir" ]]; then
        echo "Error: The gogodir directory ('$gogodir') does not exist under '$GOPATH/pkg/mod'." >&2
        echo "make sure you ran 'go mod tidy' in the lava main repo"
        exit 1
    fi

    googledir="$GOPATH/pkg/mod/github.com/gogo/googleapis@v1.4.1"

    if [[ ! -d "$googledir" ]]; then
        echo "Error: The googledir directory ('$googledir') does not exist under '$GOPATH/pkg/mod'." >&2
        echo "make sure you ran 'go mod tidy' in the lava main repo"
        exit 1
    fi

    cosmosprotosdir="$GOPATH/pkg/mod/github.com/cosmos/cosmos-proto@v1.0.0-beta.5"

    if [[ ! -d "$cosmosprotosdir" ]]; then
        echo "Error: The cosmosprotosdir directory ('$cosmosprotosdir') does not exist under '$GOPATH/pkg/mod'." >&2
        echo "make sure you ran 'go mod tidy' in the lava main repo"
        exit 1
    fi

    $SUDO rm -rf ./proto

    mkdir -p proto/cosmos/cosmos-sdk/google/api

    $SUDO rm -rf ./proto/cosmos/cosmos-sdk/cosmos; cp -r $specific_dir/proto/cosmos ./proto/cosmos/cosmos-sdk
    $SUDO rm -rf ./proto/cosmos/cosmos-sdk/amino; cp -r $specific_dir/proto/amino ./proto/cosmos/cosmos-sdk
    $SUDO rm -rf ./proto/cosmos/cosmos-sdk/tendermint; cp -r $specific_dir/proto/tendermint ./proto/cosmos/cosmos-sdk
    $SUDO rm -rf ./proto/cosmos/cosmos-sdk/gogoproto; cp -r $gogodir/gogoproto ./proto/cosmos/cosmos-sdk
    $SUDO rm -rf ./proto/cosmos/cosmos-sdk/google; cp -r $gogodir/protobuf/google ./proto/cosmos/cosmos-sdk
    $SUDO rm -rf ./proto/cosmos/cosmos-sdk/cosmos_proto; cp -r $cosmosprotosdir/proto/cosmos_proto ./proto/cosmos/cosmos-sdk
    $SUDO cp -r $googledir/google/api ./proto/cosmos/cosmos-sdk/google

    cp -r ../../proto/lavanet ./proto

    group=$(groups $(whoami) | cut -d' ' -f1)
    $SUDO chown -R $(whoami):$group ./proto
}