# Multiple providers load balance via Nginx

This example demonstrates how to deploy multiple providers behind a single proxy with only one stake command.

## Setup Instructions

1. Run the following command to setup the compose:

   ```shell
   docker compose -f docker/load-balancing/docker-compose.yml up -d
   ```

2. That's it! You should have a running Lava node ready to use. To test the setup run the following command:

   ```shell
   curl -X POST -H "Content-Type: application/json" localhost:26657 --data '{"jsonrpc": "2.0", "id": 1, "method": "status", "params": []}'
   ```

   The result will look something like this:

    ```json
    {
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "node_info": {
        "protocol_version": {
            "p2p": "8",
            "block": "11",
            "app": "0"
        },
        "id": "a803429fe79d72656bd07e0d290ec048c0615037",
        "listen_addr": "tcp://0.0.0.0:26656",
        "network": "lava",
        "version": "0.37.4",
        "channels": "40202122233038606100",
        "moniker": "validator",
        "other": {
            "tx_index": "on",
            "rpc_address": "tcp://0.0.0.0:26657"
        }
        },
        "sync_info": {
        "latest_block_hash": "B24647F6A2578D361F50796B9FBD936DADC589B6AD56F4F34B1E242184484199",
        "latest_app_hash": "1B9147BFB6548B64CCAAAD51C957C3A5C5FF2786F7518DE29AD416E038B52397",
        "latest_block_height": "791",
        "latest_block_time": "2024-07-28T12:14:01.905074847Z",
        "earliest_block_hash": "C8851AF02B28E00491BED38EDA5E719317BCA31433A7C8204FC16318571CA754",
        "earliest_app_hash": "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
        "earliest_block_height": "1",
        "earliest_block_time": "2024-07-28T12:00:16.125804428Z",
        "catching_up": false
        },
        "validator_info": {
        "address": "430C6891B7C10939270144206B05ABF4D41CF04B",
        "pub_key": {
            "type": "tendermint/PubKeyEd25519",
            "value": "y9msPfAEV27Gr3J4uiFJeOhnZnruH49BPCQK6Ib/Lq4="
        },
        "voting_power": "10000000"
        }
    }
    }
    ```

3. To clean up the setup simply run:

   ```shell
   docker compose -f docker/load-balancing/docker-compose.yml down -v
   ```
