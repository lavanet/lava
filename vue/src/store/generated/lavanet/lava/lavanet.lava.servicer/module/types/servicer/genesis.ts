/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Params } from "../servicer/params";
import { StakeMap } from "../servicer/stake_map";
import { SpecStakeStorage } from "../servicer/spec_stake_storage";
import { BlockDeadlineForCallback } from "../servicer/block_deadline_for_callback";
import { UnstakingServicersAllSpecs } from "../servicer/unstaking_servicers_all_specs";
import { CurrentSessionStart } from "../servicer/current_session_start";
import { PreviousSessionBlocks } from "../servicer/previous_session_blocks";
import { SessionStorageForSpec } from "../servicer/session_storage_for_spec";
import { EarliestSessionStart } from "../servicer/earliest_session_start";
import { UniquePaymentStorageUserServicer } from "../servicer/unique_payment_storage_user_servicer";
import { UserPaymentStorage } from "../servicer/user_payment_storage";
import { SessionPayments } from "../servicer/session_payments";

export const protobufPackage = "lavanet.lava.servicer";

/** GenesisState defines the servicer module's genesis state. */
export interface GenesisState {
  params: Params | undefined;
  stakeMapList: StakeMap[];
  specStakeStorageList: SpecStakeStorage[];
  blockDeadlineForCallback: BlockDeadlineForCallback | undefined;
  unstakingServicersAllSpecsList: UnstakingServicersAllSpecs[];
  unstakingServicersAllSpecsCount: number;
  currentSessionStart: CurrentSessionStart | undefined;
  previousSessionBlocks: PreviousSessionBlocks | undefined;
  sessionStorageForSpecList: SessionStorageForSpec[];
  earliestSessionStart: EarliestSessionStart | undefined;
  uniquePaymentStorageUserServicerList: UniquePaymentStorageUserServicer[];
  userPaymentStorageList: UserPaymentStorage[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  sessionPaymentsList: SessionPayments[];
}

const baseGenesisState: object = { unstakingServicersAllSpecsCount: 0 };

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.stakeMapList) {
      StakeMap.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.specStakeStorageList) {
      SpecStakeStorage.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    if (message.blockDeadlineForCallback !== undefined) {
      BlockDeadlineForCallback.encode(
        message.blockDeadlineForCallback,
        writer.uint32(34).fork()
      ).ldelim();
    }
    for (const v of message.unstakingServicersAllSpecsList) {
      UnstakingServicersAllSpecs.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    if (message.unstakingServicersAllSpecsCount !== 0) {
      writer.uint32(48).uint64(message.unstakingServicersAllSpecsCount);
    }
    if (message.currentSessionStart !== undefined) {
      CurrentSessionStart.encode(
        message.currentSessionStart,
        writer.uint32(58).fork()
      ).ldelim();
    }
    if (message.previousSessionBlocks !== undefined) {
      PreviousSessionBlocks.encode(
        message.previousSessionBlocks,
        writer.uint32(66).fork()
      ).ldelim();
    }
    for (const v of message.sessionStorageForSpecList) {
      SessionStorageForSpec.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    if (message.earliestSessionStart !== undefined) {
      EarliestSessionStart.encode(
        message.earliestSessionStart,
        writer.uint32(82).fork()
      ).ldelim();
    }
    for (const v of message.uniquePaymentStorageUserServicerList) {
      UniquePaymentStorageUserServicer.encode(
        v!,
        writer.uint32(90).fork()
      ).ldelim();
    }
    for (const v of message.userPaymentStorageList) {
      UserPaymentStorage.encode(v!, writer.uint32(98).fork()).ldelim();
    }
    for (const v of message.sessionPaymentsList) {
      SessionPayments.encode(v!, writer.uint32(106).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.stakeMapList = [];
    message.specStakeStorageList = [];
    message.unstakingServicersAllSpecsList = [];
    message.sessionStorageForSpecList = [];
    message.uniquePaymentStorageUserServicerList = [];
    message.userPaymentStorageList = [];
    message.sessionPaymentsList = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.stakeMapList.push(StakeMap.decode(reader, reader.uint32()));
          break;
        case 3:
          message.specStakeStorageList.push(
            SpecStakeStorage.decode(reader, reader.uint32())
          );
          break;
        case 4:
          message.blockDeadlineForCallback = BlockDeadlineForCallback.decode(
            reader,
            reader.uint32()
          );
          break;
        case 5:
          message.unstakingServicersAllSpecsList.push(
            UnstakingServicersAllSpecs.decode(reader, reader.uint32())
          );
          break;
        case 6:
          message.unstakingServicersAllSpecsCount = longToNumber(
            reader.uint64() as Long
          );
          break;
        case 7:
          message.currentSessionStart = CurrentSessionStart.decode(
            reader,
            reader.uint32()
          );
          break;
        case 8:
          message.previousSessionBlocks = PreviousSessionBlocks.decode(
            reader,
            reader.uint32()
          );
          break;
        case 9:
          message.sessionStorageForSpecList.push(
            SessionStorageForSpec.decode(reader, reader.uint32())
          );
          break;
        case 10:
          message.earliestSessionStart = EarliestSessionStart.decode(
            reader,
            reader.uint32()
          );
          break;
        case 11:
          message.uniquePaymentStorageUserServicerList.push(
            UniquePaymentStorageUserServicer.decode(reader, reader.uint32())
          );
          break;
        case 12:
          message.userPaymentStorageList.push(
            UserPaymentStorage.decode(reader, reader.uint32())
          );
          break;
        case 13:
          message.sessionPaymentsList.push(
            SessionPayments.decode(reader, reader.uint32())
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.stakeMapList = [];
    message.specStakeStorageList = [];
    message.unstakingServicersAllSpecsList = [];
    message.sessionStorageForSpecList = [];
    message.uniquePaymentStorageUserServicerList = [];
    message.userPaymentStorageList = [];
    message.sessionPaymentsList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromJSON(object.params);
    } else {
      message.params = undefined;
    }
    if (object.stakeMapList !== undefined && object.stakeMapList !== null) {
      for (const e of object.stakeMapList) {
        message.stakeMapList.push(StakeMap.fromJSON(e));
      }
    }
    if (
      object.specStakeStorageList !== undefined &&
      object.specStakeStorageList !== null
    ) {
      for (const e of object.specStakeStorageList) {
        message.specStakeStorageList.push(SpecStakeStorage.fromJSON(e));
      }
    }
    if (
      object.blockDeadlineForCallback !== undefined &&
      object.blockDeadlineForCallback !== null
    ) {
      message.blockDeadlineForCallback = BlockDeadlineForCallback.fromJSON(
        object.blockDeadlineForCallback
      );
    } else {
      message.blockDeadlineForCallback = undefined;
    }
    if (
      object.unstakingServicersAllSpecsList !== undefined &&
      object.unstakingServicersAllSpecsList !== null
    ) {
      for (const e of object.unstakingServicersAllSpecsList) {
        message.unstakingServicersAllSpecsList.push(
          UnstakingServicersAllSpecs.fromJSON(e)
        );
      }
    }
    if (
      object.unstakingServicersAllSpecsCount !== undefined &&
      object.unstakingServicersAllSpecsCount !== null
    ) {
      message.unstakingServicersAllSpecsCount = Number(
        object.unstakingServicersAllSpecsCount
      );
    } else {
      message.unstakingServicersAllSpecsCount = 0;
    }
    if (
      object.currentSessionStart !== undefined &&
      object.currentSessionStart !== null
    ) {
      message.currentSessionStart = CurrentSessionStart.fromJSON(
        object.currentSessionStart
      );
    } else {
      message.currentSessionStart = undefined;
    }
    if (
      object.previousSessionBlocks !== undefined &&
      object.previousSessionBlocks !== null
    ) {
      message.previousSessionBlocks = PreviousSessionBlocks.fromJSON(
        object.previousSessionBlocks
      );
    } else {
      message.previousSessionBlocks = undefined;
    }
    if (
      object.sessionStorageForSpecList !== undefined &&
      object.sessionStorageForSpecList !== null
    ) {
      for (const e of object.sessionStorageForSpecList) {
        message.sessionStorageForSpecList.push(
          SessionStorageForSpec.fromJSON(e)
        );
      }
    }
    if (
      object.earliestSessionStart !== undefined &&
      object.earliestSessionStart !== null
    ) {
      message.earliestSessionStart = EarliestSessionStart.fromJSON(
        object.earliestSessionStart
      );
    } else {
      message.earliestSessionStart = undefined;
    }
    if (
      object.uniquePaymentStorageUserServicerList !== undefined &&
      object.uniquePaymentStorageUserServicerList !== null
    ) {
      for (const e of object.uniquePaymentStorageUserServicerList) {
        message.uniquePaymentStorageUserServicerList.push(
          UniquePaymentStorageUserServicer.fromJSON(e)
        );
      }
    }
    if (
      object.userPaymentStorageList !== undefined &&
      object.userPaymentStorageList !== null
    ) {
      for (const e of object.userPaymentStorageList) {
        message.userPaymentStorageList.push(UserPaymentStorage.fromJSON(e));
      }
    }
    if (
      object.sessionPaymentsList !== undefined &&
      object.sessionPaymentsList !== null
    ) {
      for (const e of object.sessionPaymentsList) {
        message.sessionPaymentsList.push(SessionPayments.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined &&
      (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    if (message.stakeMapList) {
      obj.stakeMapList = message.stakeMapList.map((e) =>
        e ? StakeMap.toJSON(e) : undefined
      );
    } else {
      obj.stakeMapList = [];
    }
    if (message.specStakeStorageList) {
      obj.specStakeStorageList = message.specStakeStorageList.map((e) =>
        e ? SpecStakeStorage.toJSON(e) : undefined
      );
    } else {
      obj.specStakeStorageList = [];
    }
    message.blockDeadlineForCallback !== undefined &&
      (obj.blockDeadlineForCallback = message.blockDeadlineForCallback
        ? BlockDeadlineForCallback.toJSON(message.blockDeadlineForCallback)
        : undefined);
    if (message.unstakingServicersAllSpecsList) {
      obj.unstakingServicersAllSpecsList = message.unstakingServicersAllSpecsList.map(
        (e) => (e ? UnstakingServicersAllSpecs.toJSON(e) : undefined)
      );
    } else {
      obj.unstakingServicersAllSpecsList = [];
    }
    message.unstakingServicersAllSpecsCount !== undefined &&
      (obj.unstakingServicersAllSpecsCount =
        message.unstakingServicersAllSpecsCount);
    message.currentSessionStart !== undefined &&
      (obj.currentSessionStart = message.currentSessionStart
        ? CurrentSessionStart.toJSON(message.currentSessionStart)
        : undefined);
    message.previousSessionBlocks !== undefined &&
      (obj.previousSessionBlocks = message.previousSessionBlocks
        ? PreviousSessionBlocks.toJSON(message.previousSessionBlocks)
        : undefined);
    if (message.sessionStorageForSpecList) {
      obj.sessionStorageForSpecList = message.sessionStorageForSpecList.map(
        (e) => (e ? SessionStorageForSpec.toJSON(e) : undefined)
      );
    } else {
      obj.sessionStorageForSpecList = [];
    }
    message.earliestSessionStart !== undefined &&
      (obj.earliestSessionStart = message.earliestSessionStart
        ? EarliestSessionStart.toJSON(message.earliestSessionStart)
        : undefined);
    if (message.uniquePaymentStorageUserServicerList) {
      obj.uniquePaymentStorageUserServicerList = message.uniquePaymentStorageUserServicerList.map(
        (e) => (e ? UniquePaymentStorageUserServicer.toJSON(e) : undefined)
      );
    } else {
      obj.uniquePaymentStorageUserServicerList = [];
    }
    if (message.userPaymentStorageList) {
      obj.userPaymentStorageList = message.userPaymentStorageList.map((e) =>
        e ? UserPaymentStorage.toJSON(e) : undefined
      );
    } else {
      obj.userPaymentStorageList = [];
    }
    if (message.sessionPaymentsList) {
      obj.sessionPaymentsList = message.sessionPaymentsList.map((e) =>
        e ? SessionPayments.toJSON(e) : undefined
      );
    } else {
      obj.sessionPaymentsList = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.stakeMapList = [];
    message.specStakeStorageList = [];
    message.unstakingServicersAllSpecsList = [];
    message.sessionStorageForSpecList = [];
    message.uniquePaymentStorageUserServicerList = [];
    message.userPaymentStorageList = [];
    message.sessionPaymentsList = [];
    if (object.params !== undefined && object.params !== null) {
      message.params = Params.fromPartial(object.params);
    } else {
      message.params = undefined;
    }
    if (object.stakeMapList !== undefined && object.stakeMapList !== null) {
      for (const e of object.stakeMapList) {
        message.stakeMapList.push(StakeMap.fromPartial(e));
      }
    }
    if (
      object.specStakeStorageList !== undefined &&
      object.specStakeStorageList !== null
    ) {
      for (const e of object.specStakeStorageList) {
        message.specStakeStorageList.push(SpecStakeStorage.fromPartial(e));
      }
    }
    if (
      object.blockDeadlineForCallback !== undefined &&
      object.blockDeadlineForCallback !== null
    ) {
      message.blockDeadlineForCallback = BlockDeadlineForCallback.fromPartial(
        object.blockDeadlineForCallback
      );
    } else {
      message.blockDeadlineForCallback = undefined;
    }
    if (
      object.unstakingServicersAllSpecsList !== undefined &&
      object.unstakingServicersAllSpecsList !== null
    ) {
      for (const e of object.unstakingServicersAllSpecsList) {
        message.unstakingServicersAllSpecsList.push(
          UnstakingServicersAllSpecs.fromPartial(e)
        );
      }
    }
    if (
      object.unstakingServicersAllSpecsCount !== undefined &&
      object.unstakingServicersAllSpecsCount !== null
    ) {
      message.unstakingServicersAllSpecsCount =
        object.unstakingServicersAllSpecsCount;
    } else {
      message.unstakingServicersAllSpecsCount = 0;
    }
    if (
      object.currentSessionStart !== undefined &&
      object.currentSessionStart !== null
    ) {
      message.currentSessionStart = CurrentSessionStart.fromPartial(
        object.currentSessionStart
      );
    } else {
      message.currentSessionStart = undefined;
    }
    if (
      object.previousSessionBlocks !== undefined &&
      object.previousSessionBlocks !== null
    ) {
      message.previousSessionBlocks = PreviousSessionBlocks.fromPartial(
        object.previousSessionBlocks
      );
    } else {
      message.previousSessionBlocks = undefined;
    }
    if (
      object.sessionStorageForSpecList !== undefined &&
      object.sessionStorageForSpecList !== null
    ) {
      for (const e of object.sessionStorageForSpecList) {
        message.sessionStorageForSpecList.push(
          SessionStorageForSpec.fromPartial(e)
        );
      }
    }
    if (
      object.earliestSessionStart !== undefined &&
      object.earliestSessionStart !== null
    ) {
      message.earliestSessionStart = EarliestSessionStart.fromPartial(
        object.earliestSessionStart
      );
    } else {
      message.earliestSessionStart = undefined;
    }
    if (
      object.uniquePaymentStorageUserServicerList !== undefined &&
      object.uniquePaymentStorageUserServicerList !== null
    ) {
      for (const e of object.uniquePaymentStorageUserServicerList) {
        message.uniquePaymentStorageUserServicerList.push(
          UniquePaymentStorageUserServicer.fromPartial(e)
        );
      }
    }
    if (
      object.userPaymentStorageList !== undefined &&
      object.userPaymentStorageList !== null
    ) {
      for (const e of object.userPaymentStorageList) {
        message.userPaymentStorageList.push(UserPaymentStorage.fromPartial(e));
      }
    }
    if (
      object.sessionPaymentsList !== undefined &&
      object.sessionPaymentsList !== null
    ) {
      for (const e of object.sessionPaymentsList) {
        message.sessionPaymentsList.push(SessionPayments.fromPartial(e));
      }
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
