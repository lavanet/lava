/* eslint-disable */
import { Reader, util, configure, Writer } from "protobufjs/minimal";
import * as Long from "long";
import { SpecName } from "../servicer/spec_name";
import { Coin } from "../cosmos/base/v1beta1/coin";
import { BlockNum } from "../servicer/block_num";
import { SessionID } from "../servicer/session_id";
import { ClientRequest } from "../servicer/client_request";
import { WorkProof } from "../servicer/work_proof";

export const protobufPackage = "lavanet.lava.servicer";

export interface MsgStakeServicer {
  creator: string;
  spec: SpecName | undefined;
  amount: Coin | undefined;
  deadline: BlockNum | undefined;
  operatorAddresses: string[];
}

export interface MsgStakeServicerResponse {}

export interface MsgUnstakeServicer {
  creator: string;
  spec: SpecName | undefined;
  deadline: BlockNum | undefined;
}

export interface MsgUnstakeServicerResponse {}

export interface MsgProofOfWork {
  creator: string;
  spec: SpecName | undefined;
  session: SessionID | undefined;
  clientRequest: ClientRequest | undefined;
  workProof: WorkProof | undefined;
  computeUnits: number;
  blockOfWork: BlockNum | undefined;
}

export interface MsgProofOfWorkResponse {}

const baseMsgStakeServicer: object = { creator: "", operatorAddresses: "" };

export const MsgStakeServicer = {
  encode(message: MsgStakeServicer, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.spec !== undefined) {
      SpecName.encode(message.spec, writer.uint32(18).fork()).ldelim();
    }
    if (message.amount !== undefined) {
      Coin.encode(message.amount, writer.uint32(26).fork()).ldelim();
    }
    if (message.deadline !== undefined) {
      BlockNum.encode(message.deadline, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.operatorAddresses) {
      writer.uint32(42).string(v!);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgStakeServicer {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgStakeServicer } as MsgStakeServicer;
    message.operatorAddresses = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.spec = SpecName.decode(reader, reader.uint32());
          break;
        case 3:
          message.amount = Coin.decode(reader, reader.uint32());
          break;
        case 4:
          message.deadline = BlockNum.decode(reader, reader.uint32());
          break;
        case 5:
          message.operatorAddresses.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgStakeServicer {
    const message = { ...baseMsgStakeServicer } as MsgStakeServicer;
    message.operatorAddresses = [];
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.spec !== undefined && object.spec !== null) {
      message.spec = SpecName.fromJSON(object.spec);
    } else {
      message.spec = undefined;
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = Coin.fromJSON(object.amount);
    } else {
      message.amount = undefined;
    }
    if (object.deadline !== undefined && object.deadline !== null) {
      message.deadline = BlockNum.fromJSON(object.deadline);
    } else {
      message.deadline = undefined;
    }
    if (
      object.operatorAddresses !== undefined &&
      object.operatorAddresses !== null
    ) {
      for (const e of object.operatorAddresses) {
        message.operatorAddresses.push(String(e));
      }
    }
    return message;
  },

  toJSON(message: MsgStakeServicer): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.spec !== undefined &&
      (obj.spec = message.spec ? SpecName.toJSON(message.spec) : undefined);
    message.amount !== undefined &&
      (obj.amount = message.amount ? Coin.toJSON(message.amount) : undefined);
    message.deadline !== undefined &&
      (obj.deadline = message.deadline
        ? BlockNum.toJSON(message.deadline)
        : undefined);
    if (message.operatorAddresses) {
      obj.operatorAddresses = message.operatorAddresses.map((e) => e);
    } else {
      obj.operatorAddresses = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<MsgStakeServicer>): MsgStakeServicer {
    const message = { ...baseMsgStakeServicer } as MsgStakeServicer;
    message.operatorAddresses = [];
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.spec !== undefined && object.spec !== null) {
      message.spec = SpecName.fromPartial(object.spec);
    } else {
      message.spec = undefined;
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = Coin.fromPartial(object.amount);
    } else {
      message.amount = undefined;
    }
    if (object.deadline !== undefined && object.deadline !== null) {
      message.deadline = BlockNum.fromPartial(object.deadline);
    } else {
      message.deadline = undefined;
    }
    if (
      object.operatorAddresses !== undefined &&
      object.operatorAddresses !== null
    ) {
      for (const e of object.operatorAddresses) {
        message.operatorAddresses.push(e);
      }
    }
    return message;
  },
};

const baseMsgStakeServicerResponse: object = {};

export const MsgStakeServicerResponse = {
  encode(
    _: MsgStakeServicerResponse,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgStakeServicerResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgStakeServicerResponse,
    } as MsgStakeServicerResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): MsgStakeServicerResponse {
    const message = {
      ...baseMsgStakeServicerResponse,
    } as MsgStakeServicerResponse;
    return message;
  },

  toJSON(_: MsgStakeServicerResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<MsgStakeServicerResponse>
  ): MsgStakeServicerResponse {
    const message = {
      ...baseMsgStakeServicerResponse,
    } as MsgStakeServicerResponse;
    return message;
  },
};

const baseMsgUnstakeServicer: object = { creator: "" };

export const MsgUnstakeServicer = {
  encode(
    message: MsgUnstakeServicer,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.spec !== undefined) {
      SpecName.encode(message.spec, writer.uint32(18).fork()).ldelim();
    }
    if (message.deadline !== undefined) {
      BlockNum.encode(message.deadline, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgUnstakeServicer {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgUnstakeServicer } as MsgUnstakeServicer;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.spec = SpecName.decode(reader, reader.uint32());
          break;
        case 3:
          message.deadline = BlockNum.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgUnstakeServicer {
    const message = { ...baseMsgUnstakeServicer } as MsgUnstakeServicer;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.spec !== undefined && object.spec !== null) {
      message.spec = SpecName.fromJSON(object.spec);
    } else {
      message.spec = undefined;
    }
    if (object.deadline !== undefined && object.deadline !== null) {
      message.deadline = BlockNum.fromJSON(object.deadline);
    } else {
      message.deadline = undefined;
    }
    return message;
  },

  toJSON(message: MsgUnstakeServicer): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.spec !== undefined &&
      (obj.spec = message.spec ? SpecName.toJSON(message.spec) : undefined);
    message.deadline !== undefined &&
      (obj.deadline = message.deadline
        ? BlockNum.toJSON(message.deadline)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgUnstakeServicer>): MsgUnstakeServicer {
    const message = { ...baseMsgUnstakeServicer } as MsgUnstakeServicer;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.spec !== undefined && object.spec !== null) {
      message.spec = SpecName.fromPartial(object.spec);
    } else {
      message.spec = undefined;
    }
    if (object.deadline !== undefined && object.deadline !== null) {
      message.deadline = BlockNum.fromPartial(object.deadline);
    } else {
      message.deadline = undefined;
    }
    return message;
  },
};

const baseMsgUnstakeServicerResponse: object = {};

export const MsgUnstakeServicerResponse = {
  encode(
    _: MsgUnstakeServicerResponse,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgUnstakeServicerResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgUnstakeServicerResponse,
    } as MsgUnstakeServicerResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): MsgUnstakeServicerResponse {
    const message = {
      ...baseMsgUnstakeServicerResponse,
    } as MsgUnstakeServicerResponse;
    return message;
  },

  toJSON(_: MsgUnstakeServicerResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<MsgUnstakeServicerResponse>
  ): MsgUnstakeServicerResponse {
    const message = {
      ...baseMsgUnstakeServicerResponse,
    } as MsgUnstakeServicerResponse;
    return message;
  },
};

const baseMsgProofOfWork: object = { creator: "", computeUnits: 0 };

export const MsgProofOfWork = {
  encode(message: MsgProofOfWork, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.spec !== undefined) {
      SpecName.encode(message.spec, writer.uint32(18).fork()).ldelim();
    }
    if (message.session !== undefined) {
      SessionID.encode(message.session, writer.uint32(26).fork()).ldelim();
    }
    if (message.clientRequest !== undefined) {
      ClientRequest.encode(
        message.clientRequest,
        writer.uint32(34).fork()
      ).ldelim();
    }
    if (message.workProof !== undefined) {
      WorkProof.encode(message.workProof, writer.uint32(42).fork()).ldelim();
    }
    if (message.computeUnits !== 0) {
      writer.uint32(48).uint64(message.computeUnits);
    }
    if (message.blockOfWork !== undefined) {
      BlockNum.encode(message.blockOfWork, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgProofOfWork {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgProofOfWork } as MsgProofOfWork;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.spec = SpecName.decode(reader, reader.uint32());
          break;
        case 3:
          message.session = SessionID.decode(reader, reader.uint32());
          break;
        case 4:
          message.clientRequest = ClientRequest.decode(reader, reader.uint32());
          break;
        case 5:
          message.workProof = WorkProof.decode(reader, reader.uint32());
          break;
        case 6:
          message.computeUnits = longToNumber(reader.uint64() as Long);
          break;
        case 7:
          message.blockOfWork = BlockNum.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgProofOfWork {
    const message = { ...baseMsgProofOfWork } as MsgProofOfWork;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.spec !== undefined && object.spec !== null) {
      message.spec = SpecName.fromJSON(object.spec);
    } else {
      message.spec = undefined;
    }
    if (object.session !== undefined && object.session !== null) {
      message.session = SessionID.fromJSON(object.session);
    } else {
      message.session = undefined;
    }
    if (object.clientRequest !== undefined && object.clientRequest !== null) {
      message.clientRequest = ClientRequest.fromJSON(object.clientRequest);
    } else {
      message.clientRequest = undefined;
    }
    if (object.workProof !== undefined && object.workProof !== null) {
      message.workProof = WorkProof.fromJSON(object.workProof);
    } else {
      message.workProof = undefined;
    }
    if (object.computeUnits !== undefined && object.computeUnits !== null) {
      message.computeUnits = Number(object.computeUnits);
    } else {
      message.computeUnits = 0;
    }
    if (object.blockOfWork !== undefined && object.blockOfWork !== null) {
      message.blockOfWork = BlockNum.fromJSON(object.blockOfWork);
    } else {
      message.blockOfWork = undefined;
    }
    return message;
  },

  toJSON(message: MsgProofOfWork): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.spec !== undefined &&
      (obj.spec = message.spec ? SpecName.toJSON(message.spec) : undefined);
    message.session !== undefined &&
      (obj.session = message.session
        ? SessionID.toJSON(message.session)
        : undefined);
    message.clientRequest !== undefined &&
      (obj.clientRequest = message.clientRequest
        ? ClientRequest.toJSON(message.clientRequest)
        : undefined);
    message.workProof !== undefined &&
      (obj.workProof = message.workProof
        ? WorkProof.toJSON(message.workProof)
        : undefined);
    message.computeUnits !== undefined &&
      (obj.computeUnits = message.computeUnits);
    message.blockOfWork !== undefined &&
      (obj.blockOfWork = message.blockOfWork
        ? BlockNum.toJSON(message.blockOfWork)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgProofOfWork>): MsgProofOfWork {
    const message = { ...baseMsgProofOfWork } as MsgProofOfWork;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.spec !== undefined && object.spec !== null) {
      message.spec = SpecName.fromPartial(object.spec);
    } else {
      message.spec = undefined;
    }
    if (object.session !== undefined && object.session !== null) {
      message.session = SessionID.fromPartial(object.session);
    } else {
      message.session = undefined;
    }
    if (object.clientRequest !== undefined && object.clientRequest !== null) {
      message.clientRequest = ClientRequest.fromPartial(object.clientRequest);
    } else {
      message.clientRequest = undefined;
    }
    if (object.workProof !== undefined && object.workProof !== null) {
      message.workProof = WorkProof.fromPartial(object.workProof);
    } else {
      message.workProof = undefined;
    }
    if (object.computeUnits !== undefined && object.computeUnits !== null) {
      message.computeUnits = object.computeUnits;
    } else {
      message.computeUnits = 0;
    }
    if (object.blockOfWork !== undefined && object.blockOfWork !== null) {
      message.blockOfWork = BlockNum.fromPartial(object.blockOfWork);
    } else {
      message.blockOfWork = undefined;
    }
    return message;
  },
};

const baseMsgProofOfWorkResponse: object = {};

export const MsgProofOfWorkResponse = {
  encode(_: MsgProofOfWorkResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgProofOfWorkResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgProofOfWorkResponse } as MsgProofOfWorkResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): MsgProofOfWorkResponse {
    const message = { ...baseMsgProofOfWorkResponse } as MsgProofOfWorkResponse;
    return message;
  },

  toJSON(_: MsgProofOfWorkResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgProofOfWorkResponse>): MsgProofOfWorkResponse {
    const message = { ...baseMsgProofOfWorkResponse } as MsgProofOfWorkResponse;
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  StakeServicer(request: MsgStakeServicer): Promise<MsgStakeServicerResponse>;
  UnstakeServicer(
    request: MsgUnstakeServicer
  ): Promise<MsgUnstakeServicerResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  ProofOfWork(request: MsgProofOfWork): Promise<MsgProofOfWorkResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  StakeServicer(request: MsgStakeServicer): Promise<MsgStakeServicerResponse> {
    const data = MsgStakeServicer.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Msg",
      "StakeServicer",
      data
    );
    return promise.then((data) =>
      MsgStakeServicerResponse.decode(new Reader(data))
    );
  }

  UnstakeServicer(
    request: MsgUnstakeServicer
  ): Promise<MsgUnstakeServicerResponse> {
    const data = MsgUnstakeServicer.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Msg",
      "UnstakeServicer",
      data
    );
    return promise.then((data) =>
      MsgUnstakeServicerResponse.decode(new Reader(data))
    );
  }

  ProofOfWork(request: MsgProofOfWork): Promise<MsgProofOfWorkResponse> {
    const data = MsgProofOfWork.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.servicer.Msg",
      "ProofOfWork",
      data
    );
    return promise.then((data) =>
      MsgProofOfWorkResponse.decode(new Reader(data))
    );
  }
}

interface Rpc {
  request(
    service: string,
    method: string,
    data: Uint8Array
  ): Promise<Uint8Array>;
}

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
