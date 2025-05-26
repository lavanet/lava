# Smart Router

The Smart Router supports any RPC (Lava, Alchemy, self-hosted etc.) and chooses the best nodes for you with automatic failover, error recovery, smart caching, fast TXs and more.

Being one of the core components in the Lava stack, it is already used and trusted by teams like Fireblocks, Movement, Arbiturm, NEAR, Fileocin, Cosmos and many more.

The Smart Router:
1) Routes requests to the best nodes based on reliability, speed, and sync
2) Automatically retries and fallbacks when providers face errors or downtime
3) Delivers faster transaction propagation by broadcasting to all providers at once
4) Has two layers of smart caching
6) Samples and checks data accuracy

## Usage
1. Clone the repository
2. `cd` into the repository folder
3. Run `make install-all`
4. Create a configuration file with the following format:

```
endpoints:
  - network-address: <network-address>
    chain-id: <chain-id>
    api-interface: <api-interface>
  - network-address: <network-address>
    chain-id: <chain-id>
    api-interface: <api-interface>
```
The `network-address` specifies the IP address and port number of the node, `chain-id` specifies the unique identifier of the blockchain, and `api-interface` specifies the API interface used by the node.

5. Start the consumer using the command `rpcconsumer --config <path/to/config/file>`
