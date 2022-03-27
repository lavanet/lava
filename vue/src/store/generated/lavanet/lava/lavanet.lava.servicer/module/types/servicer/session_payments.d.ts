import { UserPaymentStorage } from "../servicer/user_payment_storage";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface SessionPayments {
    index: string;
    usersPayments: UserPaymentStorage | undefined;
}
export declare const SessionPayments: {
    encode(message: SessionPayments, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): SessionPayments;
    fromJSON(object: any): SessionPayments;
    toJSON(message: SessionPayments): unknown;
    fromPartial(object: DeepPartial<SessionPayments>): SessionPayments;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
