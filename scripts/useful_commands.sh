
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