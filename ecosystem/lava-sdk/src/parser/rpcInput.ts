import { Metadata } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";

export interface RPCInput {
  GetParams(): any;
  GetResult(): Uint8Array;
  ParseBlock(block: string): number | Error;
  GetHeaders(): Metadata[];
}
