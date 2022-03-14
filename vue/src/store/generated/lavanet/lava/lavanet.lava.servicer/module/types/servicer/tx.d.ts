import { Reader, Writer } from "protobufjs/minimal";
import { SpecName } from "../servicer/spec_name";
import { Coin } from "../cosmos/base/v1beta1/coin";
import { BlockNum } from "../servicer/block_num";
import { SessionID } from "../servicer/session_id";
import { ClientRequest } from "../servicer/client_request";
import { WorkProof } from "../servicer/work_proof";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface MsgStakeServicer {
    creator: string;
    spec: SpecName | undefined;
    amount: Coin | undefined;
    deadline: BlockNum | undefined;
}
export interface MsgStakeServicerResponse {
}
export interface MsgUnstakeServicer {
    creator: string;
    spec: SpecName | undefined;
    deadline: BlockNum | undefined;
}
export interface MsgUnstakeServicerResponse {
}
export interface MsgProofOfWork {
    creator: string;
    spec: SpecName | undefined;
    session: SessionID | undefined;
    clientRequest: ClientRequest | undefined;
    workProof: WorkProof | undefined;
    computeUnits: number;
    blockOfWork: BlockNum | undefined;
}
export interface MsgProofOfWorkResponse {
}
export declare const MsgStakeServicer: {
    encode(message: MsgStakeServicer, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgStakeServicer;
    fromJSON(object: any): MsgStakeServicer;
    toJSON(message: MsgStakeServicer): unknown;
    fromPartial(object: DeepPartial<MsgStakeServicer>): MsgStakeServicer;
};
export declare const MsgStakeServicerResponse: {
    encode(_: MsgStakeServicerResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgStakeServicerResponse;
    fromJSON(_: any): MsgStakeServicerResponse;
    toJSON(_: MsgStakeServicerResponse): unknown;
    fromPartial(_: DeepPartial<MsgStakeServicerResponse>): MsgStakeServicerResponse;
};
export declare const MsgUnstakeServicer: {
    encode(message: MsgUnstakeServicer, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgUnstakeServicer;
    fromJSON(object: any): MsgUnstakeServicer;
    toJSON(message: MsgUnstakeServicer): unknown;
    fromPartial(object: DeepPartial<MsgUnstakeServicer>): MsgUnstakeServicer;
};
export declare const MsgUnstakeServicerResponse: {
    encode(_: MsgUnstakeServicerResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgUnstakeServicerResponse;
    fromJSON(_: any): MsgUnstakeServicerResponse;
    toJSON(_: MsgUnstakeServicerResponse): unknown;
    fromPartial(_: DeepPartial<MsgUnstakeServicerResponse>): MsgUnstakeServicerResponse;
};
export declare const MsgProofOfWork: {
    encode(message: MsgProofOfWork, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgProofOfWork;
    fromJSON(object: any): MsgProofOfWork;
    toJSON(message: MsgProofOfWork): unknown;
    fromPartial(object: DeepPartial<MsgProofOfWork>): MsgProofOfWork;
};
export declare const MsgProofOfWorkResponse: {
    encode(_: MsgProofOfWorkResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgProofOfWorkResponse;
    fromJSON(_: any): MsgProofOfWorkResponse;
    toJSON(_: MsgProofOfWorkResponse): unknown;
    fromPartial(_: DeepPartial<MsgProofOfWorkResponse>): MsgProofOfWorkResponse;
};
/** Msg defines the Msg service. */
export interface Msg {
    StakeServicer(request: MsgStakeServicer): Promise<MsgStakeServicerResponse>;
    UnstakeServicer(request: MsgUnstakeServicer): Promise<MsgUnstakeServicerResponse>;
    /** this line is used by starport scaffolding # proto/tx/rpc */
    ProofOfWork(request: MsgProofOfWork): Promise<MsgProofOfWorkResponse>;
}
export declare class MsgClientImpl implements Msg {
    private readonly rpc;
    constructor(rpc: Rpc);
    StakeServicer(request: MsgStakeServicer): Promise<MsgStakeServicerResponse>;
    UnstakeServicer(request: MsgUnstakeServicer): Promise<MsgUnstakeServicerResponse>;
    ProofOfWork(request: MsgProofOfWork): Promise<MsgProofOfWorkResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
