import { Metadata } from "../../../grpc_web_services/lavanet/lava/pairing/relay_pb";

export interface GenericMessage {
  getHeaders(): Metadata[];
}
