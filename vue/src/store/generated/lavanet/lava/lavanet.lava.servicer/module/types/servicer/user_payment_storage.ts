/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { UniquePaymentStorageUserServicer } from "../servicer/unique_payment_storage_user_servicer";

export const protobufPackage = "lavanet.lava.servicer";

export interface UserPaymentStorage {
  index: string;
  uniquePaymentStorageUserServicer:
    | UniquePaymentStorageUserServicer
    | undefined;
  totalCU: number;
  session: number;
}

const baseUserPaymentStorage: object = { index: "", totalCU: 0, session: 0 };

export const UserPaymentStorage = {
  encode(
    message: UserPaymentStorage,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.uniquePaymentStorageUserServicer !== undefined) {
      UniquePaymentStorageUserServicer.encode(
        message.uniquePaymentStorageUserServicer,
        writer.uint32(18).fork()
      ).ldelim();
    }
    if (message.totalCU !== 0) {
      writer.uint32(24).uint64(message.totalCU);
    }
    if (message.session !== 0) {
      writer.uint32(32).uint64(message.session);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): UserPaymentStorage {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseUserPaymentStorage } as UserPaymentStorage;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        case 2:
          message.uniquePaymentStorageUserServicer = UniquePaymentStorageUserServicer.decode(
            reader,
            reader.uint32()
          );
          break;
        case 3:
          message.totalCU = longToNumber(reader.uint64() as Long);
          break;
        case 4:
          message.session = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UserPaymentStorage {
    const message = { ...baseUserPaymentStorage } as UserPaymentStorage;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    if (
      object.uniquePaymentStorageUserServicer !== undefined &&
      object.uniquePaymentStorageUserServicer !== null
    ) {
      message.uniquePaymentStorageUserServicer = UniquePaymentStorageUserServicer.fromJSON(
        object.uniquePaymentStorageUserServicer
      );
    } else {
      message.uniquePaymentStorageUserServicer = undefined;
    }
    if (object.totalCU !== undefined && object.totalCU !== null) {
      message.totalCU = Number(object.totalCU);
    } else {
      message.totalCU = 0;
    }
    if (object.session !== undefined && object.session !== null) {
      message.session = Number(object.session);
    } else {
      message.session = 0;
    }
    return message;
  },

  toJSON(message: UserPaymentStorage): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.uniquePaymentStorageUserServicer !== undefined &&
      (obj.uniquePaymentStorageUserServicer = message.uniquePaymentStorageUserServicer
        ? UniquePaymentStorageUserServicer.toJSON(
            message.uniquePaymentStorageUserServicer
          )
        : undefined);
    message.totalCU !== undefined && (obj.totalCU = message.totalCU);
    message.session !== undefined && (obj.session = message.session);
    return obj;
  },

  fromPartial(object: DeepPartial<UserPaymentStorage>): UserPaymentStorage {
    const message = { ...baseUserPaymentStorage } as UserPaymentStorage;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    if (
      object.uniquePaymentStorageUserServicer !== undefined &&
      object.uniquePaymentStorageUserServicer !== null
    ) {
      message.uniquePaymentStorageUserServicer = UniquePaymentStorageUserServicer.fromPartial(
        object.uniquePaymentStorageUserServicer
      );
    } else {
      message.uniquePaymentStorageUserServicer = undefined;
    }
    if (object.totalCU !== undefined && object.totalCU !== null) {
      message.totalCU = object.totalCU;
    } else {
      message.totalCU = 0;
    }
    if (object.session !== undefined && object.session !== null) {
      message.session = object.session;
    } else {
      message.session = 0;
    }
    return message;
  },
};

declare var self: any | undefined;
declare var window: any | undefined;
var globalThis: any = (() => {
  if (typeof globalThis !== "undefined") return globalThis;
  if (typeof self !== "undefined") return self;
  if (typeof window !== "undefined") return window;
  if (typeof global !== "undefined") return global;
  throw "Unable to locate global object";
})();

type Builtin = Date | Function | Uint8Array | string | number | undefined;
export type DeepPartial<T> = T extends Builtin
  ? T
  : T extends Array<infer U>
  ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U>
  ? ReadonlyArray<DeepPartial<U>>
  : T extends {}
  ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (util.Long !== Long) {
  util.Long = Long as any;
  configure();
}
