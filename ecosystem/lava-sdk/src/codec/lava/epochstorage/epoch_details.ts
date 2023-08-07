/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.epochstorage";

export interface EpochDetails {
  startBlock: Long;
  earliestStart: Long;
  deletedEpochs: Long[];
}

function createBaseEpochDetails(): EpochDetails {
  return { startBlock: Long.UZERO, earliestStart: Long.UZERO, deletedEpochs: [] };
}

export const EpochDetails = {
  encode(message: EpochDetails, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (!message.startBlock.isZero()) {
      writer.uint32(8).uint64(message.startBlock);
    }
    if (!message.earliestStart.isZero()) {
      writer.uint32(16).uint64(message.earliestStart);
    }
    writer.uint32(26).fork();
    for (const v of message.deletedEpochs) {
      writer.uint64(v);
    }
    writer.ldelim();
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EpochDetails {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEpochDetails();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 8) {
            break;
          }

          message.startBlock = reader.uint64() as Long;
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.earliestStart = reader.uint64() as Long;
          continue;
        case 3:
          if (tag == 24) {
            message.deletedEpochs.push(reader.uint64() as Long);
            continue;
          }

          if (tag == 26) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.deletedEpochs.push(reader.uint64() as Long);
            }

            continue;
          }

          break;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): EpochDetails {
    return {
      startBlock: isSet(object.startBlock) ? Long.fromValue(object.startBlock) : Long.UZERO,
      earliestStart: isSet(object.earliestStart) ? Long.fromValue(object.earliestStart) : Long.UZERO,
      deletedEpochs: Array.isArray(object?.deletedEpochs)
        ? object.deletedEpochs.map((e: any) => Long.fromValue(e))
        : [],
    };
  },

  toJSON(message: EpochDetails): unknown {
    const obj: any = {};
    message.startBlock !== undefined && (obj.startBlock = (message.startBlock || Long.UZERO).toString());
    message.earliestStart !== undefined && (obj.earliestStart = (message.earliestStart || Long.UZERO).toString());
    if (message.deletedEpochs) {
      obj.deletedEpochs = message.deletedEpochs.map((e) => (e || Long.UZERO).toString());
    } else {
      obj.deletedEpochs = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<EpochDetails>, I>>(base?: I): EpochDetails {
    return EpochDetails.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<EpochDetails>, I>>(object: I): EpochDetails {
    const message = createBaseEpochDetails();
    message.startBlock = (object.startBlock !== undefined && object.startBlock !== null)
      ? Long.fromValue(object.startBlock)
      : Long.UZERO;
    message.earliestStart = (object.earliestStart !== undefined && object.earliestStart !== null)
      ? Long.fromValue(object.earliestStart)
      : Long.UZERO;
    message.deletedEpochs = object.deletedEpochs?.map((e) => Long.fromValue(e)) || [];
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
