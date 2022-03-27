import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.spec";
export interface ServiceApi {
    name: string;
    computeUnits: number;
    status: string;
}
export declare const ServiceApi: {
    encode(message: ServiceApi, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): ServiceApi;
    fromJSON(object: any): ServiceApi;
    toJSON(message: ServiceApi): unknown;
    fromPartial(object: DeepPartial<ServiceApi>): ServiceApi;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
