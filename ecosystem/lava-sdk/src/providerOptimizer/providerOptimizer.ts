import { LRUCache } from "lru-cache";
import {
  ProviderOptimizer as ProviderOptimizerInterface,
  QualityOfServiceReport,
} from "../lavasession/consumerTypes";
import { Logger } from "../logger/logger";

// TODO: move them somewhere
const hourInMillis = 60 * 60 * 1000;
function millisToSeconds(millis: number): number {
  return millis / 1000;
}

const CACHE_OPTIONS = {
  max: 2000,
};
const HALF_LIFE_TIME = hourInMillis;
const MAX_HALF_TIME = 14 * 24 * hourInMillis;
const PROBE_UPDATE_WEIGHT = 0.25;

// TODO: move this to utils/score/decayScore.ts
export class ScoreStore {
  public constructor(
    public readonly num: number,
    public readonly denom: number,
    public readonly time: number
  ) {}

  public static calculateTimeDecayFunctionUpdate(
    oldScore: ScoreStore,
    newScore: ScoreStore,
    halfLife: number,
    updateWeight: number,
    sampleTime: number
  ): ScoreStore {
    const oldDecayExponent =
      (Math.LN2 * millisToSeconds(sampleTime - oldScore.time)) /
      millisToSeconds(halfLife);
    const oldDecayFactor = Math.exp(-oldDecayExponent);
    const newDecayExponent =
      (Math.LN2 * millisToSeconds(sampleTime - newScore.time)) /
      millisToSeconds(halfLife);
    const newDecayFactor = Math.exp(-newDecayExponent);
    const updatedNum =
      oldScore.num * oldDecayFactor +
      newScore.num * newDecayFactor * updateWeight;
    const updatedDenom =
      oldScore.denom * oldDecayFactor +
      newScore.denom * newDecayFactor * updateWeight;
    return new ScoreStore(updatedNum, updatedDenom, sampleTime);
  }
}

export interface ProviderData {
  availability: ScoreStore;
  latency: ScoreStore;
  sync: ScoreStore;
  syncBlock: number;
}

export enum Strategy {
  Balanced,
  Latency,
  SyncFreshness,
  Cost,
  Privacy,
  Accuracy,
}

export class ProviderOptimizer implements ProviderOptimizerInterface {
  private readonly strategy: Strategy;
  private readonly providersStorage = new LRUCache<string, ProviderData>(
    CACHE_OPTIONS
  ); // todo: ristretto.Cache in go (see what it does)
  private readonly providerRelayStats = new LRUCache<string, number[]>(
    CACHE_OPTIONS
  ); // todo: ristretto.Cache in go (see what it does)
  private readonly averageBlockTime: number;
  private readonly baseWorldLatency: number;
  private readonly wantedNumProvidersInConcurrency: number; // todo: see if this is actually needed
  private readonly latestSyncData: any; // todo: ConcurrentBlockStore in go

  public constructor(
    strategy: Strategy,
    averageBlockTime: number,
    baseWorldLatency: number,
    wantedNumProvidersInConcurrency: number
  ) {
    this.strategy = strategy;
    this.averageBlockTime = averageBlockTime;
    this.baseWorldLatency = baseWorldLatency;

    if (strategy === Strategy.Privacy) {
      wantedNumProvidersInConcurrency = 1;
    }

    this.wantedNumProvidersInConcurrency = wantedNumProvidersInConcurrency;
  }

  public appendProbeRelayData(
    providerAddress: string,
    latency: number,
    success: boolean
  ): void {
    let providerData = this.getProviderData(providerAddress);
    const sampleTime = performance.now();
    const halfTime = this.calculateHalfTime(providerAddress);

    providerData = this.updateProbeEntryAvailability(
      providerData,
      success,
      PROBE_UPDATE_WEIGHT,
      halfTime,
      sampleTime
    );

    if (success && latency > 0) {
      providerData = this.updateProbeEntryLatency(
        providerData,
        latency,
        this.baseWorldLatency,
        PROBE_UPDATE_WEIGHT,
        halfTime,
        sampleTime
      );
    }

    this.providersStorage.set(providerAddress, providerData);
    Logger.debug("probe update", providerAddress, latency, success);
  }

  public appendRelayData(
    providerAddress: string,
    latency: number,
    isHangingApi: boolean,
    cu: number,
    syncBlock: number
  ): void {
    console.log("appendRelayData");
  }

  public appendRelayFailure(providerAddress: string): void {
    console.log("appendRelayFailure");
  }

  public chooseProvider(
    allAddresses: string[],
    ignoredProviders: string[],
    cu: number,
    requestedBlock: number,
    perturbationPercentage: number
  ): string[] {
    return [];
  }

  public getExcellenceQoSReportForProvider(
    providerAddress: string
  ): QualityOfServiceReport {
    return {} as unknown as QualityOfServiceReport;
  }

  public calculateProbabilityOfTimeout(
    availabilityScore: ScoreStore /* score.ScoreStore */
  ): number {
    const probabilityTimeout = 0;

    if (availabilityScore.denom > 0) {
      const mean = availabilityScore.num / availabilityScore.denom;
      return 1 - mean;
    }

    return probabilityTimeout;
  }

  public calculateProbabilityOfBlockError(
    requestedBlock: number,
    providerData: ProviderData /* ProviderData */
  ): number {
    let probabilityBlockError = 0;

    if (
      requestedBlock > 0 &&
      providerData.syncBlock < requestedBlock &&
      providerData.syncBlock > 0
    ) {
      const averageBlockTime = millisToSeconds(this.averageBlockTime);
      const blockDistanceRequired = requestedBlock - providerData.syncBlock;
      if (blockDistanceRequired > 0) {
        const timeSinceSyncReceived = millisToSeconds(
          performance.now() - providerData.sync.time
        );
        const eventRate = timeSinceSyncReceived / averageBlockTime;
        probabilityBlockError = cumulativeProbabilityFunctionForPoissonDist(
          blockDistanceRequired - 1,
          eventRate
        );
      } else {
        probabilityBlockError = 0;
      }
    }

    return probabilityBlockError;
  }

  private getProviderData(providerAddress: string): ProviderData {
    let data = this.providersStorage.get(providerAddress);
    if (data === undefined) {
      data = {
        availability: new ScoreStore(0.99, 1, performance.now() - hourInMillis),
        latency: new ScoreStore(2, 1, performance.now() - hourInMillis),
        sync: new ScoreStore(2, 1, performance.now() - hourInMillis),
        syncBlock: 0,
      };
    }
    return data;
  }

  private updateProbeEntryAvailability(
    providerData: ProviderData,
    success: boolean,
    weight: number,
    halfTime: number,
    sampleTime: number
  ): ProviderData {
    const newNumerator = Number(success);
    const oldScore = providerData.availability;
    const newScore = new ScoreStore(newNumerator, 1, sampleTime);
    providerData.availability = ScoreStore.calculateTimeDecayFunctionUpdate(
      oldScore,
      newScore,
      halfTime,
      weight,
      sampleTime
    );
    return providerData;
  }

  private updateProbeEntryLatency(
    providerData: ProviderData,
    latency: number,
    baseLatency: number,
    weight: number,
    halfTime: number,
    sampleTime: number
  ): ProviderData {
    const newScore = new ScoreStore(
      millisToSeconds(latency),
      millisToSeconds(baseLatency),
      sampleTime
    );
    const oldScore = providerData.latency;
    providerData.latency = ScoreStore.calculateTimeDecayFunctionUpdate(
      oldScore,
      newScore,
      halfTime,
      weight,
      sampleTime
    );
    return providerData;
  }

  private calculateHalfTime(providerAddress: string): number {
    const relaysHalfTime = this.getRelayStatsTimeDiff(providerAddress);
    let halfTime = HALF_LIFE_TIME;

    if (relaysHalfTime > halfTime) {
      halfTime = relaysHalfTime;
    }

    if (halfTime > MAX_HALF_TIME) {
      halfTime = MAX_HALF_TIME;
    }

    return halfTime;
  }

  private getRelayStatsTimeDiff(providerAddress: string): number {
    const relayStatsTime = this.getRelayStatsTime(providerAddress);
    if (relayStatsTime.length === 0) {
      return 0;
    }
    const idx = Math.floor((relayStatsTime.length - 1) / 2);
    return performance.now() - relayStatsTime[idx];
  }

  private getRelayStatsTime(providerAddress: string): number[] {
    return this.providerRelayStats.get(providerAddress) || [];
  }
}

function cumulativeProbabilityFunctionForPoissonDist(
  kEvents: number,
  lambda: number
): number {
  // TODO: regularized incomplete Gamma integral implementation
  return 1 - Math.random();
}
