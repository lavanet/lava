import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface ClientRequest {
    data: string;
}
export declare const ClientRequest: {
    encode(message: ClientRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): ClientRequest;
    fromJSON(object: any): ClientRequest;
    toJSON(message: ClientRequest): unknown;
    fromPartial(object: DeepPartial<ClientRequest>): ClientRequest;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
