import { StakeEntry } from "../../grpc_web_services/lavanet/lava/epochstorage/stake_entry_pb";
import { Spec } from "../../grpc_web_services/lavanet/lava/spec/spec_pb";

export interface StateQuery {
  fetchPairing(): Promise<[number, number]>;
  getPairing(chainID: string): PairingResponse | undefined;
  getVirtualEpoch(): number;
  init(): Promise<void>;
}

export interface PairingResponse {
  providers: StakeEntry[];
  maxCu: number;
  currentEpoch: number;
  spec: Spec;
}
