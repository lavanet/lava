export class ConsumerConsistency {
  private specId: string;
  private latestSeen: number;
  constructor(specId: string) {
    this.specId = specId;
    this.latestSeen = 0;
  }

  private setLatestBlock(block: number) {
    this.latestSeen = block;
  }

  public getSeenBlock(): number {
    return this.latestSeen;
  }

  public setSeenBlock(block: number) {
    if (this.getSeenBlock() < block) {
      this.setLatestBlock(block);
    }
  }
}
