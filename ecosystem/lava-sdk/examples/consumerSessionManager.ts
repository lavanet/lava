// TODO when we publish package we will import latest stable version and not using relative path
import { LavaSDK } from "../src/sdk/sdk";
import Relayer from "../src/relayer/relayer";
import { StateTracker } from "../src/stateTracker/state_tracker";
import { createWallet } from "../src/wallet/wallet";
import { ConsumerSessionManager } from "../src/lavasession/consumerSessionManager";
import { RandomProviderOptimizer } from "../src/lavasession/providerOptimizer";
import { RPCEndpoint } from "../src/lavasession/consumerTypes";

async function updatesAllProviders(): Promise<void> {
  const privateKey = "<lava consumer private key>";
  const lavaChainId = "LAV1";
  const chainId = "LAV1";
  const geolocation = "1";
  const relayer = new Relayer(chainId, privateKey, lavaChainId, false);

  const wallet = await createWallet(privateKey);
  const account = await wallet.getConsumerAccount();

  const rpcEndpoint = new RPCEndpoint(
    // TODO: get this from pairingList.json
    "127.0.0.1:2221",
    chainId,
    "apiInterface",
    Number(geolocation)
  );
  const consumerSessionManager = new ConsumerSessionManager(
    relayer,
    rpcEndpoint,
    new RandomProviderOptimizer()
  );
  const stateTracker = new StateTracker(
    "pairingList.json",
    relayer,
    [],
    {
      debug: true,
      geolocation: geolocation,
      network: "testnet",
      accountAddress: account.address,
    },
    new Map([["LAV1", [consumerSessionManager]]])
  );

  await new Promise((resolve) => setTimeout(resolve, 10000));
}

(async function () {
  try {
    await updatesAllProviders();
    process.exit(0);
  } catch (e) {
    console.error("Updating providers:");
  }
  // try {
  //   const latestBlock = await getLatestBlock();
  //   console.log("Latest block:", latestBlock);
  //   process.exit(0);
  // } catch (error) {
  //   console.error("Error getting latest block:", error);
  // }
})();
