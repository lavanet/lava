import { ConsumerSessionsWithProvider } from "./consumerTypes";
import { median } from "../util/common";

export class LatestBlockTracker {
  private latestBlock = -1;
  private providersBlocks: Map<ConsumerSessionsWithProvider, number> =
    new Map();

  public hasLatestBlock(session: ConsumerSessionsWithProvider) {
    return this.providersBlocks.has(session);
  }

  public setLatestBlockFor(
    session: ConsumerSessionsWithProvider,
    blockNumber: number
  ) {
    this.providersBlocks.set(session, blockNumber);
    this.latestBlock = Math.floor(
      median(Array.from(this.providersBlocks.values()))
    );
  }

  public getLatestBlock(): number {
    return this.latestBlock;
  }
}
