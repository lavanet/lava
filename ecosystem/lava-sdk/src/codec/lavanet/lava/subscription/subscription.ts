/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.subscription";

export interface Subscription {
  /** creator pays for the subscription */
  creator: string;
  /** consumer uses the subscription */
  consumer: string;
  /** when the subscription was last recharged */
  block: Long;
  /** index (name) of plan */
  planIndex: string;
  /** when the plan was created */
  planBlock: Long;
  /** total requested duration in months */
  durationTotal: Long;
  /** remaining duration in months */
  durationLeft: Long;
  /** expiry time of current month */
  monthExpiryTime: Long;
  /** CU allowance during current month */
  monthCuTotal: Long;
  /** CU remaining during current month */
  monthCuLeft: Long;
}

function createBaseSubscription(): Subscription {
  return {
    creator: "",
    consumer: "",
    block: Long.UZERO,
    planIndex: "",
    planBlock: Long.UZERO,
    durationTotal: Long.UZERO,
    durationLeft: Long.UZERO,
    monthExpiryTime: Long.UZERO,
    monthCuTotal: Long.UZERO,
    monthCuLeft: Long.UZERO,
  };
}

export const Subscription = {
  encode(message: Subscription, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.consumer !== "") {
      writer.uint32(18).string(message.consumer);
    }
    if (!message.block.isZero()) {
      writer.uint32(24).uint64(message.block);
    }
    if (message.planIndex !== "") {
      writer.uint32(34).string(message.planIndex);
    }
    if (!message.planBlock.isZero()) {
      writer.uint32(40).uint64(message.planBlock);
    }
    if (!message.durationTotal.isZero()) {
      writer.uint32(48).uint64(message.durationTotal);
    }
    if (!message.durationLeft.isZero()) {
      writer.uint32(56).uint64(message.durationLeft);
    }
    if (!message.monthExpiryTime.isZero()) {
      writer.uint32(64).uint64(message.monthExpiryTime);
    }
    if (!message.monthCuTotal.isZero()) {
      writer.uint32(80).uint64(message.monthCuTotal);
    }
    if (!message.monthCuLeft.isZero()) {
      writer.uint32(88).uint64(message.monthCuLeft);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Subscription {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSubscription();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.consumer = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.block = reader.uint64() as Long;
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.planIndex = reader.string();
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.planBlock = reader.uint64() as Long;
          continue;
        case 6:
          if (tag != 48) {
            break;
          }

          message.durationTotal = reader.uint64() as Long;
          continue;
        case 7:
          if (tag != 56) {
            break;
          }

          message.durationLeft = reader.uint64() as Long;
          continue;
        case 8:
          if (tag != 64) {
            break;
          }

          message.monthExpiryTime = reader.uint64() as Long;
          continue;
        case 10:
          if (tag != 80) {
            break;
          }

          message.monthCuTotal = reader.uint64() as Long;
          continue;
        case 11:
          if (tag != 88) {
            break;
          }

          message.monthCuLeft = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Subscription {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      consumer: isSet(object.consumer) ? String(object.consumer) : "",
      block: isSet(object.block) ? Long.fromValue(object.block) : Long.UZERO,
      planIndex: isSet(object.planIndex) ? String(object.planIndex) : "",
      planBlock: isSet(object.planBlock) ? Long.fromValue(object.planBlock) : Long.UZERO,
      durationTotal: isSet(object.durationTotal) ? Long.fromValue(object.durationTotal) : Long.UZERO,
      durationLeft: isSet(object.durationLeft) ? Long.fromValue(object.durationLeft) : Long.UZERO,
      monthExpiryTime: isSet(object.monthExpiryTime) ? Long.fromValue(object.monthExpiryTime) : Long.UZERO,
      monthCuTotal: isSet(object.monthCuTotal) ? Long.fromValue(object.monthCuTotal) : Long.UZERO,
      monthCuLeft: isSet(object.monthCuLeft) ? Long.fromValue(object.monthCuLeft) : Long.UZERO,
    };
  },

  toJSON(message: Subscription): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.consumer !== undefined && (obj.consumer = message.consumer);
    message.block !== undefined && (obj.block = (message.block || Long.UZERO).toString());
    message.planIndex !== undefined && (obj.planIndex = message.planIndex);
    message.planBlock !== undefined && (obj.planBlock = (message.planBlock || Long.UZERO).toString());
    message.durationTotal !== undefined && (obj.durationTotal = (message.durationTotal || Long.UZERO).toString());
    message.durationLeft !== undefined && (obj.durationLeft = (message.durationLeft || Long.UZERO).toString());
    message.monthExpiryTime !== undefined && (obj.monthExpiryTime = (message.monthExpiryTime || Long.UZERO).toString());
    message.monthCuTotal !== undefined && (obj.monthCuTotal = (message.monthCuTotal || Long.UZERO).toString());
    message.monthCuLeft !== undefined && (obj.monthCuLeft = (message.monthCuLeft || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<Subscription>, I>>(base?: I): Subscription {
    return Subscription.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Subscription>, I>>(object: I): Subscription {
    const message = createBaseSubscription();
    message.creator = object.creator ?? "";
    message.consumer = object.consumer ?? "";
    message.block = (object.block !== undefined && object.block !== null) ? Long.fromValue(object.block) : Long.UZERO;
    message.planIndex = object.planIndex ?? "";
    message.planBlock = (object.planBlock !== undefined && object.planBlock !== null)
      ? Long.fromValue(object.planBlock)
      : Long.UZERO;
    message.durationTotal = (object.durationTotal !== undefined && object.durationTotal !== null)
      ? Long.fromValue(object.durationTotal)
      : Long.UZERO;
    message.durationLeft = (object.durationLeft !== undefined && object.durationLeft !== null)
      ? Long.fromValue(object.durationLeft)
      : Long.UZERO;
    message.monthExpiryTime = (object.monthExpiryTime !== undefined && object.monthExpiryTime !== null)
      ? Long.fromValue(object.monthExpiryTime)
      : Long.UZERO;
    message.monthCuTotal = (object.monthCuTotal !== undefined && object.monthCuTotal !== null)
      ? Long.fromValue(object.monthCuTotal)
      : Long.UZERO;
    message.monthCuLeft = (object.monthCuLeft !== undefined && object.monthCuLeft !== null)
      ? Long.fromValue(object.monthCuLeft)
      : Long.UZERO;
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends Array<infer U> ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
