import Long from "long";
import _m0 from "protobufjs/minimal";
import { RawMessage } from "../common/fixationEntry";
import { Params } from "./params";
export declare const protobufPackage = "lavanet.lava.projects";
/** GenesisState defines the projects module's genesis state. */
export interface GenesisState {
    params?: Params;
    projectsFS: RawMessage[];
    /** this line is used by starport scaffolding # genesis/proto/state */
    developerFS: RawMessage[];
}
export declare const GenesisState: {
    encode(message: GenesisState, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): GenesisState;
    fromJSON(object: any): GenesisState;
    toJSON(message: GenesisState): unknown;
    create<I extends {
        params?: {} | undefined;
        projectsFS?: {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] | undefined;
        developerFS?: {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] | undefined;
    } & {
        params?: ({} & {} & { [K in Exclude<keyof I["params"], never>]: never; }) | undefined;
        projectsFS?: ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] & ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & { [K_1 in Exclude<keyof I["projectsFS"][number], keyof RawMessage>]: never; })[] & { [K_2 in Exclude<keyof I["projectsFS"], keyof {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[]>]: never; }) | undefined;
        developerFS?: ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] & ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & { [K_3 in Exclude<keyof I["developerFS"][number], keyof RawMessage>]: never; })[] & { [K_4 in Exclude<keyof I["developerFS"], keyof {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_5 in Exclude<keyof I, keyof GenesisState>]: never; }>(base?: I | undefined): GenesisState;
    fromPartial<I_1 extends {
        params?: {} | undefined;
        projectsFS?: {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] | undefined;
        developerFS?: {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] | undefined;
    } & {
        params?: ({} & {} & { [K_6 in Exclude<keyof I_1["params"], never>]: never; }) | undefined;
        projectsFS?: ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] & ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & { [K_7 in Exclude<keyof I_1["projectsFS"][number], keyof RawMessage>]: never; })[] & { [K_8 in Exclude<keyof I_1["projectsFS"], keyof {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[]>]: never; }) | undefined;
        developerFS?: ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[] & ({
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        } & { [K_9 in Exclude<keyof I_1["developerFS"][number], keyof RawMessage>]: never; })[] & { [K_10 in Exclude<keyof I_1["developerFS"], keyof {
            key?: Uint8Array | undefined;
            value?: Uint8Array | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_11 in Exclude<keyof I_1, keyof GenesisState>]: never; }>(object: I_1): GenesisState;
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
