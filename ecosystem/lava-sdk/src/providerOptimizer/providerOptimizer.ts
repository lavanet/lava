import { LRUCache } from "lru-cache";
import { ProviderOptimizer as ProviderOptimizerInterface } from "../lavasession/consumerTypes";
import { Logger } from "../logger/logger";
import random from "random";
import gammainc from "@stdlib/math-base-special-gammainc";
import {
  AverageWorldLatency,
  baseTimePerCU,
  getTimePerCu,
} from "../common/timeout";
import BigNumber from "bignumber.js";
import { hourInMillis, millisToSeconds, now } from "../util/time";
import { ScoreStore } from "../util/score/decayScore";
import { QualityOfServiceReport } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";

const CACHE_OPTIONS = {
  max: 2000,
};
const HALF_LIFE_TIME = hourInMillis;
const MAX_HALF_TIME = 14 * 24 * hourInMillis;
const PROBE_UPDATE_WEIGHT = 0.25;
const RELAY_UPDATE_WEIGHT = 1;
const INITIAL_DATA_STALENESS = 24;
export const FLOAT_PRECISION = 8;
export const DECIMAL_PRECISION = 36;
export const DEFAULT_EXPLORATION_CHANCE = 0.1;
export const COST_EXPLORATION_CHANCE = 0.01;

export interface ProviderData {
  availability: ScoreStore;
  latency: ScoreStore;
  sync: ScoreStore;
  syncBlock: number;
}

export interface BlockStore {
  block: number;
  time: number;
}

export enum ProviderOptimizerStrategy {
  Balanced,
  Latency,
  SyncFreshness,
  Cost,
  Privacy,
  Accuracy,
}

export class ProviderOptimizer implements ProviderOptimizerInterface {
  private readonly strategy: ProviderOptimizerStrategy;
  private readonly providersStorage = new LRUCache<string, ProviderData>(
    CACHE_OPTIONS
  );
  private readonly providerRelayStats = new LRUCache<string, number[]>(
    CACHE_OPTIONS
  );
  private readonly averageBlockTime: number;
  private readonly baseWorldLatency: number;
  private readonly wantedNumProvidersInConcurrency: number;
  private readonly latestSyncData: BlockStore = {
    block: 0,
    time: 0,
  };

  public constructor(
    strategy: ProviderOptimizerStrategy,
    averageBlockTime: number,
    baseWorldLatency: number,
    wantedNumProvidersInConcurrency: number
  ) {
    if (averageBlockTime <= 0) {
      throw new Error("averageBlockTime must be higher than 0");
    }

    this.strategy = strategy;
    this.averageBlockTime = averageBlockTime;
    this.baseWorldLatency = baseWorldLatency;

    if (strategy === ProviderOptimizerStrategy.Privacy) {
      wantedNumProvidersInConcurrency = 1;
    }

    this.wantedNumProvidersInConcurrency = wantedNumProvidersInConcurrency;
  }

  public getStrategy(): ProviderOptimizerStrategy {
    return this.strategy;
  }

  public appendProbeRelayData(
    providerAddress: string,
    latency: number,
    success: boolean
  ): void {
    let { providerData } = this.getProviderData(providerAddress);
    const sampleTime = now();
    const halfTime = this.calculateHalfTime(providerAddress, sampleTime);

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
    this.appendRelay(
      providerAddress,
      latency,
      isHangingApi,
      true,
      cu,
      syncBlock,
      now()
    );
  }

  public appendRelayFailure(providerAddress: string): void {
    this.appendRelay(providerAddress, 0, false, false, 0, 0, now());
  }

  private appendRelay(
    providerAddress: string,
    latency: number,
    isHangingApi: boolean,
    success: boolean,
    cu: number,
    syncBlock: number,
    sampleTime: number
  ) {
    const { block, time } = this.updateLatestSyncData(syncBlock, sampleTime);
    let { providerData } = this.getProviderData(providerAddress);
    const halfTime = this.calculateHalfTime(providerAddress, sampleTime);

    providerData = this.updateProbeEntryAvailability(
      providerData,
      success,
      RELAY_UPDATE_WEIGHT,
      halfTime,
      sampleTime
    );
    if (success) {
      if (latency > 0) {
        let baseLatency = this.baseWorldLatency + baseTimePerCU(cu) / 2;
        if (isHangingApi) {
          baseLatency += this.averageBlockTime / 2;
        }
        providerData = this.updateProbeEntryLatency(
          providerData,
          latency,
          baseLatency,
          RELAY_UPDATE_WEIGHT,
          halfTime,
          sampleTime
        );
      }

      if (syncBlock > providerData.syncBlock) {
        providerData.syncBlock = syncBlock;
      }

      const syncLag = this.calculateSyncLag(
        block,
        time,
        providerData.syncBlock,
        sampleTime
      );
      providerData = this.updateProbeEntrySync(
        providerData,
        syncLag,
        this.averageBlockTime,
        halfTime,
        sampleTime
      );
    }

    this.providersStorage.set(providerAddress, providerData);
    this.updateRelayTime(providerAddress, sampleTime);
    Logger.debug(
      "relay update",
      syncBlock,
      cu,
      providerAddress,
      latency,
      success
    );
  }

  public chooseProvider(
    allAddresses: Set<string>,
    ignoredProviders: Set<string>,
    cu: number,
    requestedBlock: number,
    perturbationPercentage: number
  ): string[] {
    const returnedProviders: string[] = [""];
    let latencyScore = Number.MAX_VALUE;
    let syncScore = Number.MAX_VALUE;
    const numProviders = allAddresses.size;

    for (const providerAddress of allAddresses) {
      if (ignoredProviders.has(providerAddress)) {
        continue;
      }

      const { providerData } = this.getProviderData(providerAddress);
      let latencyScoreCurrent = this.calculateLatencyScore(
        providerData,
        cu,
        requestedBlock
      );
      latencyScoreCurrent = perturbWithNormalGaussian(
        latencyScoreCurrent,
        perturbationPercentage
      );

      let syncScoreCurrent = 0;
      if (requestedBlock < 0) {
        syncScoreCurrent = this.calculateSyncScore(providerData.sync);
        syncScoreCurrent = perturbWithNormalGaussian(
          syncScoreCurrent,
          perturbationPercentage
        );
      }

      // Logger.debug(
      //   "scores information",
      //   JSON.stringify({
      //     providerAddress,
      //     latencyScoreCurrent,
      //     syncScoreCurrent,
      //     latencyScore,
      //     syncScore,
      //   })
      // );

      const isBetterProviderScore = this.isBetterProviderScore(
        latencyScore,
        latencyScoreCurrent,
        syncScore,
        syncScoreCurrent
      );
      if (isBetterProviderScore || returnedProviders.length === 0) {
        if (
          returnedProviders[0] !== "" &&
          this.shouldExplore(returnedProviders.length, numProviders)
        ) {
          returnedProviders.push(returnedProviders[0]);
        }

        returnedProviders[0] = providerAddress;
        latencyScore = latencyScoreCurrent;
        syncScore = syncScoreCurrent;

        continue;
      }

      if (this.shouldExplore(returnedProviders.length, numProviders)) {
        returnedProviders.push(providerAddress);
      }
    }

    Logger.debug(
      "returned providers",
      JSON.stringify({
        providers: returnedProviders.join(","),
        cu,
      })
    );

    return returnedProviders;
  }

  public getExcellenceQoSReportForProvider(
    providerAddress: string
  ): QualityOfServiceReport | undefined {
    const { providerData, found } = this.getProviderData(providerAddress);
    if (!found) {
      return;
    }

    const latencyScore = floatToBigNumber(
      providerData.latency.num / providerData.latency.denom,
      FLOAT_PRECISION
    );
    const syncScore = floatToBigNumber(
      providerData.sync.num / providerData.sync.denom,
      FLOAT_PRECISION
    );
    const availabilityScore = floatToBigNumber(
      providerData.availability.num / providerData.availability.denom,
      FLOAT_PRECISION
    );
    const report: QualityOfServiceReport = new QualityOfServiceReport();
    report.setLatency(latencyScore.toFixed());
    report.setAvailability(availabilityScore.toFixed());
    report.setSync(syncScore.toFixed());

    // Logger.debug(
    //   "QoS excellence for provider",
    //   JSON.stringify({
    //     providerAddress,
    //     report,
    //   })
    // );

    return report;
  }

  public calculateProbabilityOfTimeout(availabilityScore: ScoreStore): number {
    const probabilityTimeout = 0;

    if (availabilityScore.denom > 0) {
      const mean = availabilityScore.num / availabilityScore.denom;
      return 1 - mean;
    }

    return probabilityTimeout;
  }

  public calculateProbabilityOfBlockError(
    requestedBlock: number,
    providerData: ProviderData
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
          now() - providerData.sync.time
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

  private shouldExplore(
    currentNumProviders: number,
    numProviders: number
  ): boolean {
    if (currentNumProviders >= this.wantedNumProvidersInConcurrency) {
      return false;
    }

    let explorationChance = DEFAULT_EXPLORATION_CHANCE;
    switch (this.strategy) {
      case ProviderOptimizerStrategy.Latency:
        return true;
      case ProviderOptimizerStrategy.Accuracy:
        return true;
      case ProviderOptimizerStrategy.Cost:
        explorationChance = COST_EXPLORATION_CHANCE;
        break;
      case ProviderOptimizerStrategy.Privacy:
        return false;
    }

    return random.float() < explorationChance / numProviders;
  }

  private isBetterProviderScore(
    latencyScore: number,
    latencyScoreCurrent: number,
    syncScore: number,
    syncScoreCurrent: number
  ): boolean {
    let latencyWeight: number;
    switch (this.strategy) {
      case ProviderOptimizerStrategy.Latency:
        latencyWeight = 0.7;
        break;
      case ProviderOptimizerStrategy.SyncFreshness:
        latencyWeight = 0.2;
        break;
      case ProviderOptimizerStrategy.Privacy:
        return random.int(0, 2) === 0;
      default:
        latencyWeight = 0.6;
    }

    if (syncScoreCurrent === 0) {
      return latencyScore > latencyScoreCurrent;
    }

    return (
      latencyScore * latencyWeight + syncScore * (1 - latencyWeight) >
      latencyScoreCurrent * latencyWeight +
        syncScoreCurrent * (1 - latencyWeight)
    );
  }

  private calculateSyncScore(syncScore: ScoreStore): number {
    let historicalSyncLatency = 0;
    if (syncScore.denom === 0) {
      historicalSyncLatency = 0;
    } else {
      historicalSyncLatency =
        (syncScore.num / syncScore.denom) * this.averageBlockTime;
    }

    return millisToSeconds(historicalSyncLatency);
  }

  private calculateLatencyScore(
    providerData: ProviderData,
    cu: number,
    requestedBlock: number
  ): number {
    const baseLatency = this.baseWorldLatency + baseTimePerCU(cu) / 2;
    const timeoutDuration = AverageWorldLatency + getTimePerCu(cu);

    let historicalLatency = 0;
    if (providerData.latency.denom === 0) {
      historicalLatency = baseLatency;
    } else {
      historicalLatency =
        (baseLatency * providerData.latency.num) / providerData.latency.denom;
    }

    if (historicalLatency > timeoutDuration) {
      historicalLatency = timeoutDuration;
    }

    const probabilityBlockError = this.calculateProbabilityOfBlockError(
      requestedBlock,
      providerData
    );
    const probabilityOfTimeout = this.calculateProbabilityOfTimeout(
      providerData.availability
    );
    const probabilityOfSuccess =
      (1 - probabilityBlockError) * (1 - probabilityOfTimeout);

    const historicalLatencySeconds = millisToSeconds(historicalLatency);
    const baseLatencySeconds = millisToSeconds(baseLatency);
    let costBlockError = historicalLatencySeconds + baseLatencySeconds;
    if (probabilityBlockError > 0.5) {
      costBlockError *= 3; // consistency improvement
    }
    const costTimeout = millisToSeconds(timeoutDuration) + baseLatencySeconds;
    const costSuccess = historicalLatencySeconds;

    // Logger.debug(
    //   "latency calculation breakdown",
    //   JSON.stringify({
    //     probabilityBlockError,
    //     costBlockError,
    //     probabilityOfTimeout,
    //     costTimeout,
    //     probabilityOfSuccess,
    //     costSuccess,
    //   })
    // );

    return (
      probabilityBlockError * costBlockError +
      probabilityOfTimeout * costTimeout +
      probabilityOfSuccess * costSuccess
    );
  }

  private calculateSyncLag(
    latestSync: number,
    timeSync: number,
    providerBlock: number,
    sampleTime: number
  ): number {
    if (latestSync <= providerBlock) {
      return 0;
    }

    const timeLag = sampleTime - timeSync;
    const firstBlockLag = Math.min(this.averageBlockTime, timeLag);
    const blocksGap = latestSync - providerBlock - 1;
    const blocksGapTime = blocksGap * this.averageBlockTime;

    return firstBlockLag + blocksGapTime;
  }

  private updateLatestSyncData(
    providerLatestBlock: number,
    sampleTime: number
  ): BlockStore {
    const latestBlock = this.latestSyncData.block;

    if (latestBlock < providerLatestBlock) {
      this.latestSyncData.block = providerLatestBlock;
      this.latestSyncData.time = sampleTime;
    }

    return this.latestSyncData;
  }

  private getProviderData(providerAddress: string): {
    providerData: ProviderData;
    found: boolean;
  } {
    let data = this.providersStorage.get(providerAddress);
    const found = data !== undefined;
    if (!found) {
      const time = -1 * INITIAL_DATA_STALENESS * hourInMillis;
      data = {
        availability: new ScoreStore(0.99, 1, now() + time),
        latency: new ScoreStore(1, 1, now() + time),
        sync: new ScoreStore(1, 1, now() + time),
        syncBlock: 0,
      };
    }
    return { providerData: data as ProviderData, found };
  }

  private updateProbeEntrySync(
    providerData: ProviderData,
    sync: number,
    baseSync: number,
    halfTime: number,
    sampleTime: number
  ): ProviderData {
    const newScore = new ScoreStore(
      millisToSeconds(sync),
      millisToSeconds(baseSync),
      sampleTime
    );
    const oldScore = providerData.sync;
    providerData.sync = ScoreStore.calculateTimeDecayFunctionUpdate(
      oldScore,
      newScore,
      halfTime,
      RELAY_UPDATE_WEIGHT,
      sampleTime
    );
    return providerData;
  }

  private updateProbeEntryAvailability(
    providerData: ProviderData,
    success: boolean,
    weight: number,
    halfTime: number,
    sampleTime: number
  ): ProviderData {
    const newNumerator = Number(success); // true = 1, false = 0
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

  private updateRelayTime(providerAddress: string, sampleTime: number): void {
    const relayStatsTime = this.getRelayStatsTime(providerAddress);
    relayStatsTime.push(sampleTime);
    this.providerRelayStats.set(providerAddress, relayStatsTime);
  }

  private calculateHalfTime(
    providerAddress: string,
    sampleTime: number
  ): number {
    const relaysHalfTime = this.getRelayStatsTimeDiff(
      providerAddress,
      sampleTime
    );
    let halfTime = HALF_LIFE_TIME;

    if (relaysHalfTime > halfTime) {
      halfTime = relaysHalfTime;
    }

    if (halfTime > MAX_HALF_TIME) {
      halfTime = MAX_HALF_TIME;
    }

    return halfTime;
  }

  private getRelayStatsTimeDiff(
    providerAddress: string,
    sampleTime: number
  ): number {
    const relayStatsTime = this.getRelayStatsTime(providerAddress);
    if (relayStatsTime.length === 0) {
      return 0;
    }
    const idx = Math.floor((relayStatsTime.length - 1) / 2);
    const diffTime = sampleTime - relayStatsTime[idx];

    if (diffTime < 0) {
      return now() - relayStatsTime[idx];
    }

    return diffTime;
  }

  private getRelayStatsTime(providerAddress: string): number[] {
    return this.providerRelayStats.get(providerAddress) || [];
  }
}

export function cumulativeProbabilityFunctionForPoissonDist(
  kEvents: number,
  lambda: number
): number {
  return 1 - gammainc(lambda, kEvents + 1);
}

export function perturbWithNormalGaussian(
  orig: number,
  percentage: number
): number {
  const normal = random.normal();
  const perturb = normal() * percentage * orig;
  return orig + perturb;
}

/**
 * This function is just to keep parity with the original golang implementation
 * @param value
 * @param precision
 */
export function floatToBigNumber(value: number, precision: number): BigNumber {
  const x = Math.pow(10, precision);
  const intVal = Math.round(value * x);
  return BigNumber(intVal / x);
}
