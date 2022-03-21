// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.
import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgProofOfWork } from "./types/servicer/tx";
import { MsgUnstakeServicer } from "./types/servicer/tx";
import { MsgStakeServicer } from "./types/servicer/tx";
const types = [
    ["/lavanet.lava.servicer.MsgProofOfWork", MsgProofOfWork],
    ["/lavanet.lava.servicer.MsgUnstakeServicer", MsgUnstakeServicer],
    ["/lavanet.lava.servicer.MsgStakeServicer", MsgStakeServicer],
];
export const MissingWalletError = new Error("wallet is required");
export const registry = new Registry(types);
const defaultFee = {
    amount: [],
    gas: "200000",
};
const txClient = async (wallet, { addr: addr } = { addr: "http://localhost:26657" }) => {
    if (!wallet)
        throw MissingWalletError;
    let client;
    if (addr) {
        client = await SigningStargateClient.connectWithSigner(addr, wallet, { registry });
    }
    else {
        client = await SigningStargateClient.offline(wallet, { registry });
    }
    const { address } = (await wallet.getAccounts())[0];
    return {
        signAndBroadcast: (msgs, { fee, memo } = { fee: defaultFee, memo: "" }) => client.signAndBroadcast(address, msgs, fee, memo),
        msgProofOfWork: (data) => ({ typeUrl: "/lavanet.lava.servicer.MsgProofOfWork", value: MsgProofOfWork.fromPartial(data) }),
        msgUnstakeServicer: (data) => ({ typeUrl: "/lavanet.lava.servicer.MsgUnstakeServicer", value: MsgUnstakeServicer.fromPartial(data) }),
        msgStakeServicer: (data) => ({ typeUrl: "/lavanet.lava.servicer.MsgStakeServicer", value: MsgStakeServicer.fromPartial(data) }),
    };
};
const queryClient = async ({ addr: addr } = { addr: "http://localhost:1317" }) => {
    return new Api({ baseUrl: addr });
};
export { txClient, queryClient, };
