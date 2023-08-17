import { shuffleArray } from "../util/arrays";
import { ProviderOptimizer } from "./consumerTypes";

export class RandomProviderOptimizer implements ProviderOptimizer {
  public chooseProvider(
    allAddresses: string[],
    ignoredProviders: string[],
    cu: number,
    requestedBlock: number,
    perturbationPercentage: number
  ): string[] {
    return shuffleArray(allAddresses);
  }

  public appendRelayData(
    providerAddress: string,
    latency: number,
    isHangingApi: boolean,
    cu: number,
    syncBlock: number
  ) {
    return;
  }

  public appendRelayFailure(providerAddress: string) {
    console.log("RandomProviderOptimizer.appendRelayFailure() not implemented");
    return;
  }

  public calculateProbabilityOfTimeout(
    availabilityScoreStore: ScoreStore
  ): number {
    console.log(
      "RandomProviderOptimizer.calculateProbabilityOfTimeout() not implemented"
    );
    return 0;
  }

  public calculateProbabilityOfBlockError(
    requestedBlock: number,
    providerData: ProviderData
  ): number {
    console.log(
      "RandomProviderOptimizer.calculateProbabilityOfBlockError() not implemented"
    );
    return 0;
  }

  public getExcellenceQoSReportForProvider(providerAddress: string): any {
    throw new Error("Not implemented");
  }
}

interface ProviderData {
  availability: ScoreStore;
  latency: ScoreStore;
  sync: ScoreStore;
  syncBlock: number;
}

interface ScoreStore {
  num: number;
  denom: number;
  time: number;
}
