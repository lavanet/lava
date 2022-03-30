// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.

import { StdFee } from "@cosmjs/launchpad";
import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry, OfflineSigner, EncodeObject, DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgProofOfWork } from "./types/servicer/tx";
import { MsgStakeServicer } from "./types/servicer/tx";
import { MsgUnstakeServicer } from "./types/servicer/tx";


const types = [
  ["/lavanet.lava.servicer.MsgProofOfWork", MsgProofOfWork],
  ["/lavanet.lava.servicer.MsgStakeServicer", MsgStakeServicer],
  ["/lavanet.lava.servicer.MsgUnstakeServicer", MsgUnstakeServicer],
  
];
export const MissingWalletError = new Error("wallet is required");

export const registry = new Registry(<any>types);

const defaultFee = {
  amount: [],
  gas: "200000",
};

interface TxClientOptions {
  addr: string
}

interface SignAndBroadcastOptions {
  fee: StdFee,
  memo?: string
}

const txClient = async (wallet: OfflineSigner, { addr: addr }: TxClientOptions = { addr: "http://localhost:26657" }) => {
  if (!wallet) throw MissingWalletError;
  let client;
  if (addr) {
    client = await SigningStargateClient.connectWithSigner(addr, wallet, { registry });
  }else{
    client = await SigningStargateClient.offline( wallet, { registry });
  }
  const { address } = (await wallet.getAccounts())[0];

  return {
    signAndBroadcast: (msgs: EncodeObject[], { fee, memo }: SignAndBroadcastOptions = {fee: defaultFee, memo: ""}) => client.signAndBroadcast(address, msgs, fee,memo),
    msgProofOfWork: (data: MsgProofOfWork): EncodeObject => ({ typeUrl: "/lavanet.lava.servicer.MsgProofOfWork", value: MsgProofOfWork.fromPartial( data ) }),
    msgStakeServicer: (data: MsgStakeServicer): EncodeObject => ({ typeUrl: "/lavanet.lava.servicer.MsgStakeServicer", value: MsgStakeServicer.fromPartial( data ) }),
    msgUnstakeServicer: (data: MsgUnstakeServicer): EncodeObject => ({ typeUrl: "/lavanet.lava.servicer.MsgUnstakeServicer", value: MsgUnstakeServicer.fromPartial( data ) }),
    
  };
};

interface QueryClientOptions {
  addr: string
}

const queryClient = async ({ addr: addr }: QueryClientOptions = { addr: "http://localhost:1317" }) => {
  return new Api({ baseUrl: addr });
};

export {
  txClient,
  queryClient,
};
