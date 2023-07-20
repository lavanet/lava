/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.pairing";

/** Params defines the parameters for the module. */
export interface Params {
  mintCoinsPerCU: string;
  fraudStakeSlashingFactor: string;
  fraudSlashingAmount: Long;
  epochBlocksOverlap: Long;
  unpayLimit: string;
  slashLimit: string;
  dataReliabilityReward: string;
  QoSWeight: string;
  recommendedEpochNumToCollectPayment: Long;
}

function createBaseParams(): Params {
  return {
    mintCoinsPerCU: "",
    fraudStakeSlashingFactor: "",
    fraudSlashingAmount: Long.UZERO,
    epochBlocksOverlap: Long.UZERO,
    unpayLimit: "",
    slashLimit: "",
    dataReliabilityReward: "",
    QoSWeight: "",
    recommendedEpochNumToCollectPayment: Long.UZERO,
  };
}

export const Params = {
  encode(message: Params, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.mintCoinsPerCU !== "") {
      writer.uint32(26).string(message.mintCoinsPerCU);
    }
    if (message.fraudStakeSlashingFactor !== "") {
      writer.uint32(42).string(message.fraudStakeSlashingFactor);
    }
    if (!message.fraudSlashingAmount.isZero()) {
      writer.uint32(48).uint64(message.fraudSlashingAmount);
    }
    if (!message.epochBlocksOverlap.isZero()) {
      writer.uint32(64).uint64(message.epochBlocksOverlap);
    }
    if (message.unpayLimit !== "") {
      writer.uint32(82).string(message.unpayLimit);
    }
    if (message.slashLimit !== "") {
      writer.uint32(90).string(message.slashLimit);
    }
    if (message.dataReliabilityReward !== "") {
      writer.uint32(98).string(message.dataReliabilityReward);
    }
    if (message.QoSWeight !== "") {
      writer.uint32(106).string(message.QoSWeight);
    }
    if (!message.recommendedEpochNumToCollectPayment.isZero()) {
      writer.uint32(112).uint64(message.recommendedEpochNumToCollectPayment);
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
        case 3:
          if (tag != 26) {
            break;
          }

          message.mintCoinsPerCU = reader.string();
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.fraudStakeSlashingFactor = reader.string();
          continue;
        case 6:
          if (tag != 48) {
            break;
          }

          message.fraudSlashingAmount = reader.uint64() as Long;
          continue;
        case 8:
          if (tag != 64) {
            break;
          }

          message.epochBlocksOverlap = reader.uint64() as Long;
          continue;
        case 10:
          if (tag != 82) {
            break;
          }

          message.unpayLimit = reader.string();
          continue;
        case 11:
          if (tag != 90) {
            break;
          }

          message.slashLimit = reader.string();
          continue;
        case 12:
          if (tag != 98) {
            break;
          }

          message.dataReliabilityReward = reader.string();
          continue;
        case 13:
          if (tag != 106) {
            break;
          }

          message.QoSWeight = reader.string();
          continue;
        case 14:
          if (tag != 112) {
            break;
          }

          message.recommendedEpochNumToCollectPayment = reader.uint64() as Long;
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
      mintCoinsPerCU: isSet(object.mintCoinsPerCU) ? String(object.mintCoinsPerCU) : "",
      fraudStakeSlashingFactor: isSet(object.fraudStakeSlashingFactor) ? String(object.fraudStakeSlashingFactor) : "",
      fraudSlashingAmount: isSet(object.fraudSlashingAmount) ? Long.fromValue(object.fraudSlashingAmount) : Long.UZERO,
      epochBlocksOverlap: isSet(object.epochBlocksOverlap) ? Long.fromValue(object.epochBlocksOverlap) : Long.UZERO,
      unpayLimit: isSet(object.unpayLimit) ? String(object.unpayLimit) : "",
      slashLimit: isSet(object.slashLimit) ? String(object.slashLimit) : "",
      dataReliabilityReward: isSet(object.dataReliabilityReward) ? String(object.dataReliabilityReward) : "",
      QoSWeight: isSet(object.QoSWeight) ? String(object.QoSWeight) : "",
      recommendedEpochNumToCollectPayment: isSet(object.recommendedEpochNumToCollectPayment)
        ? Long.fromValue(object.recommendedEpochNumToCollectPayment)
        : Long.UZERO,
    };
  },

  toJSON(message: Params): unknown {
    const obj: any = {};
    message.mintCoinsPerCU !== undefined && (obj.mintCoinsPerCU = message.mintCoinsPerCU);
    message.fraudStakeSlashingFactor !== undefined && (obj.fraudStakeSlashingFactor = message.fraudStakeSlashingFactor);
    message.fraudSlashingAmount !== undefined &&
      (obj.fraudSlashingAmount = (message.fraudSlashingAmount || Long.UZERO).toString());
    message.epochBlocksOverlap !== undefined &&
      (obj.epochBlocksOverlap = (message.epochBlocksOverlap || Long.UZERO).toString());
    message.unpayLimit !== undefined && (obj.unpayLimit = message.unpayLimit);
    message.slashLimit !== undefined && (obj.slashLimit = message.slashLimit);
    message.dataReliabilityReward !== undefined && (obj.dataReliabilityReward = message.dataReliabilityReward);
    message.QoSWeight !== undefined && (obj.QoSWeight = message.QoSWeight);
    message.recommendedEpochNumToCollectPayment !== undefined &&
      (obj.recommendedEpochNumToCollectPayment = (message.recommendedEpochNumToCollectPayment || Long.UZERO)
        .toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<Params>, I>>(base?: I): Params {
    return Params.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Params>, I>>(object: I): Params {
    const message = createBaseParams();
    message.mintCoinsPerCU = object.mintCoinsPerCU ?? "";
    message.fraudStakeSlashingFactor = object.fraudStakeSlashingFactor ?? "";
    message.fraudSlashingAmount = (object.fraudSlashingAmount !== undefined && object.fraudSlashingAmount !== null)
      ? Long.fromValue(object.fraudSlashingAmount)
      : Long.UZERO;
    message.epochBlocksOverlap = (object.epochBlocksOverlap !== undefined && object.epochBlocksOverlap !== null)
      ? Long.fromValue(object.epochBlocksOverlap)
      : Long.UZERO;
    message.unpayLimit = object.unpayLimit ?? "";
    message.slashLimit = object.slashLimit ?? "";
    message.dataReliabilityReward = object.dataReliabilityReward ?? "";
    message.QoSWeight = object.QoSWeight ?? "";
    message.recommendedEpochNumToCollectPayment =
      (object.recommendedEpochNumToCollectPayment !== undefined && object.recommendedEpochNumToCollectPayment !== null)
        ? Long.fromValue(object.recommendedEpochNumToCollectPayment)
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
