import { PairingResponse } from "../stateTracker/stateQuery/state_query";
import { Logger } from "../logger/logger";
import { Relayer } from "../relayer/relayer";
import {
  RPCConsumer,
  SendRelayOptions,
  SendRestRelayOptions,
} from "./rpc_consumer";

type ChainId = string;

export class Consumer {
  private rpcConsumer: Map<ChainId, RPCConsumer>;
  private geolocation: string;
  private relayer: Relayer;
  constructor(relayer: Relayer, geolocation: string) {
    this.rpcConsumer = new Map();
    this.geolocation = geolocation;
    this.relayer = relayer;
  }

  public async updateAllProviders(pairingResponse: PairingResponse) {
    const chainId = pairingResponse.spec.index;
    let rpcConsumer = this.rpcConsumer.get(chainId);
    if (!rpcConsumer) {
      // initialize the rpcConsumer
      rpcConsumer = await RPCConsumer.create(
        pairingResponse,
        this.geolocation,
        this.relayer
      );
      this.rpcConsumer.set(chainId, rpcConsumer);
      return;
    }
    rpcConsumer.setSpec(pairingResponse.spec);
    rpcConsumer.updateAllProviders(pairingResponse);
  }

  public async sendRelay(
    relayOptions: SendRelayOptions | SendRestRelayOptions
  ) {
    if (relayOptions.chainId) {
      // if we have a chainId, field validate we have the RPC consumer instance
      const rpcConsumer = this.rpcConsumer.get(relayOptions.chainId);
      if (!rpcConsumer) {
        throw Logger.fatal(
          "Missing chainId in RPC Consumer map, validate sdk initialization",
          "Available Consumers:",
          this.rpcConsumer.keys()
        );
      }
    }
  }
}
