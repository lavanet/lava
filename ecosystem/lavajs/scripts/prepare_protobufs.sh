#!/bin/bash

function prepare() {
    echo "🔹make sure to run 'go mod tidy' from the lava repo before trying to run this file"

    file_path="../../go.mod"
    expected_lines=(
        "github.com/gogo/googleapis v1.4.1 // indirect"
        "github.com/cosmos/cosmos-sdk v0.47.3"
        "github.com/cosmos/gogoproto v1.4.10"
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
    specific_dir="$gopath/pkg/mod/github.com/cosmos/cosmos-sdk@v0.47.3"

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

    sudo rm -rf ./proto/cosmos; cp -r $specific_dir/proto/cosmos ./proto
    sudo rm -rf ./proto/amino; cp -r $specific_dir/proto/amino ./proto
    sudo rm -rf ./proto/tendermint; cp -r $specific_dir/proto/tendermint ./proto
    sudo rm -rf ./proto/gogoproto; cp -r $gogodir/gogoproto ./proto
    sudo rm -rf ./proto/google; cp -r $gogodir/protobuf/google ./proto
    sudo mkdir ./proto/google/api
    sudo cp -r $googledir/google/api/annotations.proto ./proto/google/api/.
    sudo cp -r $googledir/google/api/http.proto ./proto/google/api/.

    sudo chown -R $(whoami):$(whoami) ./proto
}