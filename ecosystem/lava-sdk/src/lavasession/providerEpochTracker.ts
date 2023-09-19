import { median } from "../util/common";

export class ProviderEpochTracker {
  private providersEpochs: Map<string, number> = new Map();
  private epoch = 0;

  public reset() {
    this.providersEpochs = new Map();
  }

  public hasEpochDataForProviderAddress(providersPublicAddress: string) {
    return this.providersEpochs.has(providersPublicAddress);
  }

  public setEpoch(providersPublicAddress: string, epoch: number) {
    this.providersEpochs.set(providersPublicAddress, epoch);
    this.epoch = Math.floor(median(Array.from(this.providersEpochs.values())));
  }

  public getEpoch(): number {
    return this.epoch; // median epoch from all provider probes.
  }

  public getProviderListSize(): number {
    return this.providersEpochs.size;
  }
}
