import { Spec } from "../spec/spec";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.spec";
export interface SpecAddProposal {
    title: string;
    description: string;
    specs: Spec[];
}
export declare const SpecAddProposal: {
    encode(message: SpecAddProposal, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): SpecAddProposal;
    fromJSON(object: any): SpecAddProposal;
    toJSON(message: SpecAddProposal): unknown;
    fromPartial(object: DeepPartial<SpecAddProposal>): SpecAddProposal;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
