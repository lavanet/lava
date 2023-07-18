import Long from "long";
import _m0 from "protobufjs/minimal";
import { Coin } from "../../base/v1beta1/coin";
import { Params } from "./distribution";
export declare const protobufPackage = "cosmos.distribution.v1beta1";
/**
 * MsgSetWithdrawAddress sets the withdraw address for
 * a delegator (or validator self-delegation).
 */
export interface MsgSetWithdrawAddress {
    delegatorAddress: string;
    withdrawAddress: string;
}
/**
 * MsgSetWithdrawAddressResponse defines the Msg/SetWithdrawAddress response
 * type.
 */
export interface MsgSetWithdrawAddressResponse {
}
/**
 * MsgWithdrawDelegatorReward represents delegation withdrawal to a delegator
 * from a single validator.
 */
export interface MsgWithdrawDelegatorReward {
    delegatorAddress: string;
    validatorAddress: string;
}
/**
 * MsgWithdrawDelegatorRewardResponse defines the Msg/WithdrawDelegatorReward
 * response type.
 */
export interface MsgWithdrawDelegatorRewardResponse {
    /** Since: cosmos-sdk 0.46 */
    amount: Coin[];
}
/**
 * MsgWithdrawValidatorCommission withdraws the full commission to the validator
 * address.
 */
export interface MsgWithdrawValidatorCommission {
    validatorAddress: string;
}
/**
 * MsgWithdrawValidatorCommissionResponse defines the
 * Msg/WithdrawValidatorCommission response type.
 */
export interface MsgWithdrawValidatorCommissionResponse {
    /** Since: cosmos-sdk 0.46 */
    amount: Coin[];
}
/**
 * MsgFundCommunityPool allows an account to directly
 * fund the community pool.
 */
export interface MsgFundCommunityPool {
    amount: Coin[];
    depositor: string;
}
/** MsgFundCommunityPoolResponse defines the Msg/FundCommunityPool response type. */
export interface MsgFundCommunityPoolResponse {
}
/**
 * MsgUpdateParams is the Msg/UpdateParams request type.
 *
 * Since: cosmos-sdk 0.47
 */
export interface MsgUpdateParams {
    /** authority is the address of the governance account. */
    authority: string;
    /**
     * params defines the x/distribution parameters to update.
     *
     * NOTE: All parameters must be supplied.
     */
    params?: Params;
}
/**
 * MsgUpdateParamsResponse defines the response structure for executing a
 * MsgUpdateParams message.
 *
 * Since: cosmos-sdk 0.47
 */
export interface MsgUpdateParamsResponse {
}
/**
 * MsgCommunityPoolSpend defines a message for sending tokens from the community
 * pool to another account. This message is typically executed via a governance
 * proposal with the governance module being the executing authority.
 *
 * Since: cosmos-sdk 0.47
 */
export interface MsgCommunityPoolSpend {
    /** authority is the address of the governance account. */
    authority: string;
    recipient: string;
    amount: Coin[];
}
/**
 * MsgCommunityPoolSpendResponse defines the response to executing a
 * MsgCommunityPoolSpend message.
 *
 * Since: cosmos-sdk 0.47
 */
export interface MsgCommunityPoolSpendResponse {
}
export declare const MsgSetWithdrawAddress: {
    encode(message: MsgSetWithdrawAddress, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgSetWithdrawAddress;
    fromJSON(object: any): MsgSetWithdrawAddress;
    toJSON(message: MsgSetWithdrawAddress): unknown;
    create<I extends {
        delegatorAddress?: string | undefined;
        withdrawAddress?: string | undefined;
    } & {
        delegatorAddress?: string | undefined;
        withdrawAddress?: string | undefined;
    } & { [K in Exclude<keyof I, keyof MsgSetWithdrawAddress>]: never; }>(base?: I | undefined): MsgSetWithdrawAddress;
    fromPartial<I_1 extends {
        delegatorAddress?: string | undefined;
        withdrawAddress?: string | undefined;
    } & {
        delegatorAddress?: string | undefined;
        withdrawAddress?: string | undefined;
    } & { [K_1 in Exclude<keyof I_1, keyof MsgSetWithdrawAddress>]: never; }>(object: I_1): MsgSetWithdrawAddress;
};
export declare const MsgSetWithdrawAddressResponse: {
    encode(_: MsgSetWithdrawAddressResponse, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgSetWithdrawAddressResponse;
    fromJSON(_: any): MsgSetWithdrawAddressResponse;
    toJSON(_: MsgSetWithdrawAddressResponse): unknown;
    create<I extends {} & {} & { [K in Exclude<keyof I, never>]: never; }>(base?: I | undefined): MsgSetWithdrawAddressResponse;
    fromPartial<I_1 extends {} & {} & { [K_1 in Exclude<keyof I_1, never>]: never; }>(_: I_1): MsgSetWithdrawAddressResponse;
};
export declare const MsgWithdrawDelegatorReward: {
    encode(message: MsgWithdrawDelegatorReward, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgWithdrawDelegatorReward;
    fromJSON(object: any): MsgWithdrawDelegatorReward;
    toJSON(message: MsgWithdrawDelegatorReward): unknown;
    create<I extends {
        delegatorAddress?: string | undefined;
        validatorAddress?: string | undefined;
    } & {
        delegatorAddress?: string | undefined;
        validatorAddress?: string | undefined;
    } & { [K in Exclude<keyof I, keyof MsgWithdrawDelegatorReward>]: never; }>(base?: I | undefined): MsgWithdrawDelegatorReward;
    fromPartial<I_1 extends {
        delegatorAddress?: string | undefined;
        validatorAddress?: string | undefined;
    } & {
        delegatorAddress?: string | undefined;
        validatorAddress?: string | undefined;
    } & { [K_1 in Exclude<keyof I_1, keyof MsgWithdrawDelegatorReward>]: never; }>(object: I_1): MsgWithdrawDelegatorReward;
};
export declare const MsgWithdrawDelegatorRewardResponse: {
    encode(message: MsgWithdrawDelegatorRewardResponse, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgWithdrawDelegatorRewardResponse;
    fromJSON(object: any): MsgWithdrawDelegatorRewardResponse;
    toJSON(message: MsgWithdrawDelegatorRewardResponse): unknown;
    create<I extends {
        amount?: {
            denom?: string | undefined;
            amount?: string | undefined;
        }[] | undefined;
    } & {
        amount?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        }[] & ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K in Exclude<keyof I["amount"][number], keyof Coin>]: never; })[] & { [K_1 in Exclude<keyof I["amount"], keyof {
            denom?: string | undefined;
            amount?: string | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_2 in Exclude<keyof I, "amount">]: never; }>(base?: I | undefined): MsgWithdrawDelegatorRewardResponse;
    fromPartial<I_1 extends {
        amount?: {
            denom?: string | undefined;
            amount?: string | undefined;
        }[] | undefined;
    } & {
        amount?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        }[] & ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K_3 in Exclude<keyof I_1["amount"][number], keyof Coin>]: never; })[] & { [K_4 in Exclude<keyof I_1["amount"], keyof {
            denom?: string | undefined;
            amount?: string | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_5 in Exclude<keyof I_1, "amount">]: never; }>(object: I_1): MsgWithdrawDelegatorRewardResponse;
};
export declare const MsgWithdrawValidatorCommission: {
    encode(message: MsgWithdrawValidatorCommission, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgWithdrawValidatorCommission;
    fromJSON(object: any): MsgWithdrawValidatorCommission;
    toJSON(message: MsgWithdrawValidatorCommission): unknown;
    create<I extends {
        validatorAddress?: string | undefined;
    } & {
        validatorAddress?: string | undefined;
    } & { [K in Exclude<keyof I, "validatorAddress">]: never; }>(base?: I | undefined): MsgWithdrawValidatorCommission;
    fromPartial<I_1 extends {
        validatorAddress?: string | undefined;
    } & {
        validatorAddress?: string | undefined;
    } & { [K_1 in Exclude<keyof I_1, "validatorAddress">]: never; }>(object: I_1): MsgWithdrawValidatorCommission;
};
export declare const MsgWithdrawValidatorCommissionResponse: {
    encode(message: MsgWithdrawValidatorCommissionResponse, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgWithdrawValidatorCommissionResponse;
    fromJSON(object: any): MsgWithdrawValidatorCommissionResponse;
    toJSON(message: MsgWithdrawValidatorCommissionResponse): unknown;
    create<I extends {
        amount?: {
            denom?: string | undefined;
            amount?: string | undefined;
        }[] | undefined;
    } & {
        amount?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        }[] & ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K in Exclude<keyof I["amount"][number], keyof Coin>]: never; })[] & { [K_1 in Exclude<keyof I["amount"], keyof {
            denom?: string | undefined;
            amount?: string | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_2 in Exclude<keyof I, "amount">]: never; }>(base?: I | undefined): MsgWithdrawValidatorCommissionResponse;
    fromPartial<I_1 extends {
        amount?: {
            denom?: string | undefined;
            amount?: string | undefined;
        }[] | undefined;
    } & {
        amount?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        }[] & ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K_3 in Exclude<keyof I_1["amount"][number], keyof Coin>]: never; })[] & { [K_4 in Exclude<keyof I_1["amount"], keyof {
            denom?: string | undefined;
            amount?: string | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_5 in Exclude<keyof I_1, "amount">]: never; }>(object: I_1): MsgWithdrawValidatorCommissionResponse;
};
export declare const MsgFundCommunityPool: {
    encode(message: MsgFundCommunityPool, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgFundCommunityPool;
    fromJSON(object: any): MsgFundCommunityPool;
    toJSON(message: MsgFundCommunityPool): unknown;
    create<I extends {
        amount?: {
            denom?: string | undefined;
            amount?: string | undefined;
        }[] | undefined;
        depositor?: string | undefined;
    } & {
        amount?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        }[] & ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K in Exclude<keyof I["amount"][number], keyof Coin>]: never; })[] & { [K_1 in Exclude<keyof I["amount"], keyof {
            denom?: string | undefined;
            amount?: string | undefined;
        }[]>]: never; }) | undefined;
        depositor?: string | undefined;
    } & { [K_2 in Exclude<keyof I, keyof MsgFundCommunityPool>]: never; }>(base?: I | undefined): MsgFundCommunityPool;
    fromPartial<I_1 extends {
        amount?: {
            denom?: string | undefined;
            amount?: string | undefined;
        }[] | undefined;
        depositor?: string | undefined;
    } & {
        amount?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        }[] & ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K_3 in Exclude<keyof I_1["amount"][number], keyof Coin>]: never; })[] & { [K_4 in Exclude<keyof I_1["amount"], keyof {
            denom?: string | undefined;
            amount?: string | undefined;
        }[]>]: never; }) | undefined;
        depositor?: string | undefined;
    } & { [K_5 in Exclude<keyof I_1, keyof MsgFundCommunityPool>]: never; }>(object: I_1): MsgFundCommunityPool;
};
export declare const MsgFundCommunityPoolResponse: {
    encode(_: MsgFundCommunityPoolResponse, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgFundCommunityPoolResponse;
    fromJSON(_: any): MsgFundCommunityPoolResponse;
    toJSON(_: MsgFundCommunityPoolResponse): unknown;
    create<I extends {} & {} & { [K in Exclude<keyof I, never>]: never; }>(base?: I | undefined): MsgFundCommunityPoolResponse;
    fromPartial<I_1 extends {} & {} & { [K_1 in Exclude<keyof I_1, never>]: never; }>(_: I_1): MsgFundCommunityPoolResponse;
};
export declare const MsgUpdateParams: {
    encode(message: MsgUpdateParams, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgUpdateParams;
    fromJSON(object: any): MsgUpdateParams;
    toJSON(message: MsgUpdateParams): unknown;
    create<I extends {
        authority?: string | undefined;
        params?: {
            communityTax?: string | undefined;
            baseProposerReward?: string | undefined;
            bonusProposerReward?: string | undefined;
            withdrawAddrEnabled?: boolean | undefined;
        } | undefined;
    } & {
        authority?: string | undefined;
        params?: ({
            communityTax?: string | undefined;
            baseProposerReward?: string | undefined;
            bonusProposerReward?: string | undefined;
            withdrawAddrEnabled?: boolean | undefined;
        } & {
            communityTax?: string | undefined;
            baseProposerReward?: string | undefined;
            bonusProposerReward?: string | undefined;
            withdrawAddrEnabled?: boolean | undefined;
        } & { [K in Exclude<keyof I["params"], keyof Params>]: never; }) | undefined;
    } & { [K_1 in Exclude<keyof I, keyof MsgUpdateParams>]: never; }>(base?: I | undefined): MsgUpdateParams;
    fromPartial<I_1 extends {
        authority?: string | undefined;
        params?: {
            communityTax?: string | undefined;
            baseProposerReward?: string | undefined;
            bonusProposerReward?: string | undefined;
            withdrawAddrEnabled?: boolean | undefined;
        } | undefined;
    } & {
        authority?: string | undefined;
        params?: ({
            communityTax?: string | undefined;
            baseProposerReward?: string | undefined;
            bonusProposerReward?: string | undefined;
            withdrawAddrEnabled?: boolean | undefined;
        } & {
            communityTax?: string | undefined;
            baseProposerReward?: string | undefined;
            bonusProposerReward?: string | undefined;
            withdrawAddrEnabled?: boolean | undefined;
        } & { [K_2 in Exclude<keyof I_1["params"], keyof Params>]: never; }) | undefined;
    } & { [K_3 in Exclude<keyof I_1, keyof MsgUpdateParams>]: never; }>(object: I_1): MsgUpdateParams;
};
export declare const MsgUpdateParamsResponse: {
    encode(_: MsgUpdateParamsResponse, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromJSON(_: any): MsgUpdateParamsResponse;
    toJSON(_: MsgUpdateParamsResponse): unknown;
    create<I extends {} & {} & { [K in Exclude<keyof I, never>]: never; }>(base?: I | undefined): MsgUpdateParamsResponse;
    fromPartial<I_1 extends {} & {} & { [K_1 in Exclude<keyof I_1, never>]: never; }>(_: I_1): MsgUpdateParamsResponse;
};
export declare const MsgCommunityPoolSpend: {
    encode(message: MsgCommunityPoolSpend, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgCommunityPoolSpend;
    fromJSON(object: any): MsgCommunityPoolSpend;
    toJSON(message: MsgCommunityPoolSpend): unknown;
    create<I extends {
        authority?: string | undefined;
        recipient?: string | undefined;
        amount?: {
            denom?: string | undefined;
            amount?: string | undefined;
        }[] | undefined;
    } & {
        authority?: string | undefined;
        recipient?: string | undefined;
        amount?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        }[] & ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K in Exclude<keyof I["amount"][number], keyof Coin>]: never; })[] & { [K_1 in Exclude<keyof I["amount"], keyof {
            denom?: string | undefined;
            amount?: string | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_2 in Exclude<keyof I, keyof MsgCommunityPoolSpend>]: never; }>(base?: I | undefined): MsgCommunityPoolSpend;
    fromPartial<I_1 extends {
        authority?: string | undefined;
        recipient?: string | undefined;
        amount?: {
            denom?: string | undefined;
            amount?: string | undefined;
        }[] | undefined;
    } & {
        authority?: string | undefined;
        recipient?: string | undefined;
        amount?: ({
            denom?: string | undefined;
            amount?: string | undefined;
        }[] & ({
            denom?: string | undefined;
            amount?: string | undefined;
        } & {
            denom?: string | undefined;
            amount?: string | undefined;
        } & { [K_3 in Exclude<keyof I_1["amount"][number], keyof Coin>]: never; })[] & { [K_4 in Exclude<keyof I_1["amount"], keyof {
            denom?: string | undefined;
            amount?: string | undefined;
        }[]>]: never; }) | undefined;
    } & { [K_5 in Exclude<keyof I_1, keyof MsgCommunityPoolSpend>]: never; }>(object: I_1): MsgCommunityPoolSpend;
};
export declare const MsgCommunityPoolSpendResponse: {
    encode(_: MsgCommunityPoolSpendResponse, writer?: _m0.Writer): _m0.Writer;
    decode(input: _m0.Reader | Uint8Array, length?: number): MsgCommunityPoolSpendResponse;
    fromJSON(_: any): MsgCommunityPoolSpendResponse;
    toJSON(_: MsgCommunityPoolSpendResponse): unknown;
    create<I extends {} & {} & { [K in Exclude<keyof I, never>]: never; }>(base?: I | undefined): MsgCommunityPoolSpendResponse;
    fromPartial<I_1 extends {} & {} & { [K_1 in Exclude<keyof I_1, never>]: never; }>(_: I_1): MsgCommunityPoolSpendResponse;
};
/** Msg defines the distribution Msg service. */
export interface Msg {
    /**
     * SetWithdrawAddress defines a method to change the withdraw address
     * for a delegator (or validator self-delegation).
     */
    SetWithdrawAddress(request: MsgSetWithdrawAddress): Promise<MsgSetWithdrawAddressResponse>;
    /**
     * WithdrawDelegatorReward defines a method to withdraw rewards of delegator
     * from a single validator.
     */
    WithdrawDelegatorReward(request: MsgWithdrawDelegatorReward): Promise<MsgWithdrawDelegatorRewardResponse>;
    /**
     * WithdrawValidatorCommission defines a method to withdraw the
     * full commission to the validator address.
     */
    WithdrawValidatorCommission(request: MsgWithdrawValidatorCommission): Promise<MsgWithdrawValidatorCommissionResponse>;
    /**
     * FundCommunityPool defines a method to allow an account to directly
     * fund the community pool.
     */
    FundCommunityPool(request: MsgFundCommunityPool): Promise<MsgFundCommunityPoolResponse>;
    /**
     * UpdateParams defines a governance operation for updating the x/distribution
     * module parameters. The authority is defined in the keeper.
     *
     * Since: cosmos-sdk 0.47
     */
    UpdateParams(request: MsgUpdateParams): Promise<MsgUpdateParamsResponse>;
    /**
     * CommunityPoolSpend defines a governance operation for sending tokens from
     * the community pool in the x/distribution module to another account, which
     * could be the governance module itself. The authority is defined in the
     * keeper.
     *
     * Since: cosmos-sdk 0.47
     */
    CommunityPoolSpend(request: MsgCommunityPoolSpend): Promise<MsgCommunityPoolSpendResponse>;
}
export declare class MsgClientImpl implements Msg {
    private readonly rpc;
    private readonly service;
    constructor(rpc: Rpc, opts?: {
        service?: string;
    });
    SetWithdrawAddress(request: MsgSetWithdrawAddress): Promise<MsgSetWithdrawAddressResponse>;
    WithdrawDelegatorReward(request: MsgWithdrawDelegatorReward): Promise<MsgWithdrawDelegatorRewardResponse>;
    WithdrawValidatorCommission(request: MsgWithdrawValidatorCommission): Promise<MsgWithdrawValidatorCommissionResponse>;
    FundCommunityPool(request: MsgFundCommunityPool): Promise<MsgFundCommunityPoolResponse>;
    UpdateParams(request: MsgUpdateParams): Promise<MsgUpdateParamsResponse>;
    CommunityPoolSpend(request: MsgCommunityPoolSpend): Promise<MsgCommunityPoolSpendResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
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
