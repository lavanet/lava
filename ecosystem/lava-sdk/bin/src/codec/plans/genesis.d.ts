import Long from "long";
import _m0 from "protobufjs/minimal";
import { RawMessage } from "../common/fixationEntry";
import { Params } from "./params";
export declare const protobufPackage = "lavanet.lava.plans";
/** GenesisState defines the plan module's genesis state. */
export interface GenesisState {
    params?: Params;
    /** this line is used by starport scaffolding # genesis/proto/state */
    plansFS: RawMessage[];
}
export declare const GenesisState: {
    encode(message: GenesisState, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): GenesisState;
    fromJSON(object: any): GenesisState;
    toJSON(message: GenesisState): unknown;
    create<I extends {
        params?: {} | undefined;
        plansFS?: {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] | undefined;
    } & {
        params?: ({} & {} & { [K in Exclude<keyof I["params"], never>]: never; }) | undefined;
        plansFS?: ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] & ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & { [K_1 in Exclude<keyof I["plansFS"][number], keyof RawMessage>]: never; })[] & { [K_2 in Exclude<keyof I["plansFS"], keyof {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_3 in Exclude<keyof I, keyof GenesisState>]: never; }>(base?: I | undefined): GenesisState;
    fromPartial<I_1 extends {
        params?: {} | undefined;
        plansFS?: {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] | undefined;
    } & {
        params?: ({} & {} & { [K_4 in Exclude<keyof I_1["params"], never>]: never; }) | undefined;
        plansFS?: ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] & ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & { [K_5 in Exclude<keyof I_1["plansFS"][number], keyof RawMessage>]: never; })[] & { [K_6 in Exclude<keyof I_1["plansFS"], keyof {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_7 in Exclude<keyof I_1, keyof GenesisState>]: never; }>(object: I_1): GenesisState;
};
declare type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Long ? string | number | Long : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
declare type KeysOfUnion<T> = T extends T ? keyof T : never;
export declare type Exact<P, I extends P> = P extends Builtin ? P : P & {
    [K in keyof P]: Exact<P[K], I[K]>;
} & {
    [K in Exclude<keyof I, KeysOfUnion<P>>]: never;
};
export {};
