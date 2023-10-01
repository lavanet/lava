// TODO when we publish package we will import latest stable version and not using relative path
import { LavaSDK } from "../src/sdk/sdk";

/*
  Demonstrates how to use LavaSDK to send tendermintRPC calls to the Cosmos Hub.

  You can find a list with all supported chains (https://github.com/lavanet/lava-sdk/blob/main/supportedChains.json)
  
  Lava SDK supports only rpc calls with positional parameters
  {"jsonrpc": "2.0", "method": "block", "params": ["23"], "id": 1}
  But not rpc calls with named parameters
  {"jsonrpc": "2.0", "method": "subtract", "params": {"subtrahend": 23, "minuend": 42}, "id": 3}
*/
async function getLatestBlock(): Promise<Array<any>> {
  // Create dAccess for Cosmos Hub
  // Default rpcInterface for Cosmos Hub is tendermintRPC
  const cosmosHub = await LavaSDK.create({
    // private key with an active subscription
    privateKey: "<lava consumer private key>",

    // chainID for Cosmos Hub
    chainIds: "LAV1",

    // geolocation 1 for North america - geolocation 2 for Europe providers
    // default value is 1
    geolocation: "2",
  });

  // Get abci_info

  const results = [];

  for (let i = 0; i < 10; i++) {
    const info = await cosmosHub.sendRelay({
      method: "abci_info",
      params: [],
    });

    // Parse and extract response
    const parsedInfo = info.result.response;

    // Extract latest block number
    const latestBlockNumber = parsedInfo.last_block_height;
    // Fetch latest block
    const latestBlock = await cosmosHub.sendRelay({
      method: "block",
      params: [latestBlockNumber],
    });
    results.push(latestBlock);
    console.log("Latest block:", latestBlock);
  }
  return results;
}

(async function () {
  try {
    const results = await getLatestBlock();
    console.log("results:", results);
    process.exit(0);
  } catch (error) {
    console.error("Error getting latest block:", error);
    process.exit(1);
  }
})();
