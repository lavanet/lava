import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";
import { ConsumerSessionsWithProvider } from "../lavasession/consumerTypes";
import { BaseChainParser } from '../chainlib/base_chain_parser';
import { Spec } from "../codec/lavanet/lava/spec/spec";

export class Consumer {
  private consumerSessionManager: ConsumerSessionManager;
  private chainParser: BaseChainParser;
  constructor(consumerSessionManager: ConsumerSessionManager) {
    this.consumerSessionManager = consumerSessionManager;
    this.chainParser = new BaseChainParser();
  }

  public async updateAllProviders(
    epoch: number,
    pairingList: ConsumerSessionsWithProvider[]
  ): Promise<Error | undefined> {
    return this.consumerSessionManager.updateAllProviders(epoch, pairingList);
  }

  public SetSpec(spec: Spec)
}
