# RPCCONSUMER
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
