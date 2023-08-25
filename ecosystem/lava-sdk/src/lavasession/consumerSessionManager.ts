import {
  ConsumerSessionsMap,
  ConsumerSessionsWithProvider,
  IgnoredProviders,
  ProviderOptimizer,
  RPCEndpoint,
  SessionsWithProviderMap,
  SingleConsumerSession,
} from "./consumerTypes";
import { newRouterKey } from "./routerKey";
import {
  AddressIndexWasNotFoundError,
  AllProviderEndpointsDisabledError,
  BlockProviderError,
  EpochMismatchError,
  MaximumNumberOfBlockListedSessionsError,
  MaximumNumberOfSessionsExceededError,
  PairingListEmptyError,
  ReportAndBlockProviderError,
  SessionIsAlreadyBlockListedError,
} from "./errors";
import {
  MAXIMUM_NUMBER_OF_FAILURES_ALLOWED_PER_CONSUMER_SESSION,
  RELAY_NUMBER_INCREMENT,
} from "./common";
import { ProbeReply } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import BigNumber from "bignumber.js";
import Logger from "../logger/logger";
import Relayer from "../relayer/relayer";

export class ConsumerSessionManager {
  private rpcEndpoint: RPCEndpoint;
  private pairing: Map<string, ConsumerSessionsWithProvider> = new Map<
    string,
    ConsumerSessionsWithProvider
  >();
  private currentEpoch = 0;
  private numberOfResets = 0;

  private pairingAddresses: Map<number, string> = new Map<number, string>();

  public validAddresses: string[] = [];
  private addonAddresses: Map<string, string[]> = new Map<string, string[]>();
  private addedToPurgeAndReport: Set<string> = new Set();

  private pairingPurge: Map<string, ConsumerSessionsWithProvider> = new Map<
    string,
    ConsumerSessionsWithProvider
  >();
  private providerOptimizer: ProviderOptimizer;

  private relayer: Relayer;

  public constructor(
    relayer: Relayer,
    rpcEndpoint: RPCEndpoint,
    providerOptimizer: ProviderOptimizer
  ) {
    this.relayer = relayer;
    this.rpcEndpoint = rpcEndpoint;
    this.providerOptimizer = providerOptimizer;
  }

  public getRpcEndpoint(): RPCEndpoint {
    return this.rpcEndpoint;
  }

  public getCurrentEpoch(): number {
    return this.currentEpoch;
  }

  public getNumberOfResets(): number {
    return this.numberOfResets;
  }

  public getPairingAddressesLength(): number {
    return this.pairingAddresses.size;
  }

  public getAddedToPurgeAndReport(): Set<string> {
    return this.addedToPurgeAndReport;
  }

  public async updateAllProviders(
    epoch: number,
    pairingList: Map<number, ConsumerSessionsWithProvider>
  ): Promise<Error | undefined> {
    if (epoch <= this.currentEpoch) {
      Logger.error(
        `trying to update provider list for older epoch ${JSON.stringify({
          epoch,
          currentEpoch: this.currentEpoch,
        })}`
      );
      return new Error("Trying to update provider list for older epoch");
    }
    this.currentEpoch = epoch;

    // reset states
    this.pairingAddresses.clear();
    this.addedToPurgeAndReport.clear();
    this.numberOfResets = 0;
    this.removeAddonAddress();
    this.pairingPurge = this.pairing;
    this.pairing = new Map<string, ConsumerSessionsWithProvider>();

    pairingList.forEach(
      (provider: ConsumerSessionsWithProvider, idx: number) => {
        this.pairingAddresses.set(idx, provider.publicLavaAddress);
        this.pairing.set(provider.publicLavaAddress, provider);
      }
    );

    this.setValidAddressesToDefaultValue();

    Logger.debug(
      `updated providers ${JSON.stringify({
        epoch: this.currentEpoch,
        spec: this.rpcEndpoint.key(),
      })}`
    );

    await this.probeProviders(pairingList, epoch);
  }

  public removeAddonAddress(addon = "", extensions: string[] = []): void {
    if (addon === "" && extensions?.length === 0) {
      this.addonAddresses.clear();
      return;
    }

    const routerKey = newRouterKey([...extensions, addon]);
    this.addonAddresses.set(routerKey, []);
  }

  public calculateAddonValidAddresses(
    addon: string,
    extensions: string[]
  ): string[] {
    const supportingProviderAddresses: string[] = [];
    for (const address of this.validAddresses) {
      const provider = this.pairing.get(address);
      if (
        provider?.isSupportingAddon(addon) &&
        provider?.isSupportingExtensions(extensions)
      ) {
        supportingProviderAddresses.push(address);
      }
    }

    return supportingProviderAddresses;
  }

  public getSessions(
    cuNeededForSession: number,
    initUnwantedProviders: string[],
    requestedBlock: number,
    addon: string,
    extensions: string[]
  ): ConsumerSessionsMap | Error {
    const numberOfResets = this.validatePairingListNotEmpty(addon, extensions);
    const tempIgnoredProviders: IgnoredProviders = {
      providers: new Set(initUnwantedProviders),
      currentEpoch: this.currentEpoch,
    };

    let sessionWithProvidersMap = this.getValidConsumerSessionsWithProvider(
      tempIgnoredProviders,
      cuNeededForSession,
      requestedBlock,
      addon,
      extensions
    );
    if (sessionWithProvidersMap instanceof Error) {
      return sessionWithProvidersMap;
    }

    const wantedSessions = sessionWithProvidersMap.size;

    const sessions: ConsumerSessionsMap = new Map();
    while (true) {
      for (const sessionWithProviders of sessionWithProvidersMap) {
        const [providerAddress, sessionsWithProvider] = sessionWithProviders;
        const consumerSessionsWithProvider =
          sessionsWithProvider.sessionsWithProvider;
        let sessionEpoch = sessionsWithProvider.currentEpoch;

        const endpointConn =
          consumerSessionsWithProvider.fetchEndpointConnectionFromConsumerSessionWithProvider();

        if (endpointConn.error) {
          // if all provider endpoints are disabled, block and report provider
          if (endpointConn.error instanceof AllProviderEndpointsDisabledError) {
            this.blockProvider(providerAddress, true, sessionEpoch);
          } else {
            // if any other error just throw it
            throw endpointConn.error;
          }

          continue;
        }

        if (!endpointConn.connected) {
          tempIgnoredProviders.providers.add(providerAddress);
          continue;
        }

        const reportedProviders = this.getReportedProviders(sessionEpoch);
        const consumerSessionInstance =
          consumerSessionsWithProvider.getConsumerSessionInstanceFromEndpoint(
            endpointConn.endpoint,
            numberOfResets
          );
        if (consumerSessionInstance.error) {
          const { error } = consumerSessionInstance;

          if (error instanceof MaximumNumberOfSessionsExceededError) {
            tempIgnoredProviders.providers.add(providerAddress);
          } else if (error instanceof MaximumNumberOfBlockListedSessionsError) {
            this.blockProvider(providerAddress, false, sessionEpoch);
          } else {
            throw error;
          }

          continue;
        }

        const { singleConsumerSession, pairingEpoch } = consumerSessionInstance;

        if (pairingEpoch !== sessionEpoch) {
          Logger.error(
            `sessionEpoch and pairingEpoch mismatch sessionEpoch: ${sessionEpoch} pairingEpoch: ${pairingEpoch}`
          );
          sessionEpoch = pairingEpoch;
        }

        const err =
          consumerSessionsWithProvider.addUsedComputeUnits(cuNeededForSession);
        if (err) {
          Logger.debug(
            `consumerSessionWithProvider.addUsedComputeUnits error: ${err.message}`
          );

          tempIgnoredProviders.providers.add(providerAddress);
          singleConsumerSession.unlock();
          continue;
        }

        singleConsumerSession.latestRelayCu = cuNeededForSession;
        singleConsumerSession.relayNum += RELAY_NUMBER_INCREMENT;

        Logger.debug(
          `Consumer got session with provider: ${JSON.stringify({
            providerAddress,
            sessionEpoch,
            cuSum: singleConsumerSession.cuSum,
            relayNum: singleConsumerSession.relayNum,
            sessionId: singleConsumerSession.sessionId,
          })}`
        );

        sessions.set(providerAddress, {
          session: singleConsumerSession,
          epoch: sessionEpoch,
          reportedProviders: reportedProviders,
        });

        if (singleConsumerSession.relayNum > 1) {
          singleConsumerSession.qoSInfo.lastExcellenceQoSReport =
            this.providerOptimizer.getExcellenceQoSReportForProvider(
              providerAddress
            );
        }

        tempIgnoredProviders.providers.add(providerAddress);

        if (sessions.size === wantedSessions) {
          return sessions;
        }
      }

      sessionWithProvidersMap = this.getValidConsumerSessionsWithProvider(
        tempIgnoredProviders,
        cuNeededForSession,
        requestedBlock,
        addon,
        extensions
      );

      if (sessionWithProvidersMap instanceof Error && sessions.size !== 0) {
        return sessions;
      }

      if (sessionWithProvidersMap instanceof Error) {
        return sessionWithProvidersMap;
      }
    }
  }

  public onSessionUnused(
    consumerSession: SingleConsumerSession
  ): Error | undefined {
    if (consumerSession.tryLock()) {
      return new Error(
        "consumer session must be locked before accessing this method"
      );
    }

    const cuToDecrease = consumerSession.latestRelayCu;
    consumerSession.latestRelayCu = 0;
    const parentConsumerSessionsWithProvider = consumerSession.client;
    consumerSession.unlock();

    return parentConsumerSessionsWithProvider.decreaseUsedComputeUnits(
      cuToDecrease
    );
  }

  public onSessionFailure(
    consumerSession: SingleConsumerSession,
    // TODO: extract code from error
    errorReceived?: Error | null
  ): Error | undefined {
    if (!consumerSession.isLocked()) {
      return new Error("Session is not locked");
    }

    if (consumerSession.blockListed) {
      return new SessionIsAlreadyBlockListedError();
    }

    consumerSession.qoSInfo.totalRelays++;
    consumerSession.consecutiveNumberOfFailures++;

    let consumerSessionBlockListed = false;
    // TODO: verify if code == SessionOutOfSyncError.ABCICode() (from go)
    if (
      consumerSession.consecutiveNumberOfFailures >
      MAXIMUM_NUMBER_OF_FAILURES_ALLOWED_PER_CONSUMER_SESSION
    ) {
      Logger.debug(
        `Blocking consumer session id: ${consumerSession.sessionId}`
      );
      consumerSession.blockListed = true;
      consumerSessionBlockListed = true;
    }
    const cuToDecrease = consumerSession.latestRelayCu;
    this.providerOptimizer.appendRelayFailure(
      consumerSession.client.publicLavaAddress
    );
    consumerSession.latestRelayCu = 0;

    const parentConsumerSessionsWithProvider = consumerSession.client;
    consumerSession.unlock();

    const error =
      parentConsumerSessionsWithProvider.decreaseUsedComputeUnits(cuToDecrease);
    if (error) {
      return error;
    }

    let blockProvider = false;
    let reportProvider = false;
    if (errorReceived instanceof ReportAndBlockProviderError) {
      blockProvider = true;
      reportProvider = true;
    } else if (errorReceived instanceof BlockProviderError) {
      blockProvider = true;
    }

    if (
      consumerSessionBlockListed &&
      parentConsumerSessionsWithProvider.usedComputeUnits === 0
    ) {
      blockProvider = true;
      reportProvider = true;
    }

    if (blockProvider) {
      const { publicProviderAddress, pairingEpoch } =
        parentConsumerSessionsWithProvider.getPublicLavaAddressAndPairingEpoch();
      this.blockProvider(publicProviderAddress, reportProvider, pairingEpoch);
    }
  }

  public onSessionDone(
    consumerSession: SingleConsumerSession,
    latestServicedBlock: number,
    specComputeUnits: number,
    currentLatency: number,
    expectedLatency: number,
    expectedBH: number,
    numOfProviders: number,
    providersCount: number,
    isHangingApi: boolean
  ): Error | undefined {
    if (!consumerSession.isLocked()) {
      return new Error("Session is not locked");
    }

    consumerSession.cuSum += consumerSession.latestRelayCu;
    consumerSession.latestRelayCu = 0;
    consumerSession.consecutiveNumberOfFailures = 0;
    consumerSession.latestBlock = latestServicedBlock;
    consumerSession.calculateQoS(
      currentLatency,
      expectedLatency,
      expectedBH - latestServicedBlock,
      numOfProviders,
      providersCount
    );
    this.providerOptimizer.appendRelayData(
      consumerSession.client.publicLavaAddress,
      currentLatency,
      isHangingApi,
      specComputeUnits,
      latestServicedBlock
    );
    consumerSession.unlock();
  }

  public getReportedProviders(epoch: number): string {
    if (epoch != this.currentEpoch) {
      return "";
    }
    return JSON.stringify(Array.from(this.addedToPurgeAndReport));
  }

  private blockProvider(
    address: string,
    reportProvider: boolean,
    sessionEpoch: number
  ): Error | undefined {
    if (sessionEpoch != this.currentEpoch) {
      return new EpochMismatchError();
    }

    const error = this.removeAddressFromValidAddresses(address);
    if (error) {
      Logger.error(`address ${address} was not found in valid addresses`);
    }

    if (reportProvider) {
      Logger.info(`Reporting provider for unresponsiveness: ${address}`);
      this.addedToPurgeAndReport.add(address);
    }
  }

  private removeAddressFromValidAddresses(address: string): Error | undefined {
    const idx = this.validAddresses.indexOf(address);
    if (idx === -1) {
      return new AddressIndexWasNotFoundError();
    }

    this.validAddresses.splice(idx, 1);
    this.removeAddonAddress();
  }

  private getValidConsumerSessionsWithProvider(
    ignoredProviders: IgnoredProviders,
    cuNeededForSession: number,
    requestedBlock: number,
    addon: string,
    extensions: string[]
  ): SessionsWithProviderMap | Error {
    Logger.debug(
      `called getValidConsumerSessionsWithProvider ${JSON.stringify({
        ignoredProviders,
      })}`
    );

    if (ignoredProviders.currentEpoch < this.currentEpoch) {
      Logger.debug(
        `ignoredP epoch is not current epoch, resetting ignoredProviders ${JSON.stringify(
          {
            ignoredProvidersEpoch: ignoredProviders.currentEpoch,
            currentEpoch: this.currentEpoch,
          }
        )}`
      );

      ignoredProviders.providers = new Set();
      ignoredProviders.currentEpoch = this.currentEpoch;
    }

    let providerAddresses = this.getValidProviderAddress(
      Array.from(ignoredProviders.providers),
      cuNeededForSession,
      requestedBlock,
      addon,
      extensions
    );

    if (providerAddresses instanceof Error) {
      Logger.error(
        `could not get a provider addresses error: ${providerAddresses.message}`
      );
      return providerAddresses;
    }

    const wantedProviders = providerAddresses.length;
    const sessionsWithProvider: SessionsWithProviderMap = new Map();

    while (true) {
      for (const providerAddress of providerAddresses) {
        const consumerSessionsWithProvider = this.pairing.get(providerAddress);
        if (consumerSessionsWithProvider === undefined) {
          Logger.error(
            `invalid provider address returned from csm.getValidProviderAddresses ${JSON.stringify(
              {
                providerAddress,
                allProviderAddresses: providerAddresses,
                pairing: this.pairing,
                currentEpoch: this.currentEpoch,
                validAddresses: this.getValidAddresses(addon, extensions),
                wantedProviderNumber: wantedProviders,
              }
            )}`
          );

          throw new Error(
            "Invalid provider address returned from csm.getValidProviderAddresses"
          );
        }

        const err =
          consumerSessionsWithProvider.validateComputeUnits(cuNeededForSession);
        if (err) {
          ignoredProviders.providers.add(providerAddress);
          continue;
        }

        sessionsWithProvider.set(providerAddress, {
          sessionsWithProvider: consumerSessionsWithProvider,
          currentEpoch: this.currentEpoch,
        });

        ignoredProviders.providers.add(providerAddress);

        if (sessionsWithProvider.size === wantedProviders) {
          return sessionsWithProvider;
        }
      }

      providerAddresses = this.getValidProviderAddress(
        Array.from(ignoredProviders.providers),
        cuNeededForSession,
        requestedBlock,
        addon,
        extensions
      );

      if (
        providerAddresses instanceof Error &&
        sessionsWithProvider.size !== 0
      ) {
        return sessionsWithProvider;
      }

      if (providerAddresses instanceof Error) {
        Logger.debug(
          `could not get a provider address ${providerAddresses.message}`
        );
        return providerAddresses;
      }
    }
  }

  private setValidAddressesToDefaultValue(
    addon = "",
    extensions: string[] = []
  ): void {
    if (addon === "" && extensions.length === 0) {
      this.validAddresses = [];
      this.pairingAddresses.forEach((address: string) => {
        this.validAddresses.push(address);
      });

      return;
    }

    this.pairingAddresses.forEach((address: string) => {
      if (this.validAddresses.includes(address)) {
        return;
      }

      this.validAddresses.push(address);
    });

    this.removeAddonAddress(addon, extensions);
    const routerKey = newRouterKey([...extensions, addon]);
    const addonAddresses = this.calculateAddonValidAddresses(addon, extensions);
    this.addonAddresses.set(routerKey, addonAddresses);
  }

  public getValidAddresses(addon: string, extensions: string[]): string[] {
    const routerKey = newRouterKey([...extensions, addon]);

    const validAddresses = this.addonAddresses.get(routerKey);
    if (validAddresses === undefined || validAddresses.length === 0) {
      return this.calculateAddonValidAddresses(addon, extensions);
    }

    return validAddresses;
  }

  private getValidProviderAddress(
    ignoredProviderList: string[],
    cu: number,
    requestedBlock: number,
    addon: string,
    extensions: string[]
  ): string[] | Error {
    const ignoredProvidersLength = Object.keys(ignoredProviderList).length;
    const validAddresses = this.getValidAddresses(addon, extensions);
    const validAddressesLength = validAddresses.length;
    const totalValidLength = validAddressesLength - ignoredProvidersLength;

    if (totalValidLength <= 0) {
      Logger.debug(
        `pairing list empty ${JSON.stringify({
          providerList: validAddresses,
          ignoredProviderList,
        })}`
      );
      return new PairingListEmptyError();
    }

    const providers = this.providerOptimizer.chooseProvider(
      validAddresses,
      ignoredProviderList,
      cu,
      requestedBlock,
      0
    );

    Logger.debug(
      `choosing provider ${JSON.stringify({
        validAddresses,
        ignoredProviderList,
        providers,
      })}`
    );

    if (providers.length === 0) {
      Logger.debug(
        `No providers returned by the optimizer ${JSON.stringify({
          providerList: validAddresses,
          ignoredProviderList,
        })}`
      );
      return new PairingListEmptyError();
    }

    return providers;
  }

  private resetValidAddress(addon = "", extensions: string[] = []): number {
    const validAddresses = this.getValidAddresses(addon, extensions);
    if (validAddresses.length === 0) {
      Logger.warn("provider pairing list is empty, resetting state");
      this.setValidAddressesToDefaultValue(addon, extensions);
      this.numberOfResets++;
    }

    return this.numberOfResets;
  }

  private cacheAddonAddresses(addon: string, extensions: string[]): string[] {
    const routerKey = newRouterKey([...extensions, addon]);

    let addonAddresses = this.addonAddresses.get(routerKey);
    if (!addonAddresses) {
      this.removeAddonAddress(addon, extensions);
      addonAddresses = this.calculateAddonValidAddresses(addon, extensions);
      this.addonAddresses.set(routerKey, addonAddresses);
    }

    return addonAddresses;
  }

  private validatePairingListNotEmpty(
    addon: string,
    extensions: string[]
  ): number {
    const validAddresses = this.cacheAddonAddresses(addon, extensions);
    if (validAddresses.length === 0) {
      return this.resetValidAddress(addon, extensions);
    }

    return this.numberOfResets;
  }

  private async probeProviders(
    pairingList: Map<number, ConsumerSessionsWithProvider>,
    epoch: number
  ) {
    // TODO: better random generator
    const random = BigNumber.random(10);
    const guid = random.shiftedBy(random.sd()).toNumber();

    Logger.info(
      `providers probe initiated ${JSON.stringify({
        endpoint: this.rpcEndpoint,
        guid,
        epoch,
      })}`
    );

    const probePromises: Promise<{
      latency: number;
      providerAddress: string;
      error?: Error;
    }>[] = [];
    for (const consumerSessionWithProvider of pairingList.values()) {
      probePromises.push(this.probeProvider(guid, consumerSessionWithProvider));
    }

    const probeResults = await Promise.all(probePromises);
    for (const probeResult of probeResults) {
      const { latency, providerAddress, error } = probeResult;
      this.providerOptimizer.appendProbeRelayData(
        providerAddress,
        latency,
        Boolean(error)
      );
    }

    Logger.debug(
      `providers probe done ${JSON.stringify({
        endpoint: this.rpcEndpoint,
        guid,
        epoch,
      })}`
    );
  }

  private async probeProvider(
    guid: number,
    consumerSessionsWithProvider: ConsumerSessionsWithProvider
  ): Promise<{ latency: number; providerAddress: string; error?: Error }> {
    const endpointConn =
      consumerSessionsWithProvider.fetchEndpointConnectionFromConsumerSessionWithProvider();
    if (endpointConn.error || !endpointConn.connected) {
      return {
        latency: 0,
        // we always have the provider address here
        providerAddress: endpointConn.providerAddress as string,
        error: endpointConn.error,
      };
    }

    const { endpoint, providerAddress } = endpointConn;
    const client = endpoint?.client;
    const relaySentTime = Date.now();
    if (!client) {
      Logger.error(
        `returned null client in endpoint ${JSON.stringify({
          consumerSessionsWithProvider,
        })}`
      );

      return {
        latency: 0,
        providerAddress: providerAddress,
        error: new Error("endpoint client is null"),
      };
    }

    try {
      const response = await this.relayer.probeProvider(
        endpoint.networkAddress,
        this.rpcEndpoint.apiInterface,
        this.rpcEndpoint.chainId
      );
      const relayLatency = Date.now() - relaySentTime; // in milliseconds

      const providerGuid = response.getGuid();
      if (providerGuid !== guid) {
        Logger.error(
          `mismatch probe resposne ${JSON.stringify({
            providerAddress,
            providerGuid,
            sentGuid: guid,
          })}`
        );

        return {
          latency: 0,
          error: new Error("Mismatch probe response"),
          providerAddress,
        };
      }

      const latestBlock = response.getLatestBlock();
      if (latestBlock === 0) {
        Logger.error(
          `provider returned 0 latest block ${JSON.stringify({
            providerAddress,
          })}`
        );

        return {
          latency: 0,
          providerAddress: providerAddress,
          error: new Error("Provider returned 0 latest block"),
        };
      }

      consumerSessionsWithProvider.setLatestBlock(latestBlock);

      Logger.debug(
        `probed provider successfully ${JSON.stringify({
          latency: relayLatency,
          providerAddress,
        })}`
      );

      return {
        latency: relayLatency,
        providerAddress,
      };
    } catch (e) {
      Logger.error(
        `probe call error ${(e as Error).message} ${JSON.stringify({
          providerAddress,
        })}`
      );
      return {
        latency: 0,
        error: new Error("Probe call error"),
        providerAddress,
      };
    }
  }

  private async relayWithTimeout(
    timeLimit: number,
    task: Promise<ProbeReply>
  ): Promise<ProbeReply | Error> {
    return Promise.race([task, this.timeoutPromise(timeLimit)]);
  }

  private timeoutPromise(timeout: number): Promise<Error> {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        reject(new Error("Timeout exceeded"));
      }, timeout);
    });
  }
}
