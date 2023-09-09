import BigNumber from "bignumber.js";
import {
  AVAILABILITY_PERCENTAGE,
  DEFAULT_DECIMAL_PRECISION,
  MAX_ALLOWED_BLOCK_LISTED_SESSION_PER_PROVIDER,
  MAX_SESSIONS_ALLOWED_PER_PROVIDER,
  MIN_PROVIDERS_FOR_SYNC,
  PERCENTILE_TO_CALCULATE_LATENCY,
} from "./common";
import { generateRandomInt } from "../util/common";
import {
  AllProviderEndpointsDisabledError,
  AlreadyLockedError,
  MaxComputeUnitsExceededError,
  MaximumNumberOfBlockListedSessionsError,
  MaximumNumberOfSessionsExceededError,
  NegativeComputeUnitsAmountError,
  NotLockedError,
} from "./errors";
import { RelayerClient } from "../grpc_web_services/lavanet/lava/pairing/relay_pb_service";
import { Logger } from "../logger/logger";
import { Result } from "./helpers";
import { grpc } from "@improbable-eng/grpc-web";
import {
  QualityOfServiceReport,
  ReportedProvider,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb";

export interface SessionInfo {
  session: SingleConsumerSession;
  epoch: number;
  reportedProviders: Array<ReportedProvider>;
}

export type ConsumerSessionsMap = Map<string, SessionInfo>;

export interface ProviderOptimizer {
  appendProbeRelayData(
    providerAddress: string,
    latency: number,
    success: boolean
  ): void;

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
    ignoredProviders: string[],
    cu: number,
    requestedBlock: number,
    perturbationPercentage: number
  ): string[];

  getExcellenceQoSReportForProvider(
    providerAddress: string
  ): QualityOfServiceReport | undefined;
}

export interface QoSReport {
  lastQoSReport?: QualityOfServiceReport;
  lastExcellenceQoSReport?: QualityOfServiceReport;
  latencyScoreList: string[];
  syncScoreSum: number;
  totalSyncScore: number;
  totalRelays: number;
  answeredRelays: number;
}

export function calculateAvailabilityScore(qosReport: QoSReport): {
  downtimePercentage: string;
  scaledAvailabilityScore: string;
} {
  const downtimePercentage = BigNumber(
    (qosReport.totalRelays - qosReport.answeredRelays) / qosReport.totalRelays
  );

  const scaledAvailabilityScore = BigNumber(AVAILABILITY_PERCENTAGE)
    .minus(downtimePercentage)
    .div(AVAILABILITY_PERCENTAGE)
    .toPrecision();

  return {
    downtimePercentage: downtimePercentage.toPrecision(
      DEFAULT_DECIMAL_PRECISION
    ),
    scaledAvailabilityScore: BigNumber.max(
      BigNumber(0),
      scaledAvailabilityScore
    ).toPrecision(DEFAULT_DECIMAL_PRECISION),
  };
}

export interface IgnoredProviders {
  providers: Set<string>;
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
  public client: ConsumerSessionsWithProvider;
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
  private locked = false;

  public constructor(
    sessionId: number,
    client: ConsumerSessionsWithProvider,
    endpoint: Endpoint
  ) {
    this.sessionId = sessionId;
    this.client = client;
    this.endpoint = endpoint;
  }

  public tryLock(): AlreadyLockedError | undefined {
    if (!this.locked) {
      this.locked = true;
      return;
    }

    return new AlreadyLockedError();
  }

  public isLocked(): boolean {
    return this.locked;
  }

  public tryUnlock(): NotLockedError | undefined {
    if (!this.locked) {
      return new NotLockedError();
    }

    this.locked = false;
  }

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
      this.qoSInfo.lastQoSReport = new QualityOfServiceReport();
    }

    const { downtimePercentage, scaledAvailabilityScore } =
      calculateAvailabilityScore(this.qoSInfo);
    this.qoSInfo.lastQoSReport?.setAvailability(scaledAvailabilityScore);
    if (BigNumber(1).gt(this.qoSInfo.lastQoSReport.getAvailability())) {
      Logger.info(
        `QoS availability report ${JSON.stringify({
          availability: this.qoSInfo.lastQoSReport.getAvailability(),
          downPercent: downtimePercentage,
        })}`
      );
    }

    const latencyScore = this.calculateLatencyScore(expectedLatency, latency);
    this.qoSInfo.latencyScoreList.push(latencyScore);
    this.qoSInfo.latencyScoreList = this.qoSInfo.latencyScoreList.sort();
    this.qoSInfo.lastQoSReport.setLatency(
      this.qoSInfo.latencyScoreList[
        // golang int casting just cuts the decimal part
        Math.floor(
          this.qoSInfo.latencyScoreList.length * PERCENTILE_TO_CALCULATE_LATENCY
        )
      ]
    );

    const shouldCalculateSync =
      numOfProviders > Math.ceil(servicersToCount * MIN_PROVIDERS_FOR_SYNC);
    if (shouldCalculateSync) {
      if (blockHeightDiff <= 0) {
        this.qoSInfo.syncScoreSum++;
      }

      this.qoSInfo.totalSyncScore++;

      const sync = BigNumber(this.qoSInfo.syncScoreSum).div(
        this.qoSInfo.totalSyncScore
      );
      this.qoSInfo.lastQoSReport.setSync(
        sync.toPrecision(DEFAULT_DECIMAL_PRECISION)
      );

      if (BigNumber(1).gt(sync)) {
        Logger.debug(
          `QoS sync report ${JSON.stringify({
            sync: this.qoSInfo.lastQoSReport.getSync(),
            blockDiff: blockHeightDiff,
            syncScore: `${this.qoSInfo.syncScoreSum}/${this.qoSInfo.totalSyncScore}`,
            sessionId: this.sessionId,
          })}`
        );
      }
    } else {
      const sync = BigNumber(1);
      this.qoSInfo.lastQoSReport.setSync(
        sync.toPrecision(DEFAULT_DECIMAL_PRECISION)
      );
    }
    return;
  }

  private calculateLatencyScore(
    expectedLatency: number,
    latency: number
  ): string {
    const oneDec = BigNumber("1");
    const bigExpectedLatency = BigNumber(expectedLatency);
    const bigLatency = BigNumber(latency);

    return BigNumber.min(oneDec, bigExpectedLatency)
      .div(bigLatency)
      .toPrecision(DEFAULT_DECIMAL_PRECISION);
  }
}

export interface Endpoint {
  networkAddress: string;
  enabled: boolean;
  client?: RelayerClient;
  connectionRefusals: number;
  addons: Set<string>;
  extensions: Set<string>;
}

export class RPCEndpoint {
  public networkAddress = "";
  public chainId = "";
  public apiInterface = "";
  public geolocation = "1";

  public constructor(
    address: string,
    chainId: string,
    apiInterface: string,
    geolocation: string
  ) {
    this.networkAddress = address;
    this.chainId = chainId;
    this.apiInterface = apiInterface;
    this.geolocation = geolocation;
  }

  public key(): string {
    return this.chainId + this.apiInterface;
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
  private latestBlock = 0;
  private pairingEpoch: number;
  private conflictFoundAndReported = false; // 0 == not reported, 1 == reported

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

  public getLatestBlock(): number {
    return this.latestBlock;
  }

  public setLatestBlock(block: number) {
    this.latestBlock = block;
  }

  public getPublicLavaAddressAndPairingEpoch(): {
    publicProviderAddress: string;
    pairingEpoch: number;
  } {
    return {
      publicProviderAddress: this.publicLavaAddress,
      pairingEpoch: this.pairingEpoch,
    };
  }

  public conflictAlreadyReported(): boolean {
    return this.conflictFoundAndReported;
  }

  public storeConflictReported(): void {
    this.conflictFoundAndReported = true;
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

  public setPairingEpoch(epoch: number) {
    this.pairingEpoch = epoch;
  }

  public getConsumerSessionInstanceFromEndpoint(
    endpoint: Endpoint,
    numberOfResets: number
  ): Result<{
    singleConsumerSession: SingleConsumerSession;
    pairingEpoch: number;
  }> {
    const maximumBlockSessionsAllowed =
      MAX_ALLOWED_BLOCK_LISTED_SESSION_PER_PROVIDER * (numberOfResets + 1);

    let numberOfBlockedSessions = 0;
    for (const session of Object.values(this.sessions)) {
      if (session.endpoint != endpoint) {
        continue;
      }

      if (numberOfBlockedSessions >= maximumBlockSessionsAllowed) {
        return {
          pairingEpoch: 0,
          error: new MaximumNumberOfBlockListedSessionsError(),
        };
      }

      const lockError = session.tryLock();
      if (!lockError) {
        if (session.blockListed) {
          numberOfBlockedSessions++;
          const unlockError = session.tryUnlock();
          if (unlockError) {
            Logger.error("failed to unlock session", unlockError);
            return {
              error: unlockError,
            };
          }

          continue;
        }

        return {
          singleConsumerSession: session,
          pairingEpoch: this.pairingEpoch,
        };
      }
    }

    if (Object.keys(this.sessions).length > MAX_SESSIONS_ALLOWED_PER_PROVIDER) {
      throw new MaximumNumberOfSessionsExceededError();
    }

    // TODO: change Math.random to something else
    const randomSessionId = generateRandomInt();
    const session = new SingleConsumerSession(randomSessionId, this, endpoint);
    const lockError = session.tryLock();
    if (lockError) {
      Logger.error("failed to lock session", lockError);
      return {
        error: lockError,
      };
    }

    this.sessions[session.sessionId] = session;

    return {
      singleConsumerSession: session,
      pairingEpoch: this.pairingEpoch,
    };
  }

  public fetchEndpointConnectionFromConsumerSessionWithProvider(
    transport: grpc.TransportFactory
  ): Result<{
    connected: boolean;
    endpoint: Endpoint;
    providerAddress: string;
  }> {
    for (const endpoint of this.endpoints) {
      if (endpoint.enabled) {
        endpoint.client = new RelayerClient(
          "https://" + endpoint.networkAddress,
          {
            transport,
          }
        );

        this.endpoints.push(endpoint);

        return {
          connected: true,
          endpoint: endpoint,
          providerAddress: this.publicLavaAddress,
        };
      }
    }

    Logger.error(
      `purging provider after all endpoints are disabled ${JSON.stringify({
        providerEndpoints: this.endpoints,
        providerAddress: this.publicLavaAddress,
      })}`
    );

    return {
      connected: false,
      providerAddress: this.publicLavaAddress,
      error: new AllProviderEndpointsDisabledError(),
    };
  }

  public calculatedExpectedLatency(timeoutGivenToRelay: number): number {
    return timeoutGivenToRelay / 2;
  }

  public validateComputeUnits(
    cuNeededForSession: number
  ): MaxComputeUnitsExceededError | undefined {
    if (this.usedComputeUnits + cuNeededForSession > this.maxComputeUnits) {
      Logger.warn(
        `MaxComputeUnitsExceededError: ${this.publicLavaAddress} cu: ${this.usedComputeUnits} max: ${this.maxComputeUnits}`
      );
      return new MaxComputeUnitsExceededError();
    }
  }

  public addUsedComputeUnits(
    cu: number
  ): MaxComputeUnitsExceededError | undefined {
    if (this.usedComputeUnits + cu > this.maxComputeUnits) {
      return new MaxComputeUnitsExceededError();
    }

    this.usedComputeUnits += cu;
  }

  public decreaseUsedComputeUnits(
    cu: number
  ): NegativeComputeUnitsAmountError | undefined {
    if (this.usedComputeUnits - cu < 0) {
      return new NegativeComputeUnitsAmountError();
    }

    this.usedComputeUnits -= cu;
  }
}

export interface SessionsWithProvider {
  sessionsWithProvider: ConsumerSessionsWithProvider;
  currentEpoch: number;
}

export type SessionsWithProviderMap = Map<string, SessionsWithProvider>;
