import { Writer, Reader } from "protobufjs/minimal";
import { ServiceApi } from "../spec/service_api";
export declare const protobufPackage = "lavanet.lava.spec";
export interface Spec {
    id: number;
    name: string;
    apis: ServiceApi[];
    status: string;
}
export declare const Spec: {
    encode(message: Spec, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): Spec;
    fromJSON(object: any): Spec;
    toJSON(message: Spec): unknown;
    fromPartial(object: DeepPartial<Spec>): Spec;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
