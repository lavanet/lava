/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.conflict";

/** Params defines the parameters for the module. */
export interface Params {
  majorityPercent: string;
  voteStartSpan: Long;
  votePeriod: Long;
  Rewards?: Rewards;
}

export interface Rewards {
  winnerRewardPercent: string;
  clientRewardPercent: string;
  votersRewardPercent: string;
}

function createBaseParams(): Params {
  return { majorityPercent: "", voteStartSpan: Long.UZERO, votePeriod: Long.UZERO, Rewards: undefined };
}

export const Params = {
  encode(message: Params, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.majorityPercent !== "") {
      writer.uint32(10).string(message.majorityPercent);
    }
    if (!message.voteStartSpan.isZero()) {
      writer.uint32(16).uint64(message.voteStartSpan);
    }
    if (!message.votePeriod.isZero()) {
      writer.uint32(24).uint64(message.votePeriod);
    }
    if (message.Rewards !== undefined) {
      Rewards.encode(message.Rewards, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Params {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.majorityPercent = reader.string();
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.voteStartSpan = reader.uint64() as Long;
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.votePeriod = reader.uint64() as Long;
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.Rewards = Rewards.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Params {
    return {
      majorityPercent: isSet(object.majorityPercent) ? String(object.majorityPercent) : "",
      voteStartSpan: isSet(object.voteStartSpan) ? Long.fromValue(object.voteStartSpan) : Long.UZERO,
      votePeriod: isSet(object.votePeriod) ? Long.fromValue(object.votePeriod) : Long.UZERO,
      Rewards: isSet(object.Rewards) ? Rewards.fromJSON(object.Rewards) : undefined,
    };
  },

  toJSON(message: Params): unknown {
    const obj: any = {};
    message.majorityPercent !== undefined && (obj.majorityPercent = message.majorityPercent);
    message.voteStartSpan !== undefined && (obj.voteStartSpan = (message.voteStartSpan || Long.UZERO).toString());
    message.votePeriod !== undefined && (obj.votePeriod = (message.votePeriod || Long.UZERO).toString());
    message.Rewards !== undefined && (obj.Rewards = message.Rewards ? Rewards.toJSON(message.Rewards) : undefined);
    return obj;
  },

  create<I extends Exact<DeepPartial<Params>, I>>(base?: I): Params {
    return Params.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Params>, I>>(object: I): Params {
    const message = createBaseParams();
    message.majorityPercent = object.majorityPercent ?? "";
    message.voteStartSpan = (object.voteStartSpan !== undefined && object.voteStartSpan !== null)
      ? Long.fromValue(object.voteStartSpan)
      : Long.UZERO;
    message.votePeriod = (object.votePeriod !== undefined && object.votePeriod !== null)
      ? Long.fromValue(object.votePeriod)
      : Long.UZERO;
    message.Rewards = (object.Rewards !== undefined && object.Rewards !== null)
      ? Rewards.fromPartial(object.Rewards)
      : undefined;
    return message;
  },
};

function createBaseRewards(): Rewards {
  return { winnerRewardPercent: "", clientRewardPercent: "", votersRewardPercent: "" };
}

export const Rewards = {
  encode(message: Rewards, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.winnerRewardPercent !== "") {
      writer.uint32(10).string(message.winnerRewardPercent);
    }
    if (message.clientRewardPercent !== "") {
      writer.uint32(18).string(message.clientRewardPercent);
    }
    if (message.votersRewardPercent !== "") {
      writer.uint32(26).string(message.votersRewardPercent);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Rewards {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRewards();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.winnerRewardPercent = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.clientRewardPercent = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.votersRewardPercent = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Rewards {
    return {
      winnerRewardPercent: isSet(object.winnerRewardPercent) ? String(object.winnerRewardPercent) : "",
      clientRewardPercent: isSet(object.clientRewardPercent) ? String(object.clientRewardPercent) : "",
      votersRewardPercent: isSet(object.votersRewardPercent) ? String(object.votersRewardPercent) : "",
    };
  },

  toJSON(message: Rewards): unknown {
    const obj: any = {};
    message.winnerRewardPercent !== undefined && (obj.winnerRewardPercent = message.winnerRewardPercent);
    message.clientRewardPercent !== undefined && (obj.clientRewardPercent = message.clientRewardPercent);
    message.votersRewardPercent !== undefined && (obj.votersRewardPercent = message.votersRewardPercent);
    return obj;
  },

  create<I extends Exact<DeepPartial<Rewards>, I>>(base?: I): Rewards {
    return Rewards.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Rewards>, I>>(object: I): Rewards {
    const message = createBaseRewards();
    message.winnerRewardPercent = object.winnerRewardPercent ?? "";
    message.clientRewardPercent = object.clientRewardPercent ?? "";
    message.votersRewardPercent = object.votersRewardPercent ?? "";
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
