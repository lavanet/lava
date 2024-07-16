import { lavanet } from "../../../../ecosystem/lavajs/dist/codegen/lavanet/bundle";
import { getSigningLavanetClient, } from "../../../../ecosystem/lavajs/dist/codegen/lavanet/client";
import { DirectSecp256k1HdWallet, DirectSecp256k1Wallet } from '@cosmjs/proto-signing'
import { fromHex } from "@cosmjs/encoding";

const publicRpc = "http://127.0.0.1:26657"
const privKey =  process.env.PRIVATE_KEY

// console.log("[lavajs_test] sending queries for address", process.env.PUBLIC_KEY)

// sending queries specifically to lava
async function main() {
  // let wallet = await DirectSecp256k1HdWallet.fromMnemonic("cool bag river filter labor develop census harbor deliver save idea draft flag rug month prize emotion february north report humor duty pond quarter", {prefix: "lava@"})
  let wallet = await DirectSecp256k1Wallet.fromKey(Buffer.from(privKey, "hex"), "lava@")
  const [firstAccount] = await wallet.getAccounts();
  let signingClient = await getSigningLavanetClient({
    rpcEndpoint: publicRpc,
    signer: wallet,
    defaultTypes: []
  })

  console.log("[lavajs_tx] buying subscription to", firstAccount)

  const msg = lavanet.lava.subscription.MessageComposer.withTypeUrl.buy({
    creator: firstAccount.address,
    consumer: firstAccount.address,
    index: "DefaultPlan",
    duration: BigInt(1), /* in months */
    autoRenewal: false, advancePurchase: false
  })

  const fee = {
    amount: [{ amount: "1", denom: "ulava" }], // Replace with the desired fee amount and token denomination
    gas: "50000000", // Replace with the desired gas limit
  }

  let res = await signingClient.signAndBroadcast(firstAccount.address, [msg], fee, "Buying subscription on Lava blockchain!")
  if (res.code != 0) {
    console.error("ERR [lavajs_tx]", res)
    throw new Error("ERR [lavajs_tx] Failed buying subscription")
  }
  console.log("[lavajs_tx] Successfully bought default subscription", res)
}

(async () => {
  try {
      await main();
  } catch (error) {
      console.error(" ERR [lavajs_tx] "+error.message);
      process.exit(1);
  }
})();