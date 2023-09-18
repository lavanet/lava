import { shuffleArray } from "../util/arrays";
import { ProviderOptimizer } from "./consumerTypes";
import { Logger } from "../logger/logger";
export class RandomProviderOptimizer implements ProviderOptimizer {
  public chooseProvider(
    allAddresses: string[],
    ignoredProviders: string[],
    cu: number,
    requestedBlock: number,
    perturbationPercentage: number
  ): string[] {
    const returnProviders = allAddresses
      .map((address) => {
        if (ignoredProviders.includes(address)) {
          return;
        }
        return address;
      })
      .filter(Boolean) as string[];

    const provider = shuffleArray(returnProviders)[0];
    return provider ? [provider] : [];
  }

  public appendProbeRelayData(
    providerAddress: string,
    latency: number,
    success: boolean
  ) {
    Logger.warn(
      "RandomProviderOptimizer.appendProbeRelayData() not implemented"
    );
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
    // Logger.warn("RandomProviderOptimizer.appendRelayFailure() not implemented");
    return;
  }

  public calculateProbabilityOfTimeout(
    availabilityScoreStore: ScoreStore
  ): number {
    Logger.warn(
      "RandomProviderOptimizer.calculateProbabilityOfTimeout() not implemented"
    );
    return 0;
  }

  public calculateProbabilityOfBlockError(
    requestedBlock: number,
    providerData: ProviderData
  ): number {
    Logger.warn(
      "RandomProviderOptimizer.calculateProbabilityOfBlockError() not implemented"
    );
    return 0;
  }

  public getExcellenceQoSReportForProvider(providerAddress: string): any {
    Logger.warn(
      "RandomProviderOptimizer.getExcellenceQoSReportForProvider() not implemented"
    );
    return;
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
