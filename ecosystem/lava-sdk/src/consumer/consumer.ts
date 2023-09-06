import { PairingResponse } from "../stateTracker/stateQuery/state_query";
import { Logger } from "../logger/logger";
import { Relayer } from "../relayer/relayer";
import { RPCConsumer } from "./rpc_consumer";

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

  public async sendRelay() {
    
  }
}
