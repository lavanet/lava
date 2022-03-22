/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { SpecName } from "../user/spec_name";
import { Coin } from "../cosmos/base/v1beta1/coin";
import { BlockNum } from "../user/block_num";

export const protobufPackage = "lavanet.lava.user";

export interface MsgStakeUser {
  creator: string;
  spec: SpecName | undefined;
  amount: Coin | undefined;
  deadline: BlockNum | undefined;
}

export interface MsgStakeUserResponse {}

export interface MsgUnstakeUser {
  creator: string;
  spec: SpecName | undefined;
  deadline: BlockNum | undefined;
}

export interface MsgUnstakeUserResponse {}

const baseMsgStakeUser: object = { creator: "" };

export const MsgStakeUser = {
  encode(message: MsgStakeUser, writer: Writer = Writer.create()): Writer {
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
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgStakeUser {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgStakeUser } as MsgStakeUser;
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
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgStakeUser {
    const message = { ...baseMsgStakeUser } as MsgStakeUser;
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
    return message;
  },

  toJSON(message: MsgStakeUser): unknown {
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
    return obj;
  },

  fromPartial(object: DeepPartial<MsgStakeUser>): MsgStakeUser {
    const message = { ...baseMsgStakeUser } as MsgStakeUser;
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
    return message;
  },
};

const baseMsgStakeUserResponse: object = {};

export const MsgStakeUserResponse = {
  encode(_: MsgStakeUserResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgStakeUserResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgStakeUserResponse } as MsgStakeUserResponse;
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

  fromJSON(_: any): MsgStakeUserResponse {
    const message = { ...baseMsgStakeUserResponse } as MsgStakeUserResponse;
    return message;
  },

  toJSON(_: MsgStakeUserResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgStakeUserResponse>): MsgStakeUserResponse {
    const message = { ...baseMsgStakeUserResponse } as MsgStakeUserResponse;
    return message;
  },
};

const baseMsgUnstakeUser: object = { creator: "" };

export const MsgUnstakeUser = {
  encode(message: MsgUnstakeUser, writer: Writer = Writer.create()): Writer {
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

  decode(input: Reader | Uint8Array, length?: number): MsgUnstakeUser {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgUnstakeUser } as MsgUnstakeUser;
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

  fromJSON(object: any): MsgUnstakeUser {
    const message = { ...baseMsgUnstakeUser } as MsgUnstakeUser;
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

  toJSON(message: MsgUnstakeUser): unknown {
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

  fromPartial(object: DeepPartial<MsgUnstakeUser>): MsgUnstakeUser {
    const message = { ...baseMsgUnstakeUser } as MsgUnstakeUser;
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

const baseMsgUnstakeUserResponse: object = {};

export const MsgUnstakeUserResponse = {
  encode(_: MsgUnstakeUserResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgUnstakeUserResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgUnstakeUserResponse } as MsgUnstakeUserResponse;
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

  fromJSON(_: any): MsgUnstakeUserResponse {
    const message = { ...baseMsgUnstakeUserResponse } as MsgUnstakeUserResponse;
    return message;
  },

  toJSON(_: MsgUnstakeUserResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgUnstakeUserResponse>): MsgUnstakeUserResponse {
    const message = { ...baseMsgUnstakeUserResponse } as MsgUnstakeUserResponse;
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  StakeUser(request: MsgStakeUser): Promise<MsgStakeUserResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  UnstakeUser(request: MsgUnstakeUser): Promise<MsgUnstakeUserResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  StakeUser(request: MsgStakeUser): Promise<MsgStakeUserResponse> {
    const data = MsgStakeUser.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.user.Msg",
      "StakeUser",
      data
    );
    return promise.then((data) =>
      MsgStakeUserResponse.decode(new Reader(data))
    );
  }

  UnstakeUser(request: MsgUnstakeUser): Promise<MsgUnstakeUserResponse> {
    const data = MsgUnstakeUser.encode(request).finish();
    const promise = this.rpc.request(
      "lavanet.lava.user.Msg",
      "UnstakeUser",
      data
    );
    return promise.then((data) =>
      MsgUnstakeUserResponse.decode(new Reader(data))
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
