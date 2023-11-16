// TODO when we publish package we will import latest stable version and not using relative path
import { LavaSDK } from "../src/sdk/sdk";

/*
  Demonstrates how to use LavaSDK to send jsonRPC calls to the Ethereum Mainnet.

  You can find a list with all supported chains (https://github.com/lavanet/lava-sdk/blob/main/supportedChains.json)
  
  Lava SDK supports only rpc calls with positional parameters
  {"jsonrpc": "2.0", "method": "block", "params": ["23"], "id": 1}
  But not rpc calls with named parameters
  {"jsonrpc": "2.0", "method": "subtract", "params": {"subtrahend": 23, "minuend": 42}, "id": 3}
*/

let lavaSDKInstance: LavaSDK;

async function getLatestBlock(): Promise<string> {
  // Get latest block number
  const blockNumberResponse = await lavaSDKInstance.sendRelay({
    method: "eth_blockNumber",
    params: [],
  });

  // Parse and extract response
  const parsedResponse = blockNumberResponse;

  // Extract latest block number
  const latestBlockNumber = parsedResponse.result;

  // Get latest block
  const latestBlock = await lavaSDKInstance.sendRelay({
    method: "eth_getBlockByNumber",
    params: [latestBlockNumber, true],
  });

  return latestBlock;
}

async function getBatch(): Promise<Array<any>> {
  // Get abci_info and status
  return await lavaSDKInstance.sendRelay({
    relays: [
      {
        method: "abci_info",
        params: [],
      },
      {
        method: "status",
        params: [],
      },
    ],
  });
}

(async function () {
  try {
    // Create dAccess for Ethereum Mainnet
    // Default rpcInterface for Ethereum Mainnet is jsonRPC
    lavaSDKInstance = await LavaSDK.create({
      // badge data of an active badge server
      badge: {
        badgeServerAddress: "<badge server address>",
        projectId: "<badge project id>",
      },

      // chainID for Ethereum mainnet
      chainIds: "ETH1",

      // geolocation 1 for North america - geolocation 2 for Europe providers
      // default value is 1
      geolocation: "<geolocation goes here>",
    });

    console.log("Sending single request:");
    const latestBlock = await getLatestBlock();
    console.log("Latest block:", latestBlock);

    console.log("\n");

    console.log("Sending batch request:");
    const batchResponse = await getBatch();
    console.log("Batch response:", JSON.stringify(batchResponse, null, 2));
    process.exit(0);
  } catch (error) {
    console.error("Error getting latest block:", error);
    process.exit(1);
  }
})();
