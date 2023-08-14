import BigNumber from "bignumber.js";
import { AVAILABILITY_PERCENTAGE } from "./common";

export interface SessionInfo {
  session: SingleConsumerSession;
  epoch: number;
  reportedProviders: string[];
}

export type ConsumerSessionsMap = Record<string, SessionInfo>;

export interface ProviderOptimizer {
  appendRelayFailure(providerAddress: string): void;

  appendRelayData(
    providerAddress: string,
    latency: number,
    isHangingApi: boolean,
    cu: number,
    syncBlock: number
  ): void;

  chooseProvider(
    allAddresses: string[],
    ignoredProviders: Record<string, any>,
    cu: number,
    requestedBlock: number,
    perturbationPercentage: number
  ): string[];

  getExcellenceQoSReportForProvider(
    providerAddress: string
  ): QualityOfServiceReport;
}

export interface QualityOfServiceReport {
  latency: number;
  availability: number;
  sync: number;
}

export interface QoSReport {
  lastQoSReport?: QualityOfServiceReport;
  lastExcellenceQoSReport?: QualityOfServiceReport;
  latencyScoreList: number[];
  syncScoreSum: number;
  totalSyncScore: number;
  totalRelays: number;
  answeredRelays: number;
}

export function calculateAvailabilityScore(qosReport: QoSReport): {
  downtimePercentage: number;
  scaledAvailabilityScore: number;
} {
  const downtimePercentage = BigNumber(
    (qosReport.totalRelays - qosReport.answeredRelays) / qosReport.totalRelays
  )
    .precision(1)
    .toNumber();

  const scaledAvailabilityScore = BigNumber(
    (AVAILABILITY_PERCENTAGE - downtimePercentage) / AVAILABILITY_PERCENTAGE
  )
    .precision(1)
    .toNumber();

  return {
    downtimePercentage: downtimePercentage,
    scaledAvailabilityScore: Math.max(0, scaledAvailabilityScore),
  };
}

export interface IgnoredProviders {
  providers: Record<string, any>;
  currentEpoch: number;
}

export class SingleConsumerSession {
  public cuSum = 0;
  public latestRelayCu = 0;
  public qoSInfo: QoSReport = {
    latencyScoreList: [],
    totalRelays: 0,
    answeredRelays: 0,
    syncScoreSum: 0,
    totalSyncScore: 0,
  };
  public sessionId = 0;
  public client: any;
  public relayNum = 0;
  public latestBlock = 0;
  public endpoint: Endpoint = {
    networkAddress: "",
    enabled: false,
    connectionRefusals: 0,
    addons: new Set<string>(),
    extensions: new Set<string>(),
  };
  public blockListed = false;
  public consecutiveNumberOfFailures = 0;

  public calculateExpectedLatency(timeoutGivenToRelay: number): number {
    return timeoutGivenToRelay / 2;
  }

  public calculateQoS(
    latency: number,
    expectedLatency: number,
    blockHeightDiff: number,
    numOfProviders: number,
    servicersToCount: number
  ): void {
    this.qoSInfo.totalRelays++;
    this.qoSInfo.answeredRelays++;

    if (!this.qoSInfo.lastQoSReport) {
      this.qoSInfo.lastQoSReport = {
        latency: 0,
        availability: 0,
        sync: 0,
      };
    }

    const { downtimePercentage, scaledAvailabilityScore } =
      calculateAvailabilityScore(this.qoSInfo);
    if (BigNumber(1).gt(this.qoSInfo.lastQoSReport.availability)) {
      // todo: should we log? do we log in the sdk?
    }

    const latencyScore = 0;
    return;
  }

  private calculateLatencyScore(
    expectedLatency: number,
    latency: number
  ): number {
    const oneDec = BigNumber("1");
    const bigExpectedLatency = BigNumber(expectedLatency);
    const bigLatency = BigNumber(latency);

    return BigNumber.min(oneDec, bigLatency.div(bigExpectedLatency)).toNumber();
  }
}

export interface Endpoint {
  networkAddress: string;
  enabled: boolean;
  // TODO: add proper type here (RelayerClient)
  client?: any;
  connectionRefusals: number;
  addons: Set<string>;
  extensions: Set<string>;
}

export class RPCEndpoint {
  public networkAddress = "";
  public chainId = "";
  public apiInterface = "";
  public geolocation = 0;

  public constructor(
    address: string,
    chainId: string,
    apiInterface: string,
    geolocation: number
  ) {
    this.networkAddress = address;
    this.chainId = chainId;
    this.apiInterface = apiInterface;
    this.geolocation = geolocation;
  }

  public key(): string {
    return this.networkAddress + this.apiInterface;
  }

  public string(): string {
    return `${this.chainId}:${this.apiInterface} Network Address: ${this.networkAddress} Geolocation: ${this.geolocation}`;
  }
}

export class ConsumerSessionsWithProvider {
  public publicLavaAddress: string;
  public endpoints: Endpoint[];
  public sessions: Record<number, SingleConsumerSession>;
  public maxComputeUnits: number;
  public usedComputeUnits = 0;
  private pairingEpoch: number;
  private conflictFoundAndReported = 0; // 0 == not reported, 1 == reported

  public constructor(
    publicLavaAddress: string,
    endpoints: Endpoint[],
    sessions: Record<number, SingleConsumerSession>,
    maxComputeUnits: number,
    pairingEpoch: number
  ) {
    this.publicLavaAddress = publicLavaAddress;
    this.endpoints = endpoints;
    this.sessions = sessions;
    this.maxComputeUnits = maxComputeUnits;
    this.pairingEpoch = pairingEpoch;
  }

  public conflictAlreadyReported(): boolean {
    return false;
  }

  public storeConflictReported(): void {
    return;
  }

  public isSupportingAddon(addon: string): boolean {
    if (addon === "") {
      return true;
    }

    for (const endpoint of this.endpoints) {
      if (endpoint.addons.has(addon)) {
        return true;
      }
    }

    return false;
  }

  public isSupportingExtensions(extensions: string[]): boolean {
    let includesAll = true;

    for (const endpoint of this.endpoints) {
      for (const extension of extensions) {
        includesAll = includesAll && endpoint.extensions.has(extension);
      }
    }

    return includesAll;
  }

  public getPairingEpoch(): number {
    return this.pairingEpoch;
  }

  public getConsumerSessionInstanceFromEndpoint(
    endpoint: Endpoint,
    numberOfResets: number
  ): {
    singleConsumerSession: SingleConsumerSession;
    pairingEpoch: number;
  } {
    return {
      singleConsumerSession: {} as unknown as SingleConsumerSession,
      pairingEpoch: this.pairingEpoch,
    };
  }

  public calculatedExpectedLatency(timeoutGivenToRelay: number): number {
    return 0;
  }

  public calculateQoS(
    latency: number,
    expectedLatency: number,
    blockHeightDiff: number,
    numOfProviders: number,
    servicersToCount: number
  ): void {
    return;
  }
}

export interface SessionsWithProvider {
  sessionsWithProvider: ConsumerSessionsWithProvider;
  currentEpoch: number;
}

export type SessionsWithProviderMap = Record<string, SessionsWithProvider>;
