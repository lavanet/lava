// package: lavanet.lava.subscription
// file: lavanet/lava/subscription/subscription.proto

import * as jspb from "google-protobuf";

export class Subscription extends jspb.Message {
  getCreator(): string;
  setCreator(value: string): void;

  getConsumer(): string;
  setConsumer(value: string): void;

  getBlock(): number;
  setBlock(value: number): void;

  getPlanIndex(): string;
  setPlanIndex(value: string): void;

  getPlanBlock(): number;
  setPlanBlock(value: number): void;

  getDurationBought(): number;
  setDurationBought(value: number): void;

  getDurationLeft(): number;
  setDurationLeft(value: number): void;

  getMonthExpiryTime(): number;
  setMonthExpiryTime(value: number): void;

  getMonthCuTotal(): number;
  setMonthCuTotal(value: number): void;

  getMonthCuLeft(): number;
  setMonthCuLeft(value: number): void;

  getCluster(): string;
  setCluster(value: string): void;

  getDurationTotal(): number;
  setDurationTotal(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Subscription.AsObject;
  static toObject(includeInstance: boolean, msg: Subscription): Subscription.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Subscription, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Subscription;
  static deserializeBinaryFromReader(message: Subscription, reader: jspb.BinaryReader): Subscription;
}

export namespace Subscription {
  export type AsObject = {
    creator: string,
    consumer: string,
    block: number,
    planIndex: string,
    planBlock: number,
    durationBought: number,
    durationLeft: number,
    monthExpiryTime: number,
    monthCuTotal: number,
    monthCuLeft: number,
    cluster: string,
    durationTotal: number,
  }
}

