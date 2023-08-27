
function sleep_until_next_epoch {
    epoch_start=$(lavad query epochstorage show-epoch-details | grep "startBlock: ")
    echo "Waiting for the next epoch for the changes to be active"
    loading_animation="/-\\|"
    while true; do
    epoch_now=$(lavad query epochstorage show-epoch-details | grep "startBlock: ")
    for (( i=0; i<${#loading_animation}; i++ )); do
        sleep 0.2
        echo -en "${loading_animation:$i:1}" "\r"
    done
    if [[ $epoch_start != $epoch_now ]] 
    then
        echo "finished waiting"
        break
    fi
    done
}

function wait_next_block {
  current=$( lavad q block | jq .block.header.height)
  echo "Waiting for next block $current"
  while true; do
    sleep 0.5
    new=$( lavad q block | jq .block.header.height)
    if [[ $current != $new ]]
    then
      echo "finished waiting for block $new"
        break
    fi
  done
}

function wait_count_blocks {
  for i in $(seq 1 $1); do
    wait_next_block
  done
}

# Function to check if a command is available
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

latest_vote() {
  # Check if jq is not installed
  if ! command_exists yq; then
      echo "yq not found. Please install yq using the init_install.sh script or manually."
      exit 1
  fi
  lavad q gov proposals 2> /dev/null | yq eval '.proposals[].id'  | wc -l
}