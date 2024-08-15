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

    

    echo "✨✨✨✨✨ GOPATH set to $GOPATH" >&2
    echo "✨✨✨✨✨ go env GOPATH: $(go env GOPATH)" >&2
    echo "✨✨✨✨✨ go env GOMODCACHE: $(go env GOMODCACHE)" >&2
    gopath=$(go env GOPATH)
    if [[ -z "$gopath" ]]; then
        echo "Error: GOPATH is not set. setting it to ~/go" >&2
    fi

    if [[ ! -d "$gopath" ]]; then
        echo "Error: The directory specified in GOPATH ('$gopath') does not exist." >&2
        exit 1
    fi

    echo "$(ls $gopath/pkg/mod/github.com/cosmos)"
    specific_dir="$gopath/pkg/mod/github.com/lavanet/cosmos-sdk@v0.47.13-lava-cosmos"

    if [[ ! -d "$specific_dir" ]]; then
        echo "Error: The cosmos-sdk directory ('$specific_dir') does not exist under '$gopath/pkg/mod'." >&2
        echo "make sure you ran 'go mod tidy' in the lava main repo"
        exit 1
    fi

    gogodir="$gopath/pkg/mod/github.com/cosmos/gogoproto@v1.4.10"

    if [[ ! -d "$gogodir" ]]; then
        echo "Error: The gogodir directory ('$gogodir') does not exist under '$gopath/pkg/mod'." >&2
        echo "make sure you ran 'go mod tidy' in the lava main repo"
        exit 1
    fi

    googledir="$gopath/pkg/mod/github.com/gogo/googleapis@v1.4.1"

    if [[ ! -d "$googledir" ]]; then
        echo "Error: The googledir directory ('$googledir') does not exist under '$gopath/pkg/mod'." >&2
        echo "make sure you ran 'go mod tidy' in the lava main repo"
        exit 1
    fi

    cosmosprotosdir="$gopath/pkg/mod/github.com/cosmos/cosmos-proto@v1.0.0-beta.5"

    if [[ ! -d "$cosmosprotosdir" ]]; then
        echo "Error: The cosmosprotosdir directory ('$cosmosprotosdir') does not exist under '$GOPATH/pkg/mod'." >&2
        echo "make sure you ran 'go mod tidy' in the lava main repo"
        exit 1
    fi

    $SUDO rm -rf ./proto/cosmos; cp -r $specific_dir/proto/cosmos ./proto
    $SUDO rm -rf ./proto/amino; cp -r $specific_dir/proto/amino ./proto
    $SUDO rm -rf ./proto/tendermint; cp -r $specific_dir/proto/tendermint ./proto
    $SUDO rm -rf ./proto/gogoproto; cp -r $gogodir/gogoproto ./proto
    $SUDO rm -rf ./proto/google; cp -r $gogodir/protobuf/google ./proto
    $SUDO rm -rf ./proto/cosmos_proto; cp -r $cosmosprotosdir/proto/cosmos_proto ./proto
    $SUDO mkdir ./proto/google/api
    $SUDO cp -r $googledir/google/api/annotations.proto ./proto/google/api/.
    $SUDO cp -r $googledir/google/api/http.proto ./proto/google/api/.

    $SUDO chown -R $(whoami):$(whoami) ./proto
}