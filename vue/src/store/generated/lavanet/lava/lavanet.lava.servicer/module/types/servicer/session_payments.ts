/* eslint-disable */
import { UserPaymentStorage } from "../servicer/user_payment_storage";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.servicer";

export interface SessionPayments {
  index: string;
  usersPayments: UserPaymentStorage[];
}

const baseSessionPayments: object = { index: "" };

export const SessionPayments = {
  encode(message: SessionPayments, writer: Writer = Writer.create()): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    for (const v of message.usersPayments) {
      UserPaymentStorage.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): SessionPayments {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseSessionPayments } as SessionPayments;
    message.usersPayments = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        case 2:
          message.usersPayments.push(
            UserPaymentStorage.decode(reader, reader.uint32())
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SessionPayments {
    const message = { ...baseSessionPayments } as SessionPayments;
    message.usersPayments = [];
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    if (object.usersPayments !== undefined && object.usersPayments !== null) {
      for (const e of object.usersPayments) {
        message.usersPayments.push(UserPaymentStorage.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: SessionPayments): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    if (message.usersPayments) {
      obj.usersPayments = message.usersPayments.map((e) =>
        e ? UserPaymentStorage.toJSON(e) : undefined
      );
    } else {
      obj.usersPayments = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<SessionPayments>): SessionPayments {
    const message = { ...baseSessionPayments } as SessionPayments;
    message.usersPayments = [];
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    if (object.usersPayments !== undefined && object.usersPayments !== null) {
      for (const e of object.usersPayments) {
        message.usersPayments.push(UserPaymentStorage.fromPartial(e));
      }
    }
    return message;
  },
};

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
