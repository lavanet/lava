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

  let lava_client = await lavanet.ClientFactory.createRPCQueryClient({rpcEndpoint: publicRpc})
  let validators = await lava_client.cosmos.staking.v1beta1.validators({status: "BOND_STATUS_BONDED"})
  console.log("validators: ", validators.validators)
  let validator_chosen = validators.validators[0]
  
  let providers = await lava_client.lavanet.lava.pairing.providers({chainID: "LAV1", showFrozen: true})
  console.log("providers: ", providers.stakeEntry)
  let provider_chosen = providers.stakeEntry[0]
  let provider_chosen2 = providers.stakeEntry[1]
  

  const msg = lavanet.lava.dualstaking.MessageComposer.withTypeUrl.delegate({
    amount: {amount: "100", denom: "ulava"}, 
    chainID: "LAV1", 
    creator: firstAccount.address,
    provider: provider_chosen.address,
    validator: validator_chosen.operatorAddress,
  })
  // console.log("[lavajs_tx] delegating msg", msg )


  const fee = {
    amount: [], // Replace with the desired fee amount and token denomination
    gas: "270422000", // Replace with the desired gas limit
  }
  

  let txRaw = await signingClient.sign(firstAccount.address, [msg], fee, "delegating!")
  const txRawBytes = Uint8Array.from(TxRaw.encode(txRaw).finish());
  let result = await signingClient.broadcastTx(txRawBytes)
  console.log("tx, result", result)

  // re-delegate
  const msg2 = lavanet.lava.dualstaking.MessageComposer.withTypeUrl.redelegate({
    amount: {amount: "50", denom: "ulava"}, 
    fromChainID: "LAV1", 
    toChainID: "LAV1", 
    creator: firstAccount.address,
    fromProvider: provider_chosen.address,
    toProvider: provider_chosen2.address,
  })
  console.log("[lavajs_tx] delegating msg", msg2 )


  let txRaw2 = await signingClient.sign(firstAccount.address, [msg2], fee, "redelegating!")
  const txRawBytes2 = Uint8Array.from(TxRaw.encode(txRaw2).finish());
  let result2 = await signingClient.broadcastTx(txRawBytes2)
  console.log("tx, result", result2)

}

(async () => {
  try {
      await main();
  } catch (error) {
      console.error(" ERR [lavajs_tx_amino_delegating] "+error.message);
      process.exit(1);
  }
})();