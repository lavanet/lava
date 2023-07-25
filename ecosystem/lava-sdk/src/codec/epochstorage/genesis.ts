/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { EpochDetails } from "./epoch_details";
import { FixatedParams } from "./fixated_params";
import { Params } from "./params";
import { StakeStorage } from "./stake_storage";

export const protobufPackage = "lavanet.lava.epochstorage";

/** GenesisState defines the epochstorage module's genesis state. */
export interface GenesisState {
  params?: Params;
  stakeStorageList: StakeStorage[];
  epochDetails?: EpochDetails;
  /** this line is used by starport scaffolding # genesis/proto/state */
  fixatedParamsList: FixatedParams[];
}

function createBaseGenesisState(): GenesisState {
  return { params: undefined, stakeStorageList: [], epochDetails: undefined, fixatedParamsList: [] };
}

export const GenesisState = {
  encode(message: GenesisState, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.stakeStorageList) {
      StakeStorage.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.epochDetails !== undefined) {
      EpochDetails.encode(message.epochDetails, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.fixatedParamsList) {
      FixatedParams.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.params = Params.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.stakeStorageList.push(StakeStorage.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.epochDetails = EpochDetails.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.fixatedParamsList.push(FixatedParams.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GenesisState {
    return {
      params: isSet(object.params) ? Params.fromJSON(object.params) : undefined,
      stakeStorageList: Array.isArray(object?.stakeStorageList)
        ? object.stakeStorageList.map((e: any) => StakeStorage.fromJSON(e))
        : [],
      epochDetails: isSet(object.epochDetails) ? EpochDetails.fromJSON(object.epochDetails) : undefined,
      fixatedParamsList: Array.isArray(object?.fixatedParamsList)
        ? object.fixatedParamsList.map((e: any) => FixatedParams.fromJSON(e))
        : [],
    };
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined && (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    if (message.stakeStorageList) {
      obj.stakeStorageList = message.stakeStorageList.map((e) => e ? StakeStorage.toJSON(e) : undefined);
    } else {
      obj.stakeStorageList = [];
    }
    message.epochDetails !== undefined &&
      (obj.epochDetails = message.epochDetails ? EpochDetails.toJSON(message.epochDetails) : undefined);
    if (message.fixatedParamsList) {
      obj.fixatedParamsList = message.fixatedParamsList.map((e) => e ? FixatedParams.toJSON(e) : undefined);
    } else {
      obj.fixatedParamsList = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GenesisState>, I>>(base?: I): GenesisState {
    return GenesisState.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<GenesisState>, I>>(object: I): GenesisState {
    const message = createBaseGenesisState();
    message.params = (object.params !== undefined && object.params !== null)
      ? Params.fromPartial(object.params)
      : undefined;
    message.stakeStorageList = object.stakeStorageList?.map((e) => StakeStorage.fromPartial(e)) || [];
    message.epochDetails = (object.epochDetails !== undefined && object.epochDetails !== null)
      ? EpochDetails.fromPartial(object.epochDetails)
      : undefined;
    message.fixatedParamsList = object.fixatedParamsList?.map((e) => FixatedParams.fromPartial(e)) || [];
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
