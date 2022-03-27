import { Writer, Reader } from "protobufjs/minimal";
import { UniquePaymentStorageUserServicer } from "../servicer/unique_payment_storage_user_servicer";
export declare const protobufPackage = "lavanet.lava.servicer";
export interface UserPaymentStorage {
    index: string;
    uniquePaymentStorageUserServicer: UniquePaymentStorageUserServicer | undefined;
    totalCU: number;
    session: number;
}
export declare const UserPaymentStorage: {
    encode(message: UserPaymentStorage, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): UserPaymentStorage;
    fromJSON(object: any): UserPaymentStorage;
    toJSON(message: UserPaymentStorage): unknown;
    fromPartial(object: DeepPartial<UserPaymentStorage>): UserPaymentStorage;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
