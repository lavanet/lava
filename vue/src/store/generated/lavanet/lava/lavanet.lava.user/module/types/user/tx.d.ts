import { Reader, Writer } from "protobufjs/minimal";
import { SpecName } from "../user/spec_name";
import { Coin } from "../cosmos/base/v1beta1/coin";
import { BlockNum } from "../user/block_num";
export declare const protobufPackage = "lavanet.lava.user";
export interface MsgStakeUser {
    creator: string;
    spec: SpecName | undefined;
    amount: Coin | undefined;
    deadline: BlockNum | undefined;
}
export interface MsgStakeUserResponse {
}
export interface MsgUnstakeUser {
    creator: string;
    spec: SpecName | undefined;
    deadline: BlockNum | undefined;
}
export interface MsgUnstakeUserResponse {
}
export declare const MsgStakeUser: {
    encode(message: MsgStakeUser, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgStakeUser;
    fromJSON(object: any): MsgStakeUser;
    toJSON(message: MsgStakeUser): unknown;
    fromPartial(object: DeepPartial<MsgStakeUser>): MsgStakeUser;
};
export declare const MsgStakeUserResponse: {
    encode(_: MsgStakeUserResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgStakeUserResponse;
    fromJSON(_: any): MsgStakeUserResponse;
    toJSON(_: MsgStakeUserResponse): unknown;
    fromPartial(_: DeepPartial<MsgStakeUserResponse>): MsgStakeUserResponse;
};
export declare const MsgUnstakeUser: {
    encode(message: MsgUnstakeUser, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgUnstakeUser;
    fromJSON(object: any): MsgUnstakeUser;
    toJSON(message: MsgUnstakeUser): unknown;
    fromPartial(object: DeepPartial<MsgUnstakeUser>): MsgUnstakeUser;
};
export declare const MsgUnstakeUserResponse: {
    encode(_: MsgUnstakeUserResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgUnstakeUserResponse;
    fromJSON(_: any): MsgUnstakeUserResponse;
    toJSON(_: MsgUnstakeUserResponse): unknown;
    fromPartial(_: DeepPartial<MsgUnstakeUserResponse>): MsgUnstakeUserResponse;
};
/** Msg defines the Msg service. */
export interface Msg {
    StakeUser(request: MsgStakeUser): Promise<MsgStakeUserResponse>;
    /** this line is used by starport scaffolding # proto/tx/rpc */
    UnstakeUser(request: MsgUnstakeUser): Promise<MsgUnstakeUserResponse>;
}
export declare class MsgClientImpl implements Msg {
    private readonly rpc;
    constructor(rpc: Rpc);
    StakeUser(request: MsgStakeUser): Promise<MsgStakeUserResponse>;
    UnstakeUser(request: MsgUnstakeUser): Promise<MsgUnstakeUserResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
