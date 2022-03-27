import { Spec } from "../spec/spec";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.spec";
export interface SpecModifyProposal {
    title: string;
    description: string;
    specs: Spec[];
}
export declare const SpecModifyProposal: {
    encode(message: SpecModifyProposal, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): SpecModifyProposal;
    fromJSON(object: any): SpecModifyProposal;
    toJSON(message: SpecModifyProposal): unknown;
    fromPartial(object: DeepPartial<SpecModifyProposal>): SpecModifyProposal;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
