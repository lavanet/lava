import { MsgSendEncodeObject, AminoMsgSend } from "@cosmjs/stargate";
import { cosmos } from "../../../../ecosystem/lavajs/dist/codegen/cosmos/bundle";
import { lavanet } from "../../../../ecosystem/lavajs/dist/codegen/lavanet/bundle";
import { getSigningLavanetClient, } from "../../../../ecosystem/lavajs/dist/codegen/lavanet/client";
import { DirectSecp256k1HdWallet, DirectSecp256k1Wallet } from '@cosmjs/proto-signing'
import { Secp256k1Wallet, encodeSecp256k1Signature, decodeSignature } from '@cosmjs/amino'
import { fromHex } from "@cosmjs/encoding";
import { makeMultisignedTxBytes } from "@cosmjs/stargate";
import { TxRaw } from "cosmjs-types/cosmos/tx/v1beta1/tx";
import { fromBase64 } from "@cosmjs/encoding";
import { Tx } from "cosmjs-types/cosmos/tx/v1beta1/tx"

const publicRpc = "http://127.0.0.1:26657"
const privKey =  process.env.PRIVATE_KEY

// sending queries specifically to lava
async function main() {
  // let wallet = await DirectSecp256k1HdWallet.fromMnemonic("cool bag river filter labor develop census harbor deliver save idea draft flag rug month prize emotion february north report humor duty pond quarter", {prefix: "lava@"})
  let wallet = await Secp256k1Wallet.fromKey(Buffer.from(privKey, "hex"), "lava@")
  const [firstAccount] = await wallet.getAccounts();
  let signingClient = await getSigningLavanetClient({
    rpcEndpoint: publicRpc,
    signer: wallet,
    defaultTypes: []
  })

  console.log("[lavajs_tx] buying subscription to", firstAccount)

  const msg = lavanet.lava.subscription.MessageComposer.fromPartial.buy({
    creator: firstAccount.address,
    consumer: firstAccount.address,
    index: "DefaultPlan",
    duration: BigInt(1), /* in months */
    autoRenewal: false, advancePurchase: false
  })

  const fee = {
    amount: [], // Replace with the desired fee amount and token denomination
    gas: "270422000", // Replace with the desired gas limit
  }
  

  let txRaw = await signingClient.sign(firstAccount.address, [msg], fee, "Buying subscription on Lava blockchain!")
  const txRawBytes = Uint8Array.from(TxRaw.encode(txRaw).finish());
  let result = await signingClient.broadcastTx(txRawBytes)
  console.log("tx, result", result)

}

(async () => {
  try {
      await main();
  } catch (error) {
      console.error(" ERR [lavajs_tx_amino_subscription] "+error.message);
      process.exit(1);
  }
})();