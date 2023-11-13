// TODO when we publish package we will import latest stable version and not using relative path
import { LavaSDK } from "../src/sdk/sdk";

/*
  Demonstrates how to use LavaSDK to send rest API calls to the Lava Network.

  You can find a list with all supported chains (https://github.com/lavanet/lava-sdk/blob/main/supportedChains.json)
*/
async function getLatestBlockAndValidators(): Promise<[string, string]> {
  // Create dAccess for Lava Network
  // Default rpcInterface for Lava Network is tendermintRPC
  // If you want to use rest it needs to be explicitly defined
  const lavaSDKInstance = await LavaSDK.create({
    // badge data of an active badge server
    badge: {
      badgeServerAddress: "<badge server address>",
      projectId: "<badge project id>",
    },

    // chainID for Cosmos Hub
    chainIds: "LAV1",

    // geolocation 1 for North america - geolocation 2 for Europe providers
    // default value is 1
    geolocation: "<geolocation goes here>",

    // rpcInterface default is tendermintrpc / jsonrpc for respective chains.
    // in this example we want to test rest so we need to specify it
  });

  // Get latest block
  const latestBlock = await lavaSDKInstance.sendRelay({
    connectionType: "GET",
    url: "/cosmos/base/tendermint/v1beta1/node_info",
  });

  // Get latest validator-set
  const validators = await lavaSDKInstance.sendRelay({
    connectionType: "GET",
    url: "/cosmos/base/tendermint/v1beta1/validatorsets/latest?pagination.count_total=true&pagination.reverse=true",
  });

  return [latestBlock, validators];
}

(async function () {
  try {
    const [latestBlock, validators] = await getLatestBlockAndValidators();
    console.log("Latest block:", latestBlock);
    console.log("Latest validators:", validators);
    process.exit(0);
  } catch (error) {
    console.error("Error getting latest block:", error);
    process.exit(1);
  }
})();
