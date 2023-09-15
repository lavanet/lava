// package: lavanet.lava.pairing
// file: lavanet/lava/pairing/params.proto

import * as jspb from "google-protobuf";
import * as gogoproto_gogo_pb from "../../../gogoproto/gogo_pb";

export class Params extends jspb.Message {
  getMintcoinspercu(): string;
  setMintcoinspercu(value: string): void;

  getFraudstakeslashingfactor(): string;
  setFraudstakeslashingfactor(value: string): void;

  getFraudslashingamount(): number;
  setFraudslashingamount(value: number): void;

  getEpochblocksoverlap(): number;
  setEpochblocksoverlap(value: number): void;

  getUnpaylimit(): string;
  setUnpaylimit(value: string): void;

  getSlashlimit(): string;
  setSlashlimit(value: string): void;

  getDatareliabilityreward(): string;
  setDatareliabilityreward(value: string): void;

  getQosweight(): string;
  setQosweight(value: string): void;

  getRecommendedepochnumtocollectpayment(): number;
  setRecommendedepochnumtocollectpayment(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Params.AsObject;
  static toObject(includeInstance: boolean, msg: Params): Params.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Params, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Params;
  static deserializeBinaryFromReader(message: Params, reader: jspb.BinaryReader): Params;
}

export namespace Params {
  export type AsObject = {
    mintcoinspercu: string,
    fraudstakeslashingfactor: string,
    fraudslashingamount: number,
    epochblocksoverlap: number,
    unpaylimit: string,
    slashlimit: string,
    datareliabilityreward: string,
    qosweight: string,
    recommendedepochnumtocollectpayment: number,
  }
}

