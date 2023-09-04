import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";
import { ConsumerSessionsWithProvider } from "../lavasession/consumerTypes";
import { BaseChainParser } from "../chainlib/base_chain_parser";
import { Spec } from "../codec/lavanet/lava/spec/spec";
import { PairingResponse } from "../stateTracker/stateQuery/state_query";

type apiInterface = string;
class RPCConsumer {
  private consumerSessionManagerMap: Map<apiInterface, ConsumerSessionManager>;
  private chainParserMap: Map<apiInterface, BaseChainParser>;
  constructor(consumerSessionManager: ConsumerSessionManager) {
    this.consumerSessionManagerMap = new Map();
    this.chainParserMap = new Map();
  }

  static create(pairingResponse: PairingResponse): RPCConsumer {
    
  }

  updateAllProviders(pairingResponse: PairingResponse): Promise<Error | undefined> {

  }
}

type ChainId = string;

export class Consumer {
  private rpcConsumer: Map<ChainId, RPCConsumer>;
  constructor() {
    this.rpcConsumer = new Map();
  }

  public async updateAllProviders(
    pairingResponse: PairingResponse
  ): Promise<Error | undefined> {
    const chainId = pairingResponse.spec.index;
    const rpcConsumer = this.rpcConsumer.get(chainId);
    if (!rpcConsumer) {
      // initialize the rpcConsumer
      this.rpcConsumer.set(chainId, RPCConsumer.create(pairingResponse));
      return;
    }
    return rpcConsumer.updateAllProviders(pairingResponse);
  }
}
