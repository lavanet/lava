import { Reader, Writer } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface RelayRequest {
    specId: number;
    apiId: number;
    sessionId: number;
    /** total compute unit used including this relay */
    cuSum: number;
    data: Uint8Array;
    sig: Uint8Array;
    servicer: string;
    blockHeight: number;
}
export interface RelayReply {
    data: Uint8Array;
    sig: Uint8Array;
}
export declare const RelayRequest: {
    encode(message: RelayRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): RelayRequest;
    fromJSON(object: any): RelayRequest;
    toJSON(message: RelayRequest): unknown;
    fromPartial(object: DeepPartial<RelayRequest>): RelayRequest;
};
export declare const RelayReply: {
    encode(message: RelayReply, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): RelayReply;
    fromJSON(object: any): RelayReply;
    toJSON(message: RelayReply): unknown;
    fromPartial(object: DeepPartial<RelayReply>): RelayReply;
};
export interface Relayer {
    Relay(request: RelayRequest): Promise<RelayReply>;
}
export declare class RelayerClientImpl implements Relayer {
    private readonly rpc;
    constructor(rpc: Rpc);
    Relay(request: RelayRequest): Promise<RelayReply>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
