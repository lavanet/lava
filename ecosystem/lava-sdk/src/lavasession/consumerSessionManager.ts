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
import { Result } from "./helpers";
import {
  MAXIMUM_NUMBER_OF_FAILURES_ALLOWED_PER_CONSUMER_SESSION,
  RELAY_NUMBER_INCREMENT,
} from "./common";

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

  public constructor(
    rpcEndpoint: RPCEndpoint,
    providerOptimizer: ProviderOptimizer
  ) {
    this.rpcEndpoint = rpcEndpoint;
    this.providerOptimizer = providerOptimizer;
  }

  public getRpcEndpoint(): any {
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

  public updateAllProviders(
    epoch: number,
    pairingList: Map<number, ConsumerSessionsWithProvider>
  ): void {
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
  ): ConsumerSessionsMap {
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
    if (sessionWithProvidersMap.error) {
      throw sessionWithProvidersMap.error;
    }

    const wantedSessions = sessionWithProvidersMap.result.size;

    const sessions: ConsumerSessionsMap = {};
    while (true) {
      for (const sessionWithProviders of sessionWithProvidersMap.result) {
        const [providerAddress, sessionsWithProvider] = sessionWithProviders;
        const consumerSessionsWithProvider =
          sessionsWithProvider.sessionsWithProvider;
        let sessionEpoch = sessionsWithProvider.currentEpoch;

        const { connected, endpoint, error } =
          consumerSessionsWithProvider.fetchEndpointConnectionFromConsumerSessionWithProvider();
        if (error && error instanceof AllProviderEndpointsDisabledError) {
          this.blockProvider(providerAddress, true, sessionEpoch);
          continue;
        }

        if (!connected || endpoint === null) {
          tempIgnoredProviders.providers.add(providerAddress);
          continue;
        }

        const reportedProviders = this.getReportedProviders(sessionEpoch);
        const consumerSessionInstanceFromEndpoint =
          consumerSessionsWithProvider.getConsumerSessionInstanceFromEndpoint(
            endpoint,
            numberOfResets
          );
        if (consumerSessionInstanceFromEndpoint instanceof Error) {
          if (
            consumerSessionInstanceFromEndpoint instanceof
            MaximumNumberOfSessionsExceededError
          ) {
            tempIgnoredProviders.providers.add(providerAddress);
          } else if (
            consumerSessionInstanceFromEndpoint instanceof
            MaximumNumberOfBlockListedSessionsError
          ) {
            this.blockProvider(providerAddress, false, sessionEpoch);
          } else {
            throw consumerSessionInstanceFromEndpoint;
          }

          continue;
        }

        const { singleConsumerSession, pairingEpoch } =
          consumerSessionInstanceFromEndpoint;

        if (pairingEpoch !== sessionEpoch) {
          sessionEpoch = pairingEpoch;
        }

        const err =
          consumerSessionsWithProvider.addUsedComputeUnits(cuNeededForSession);
        if (err) {
          tempIgnoredProviders.providers.add(providerAddress);
          singleConsumerSession.unlock();
          continue;
        }

        singleConsumerSession.latestRelayCu = cuNeededForSession;
        singleConsumerSession.relayNum += RELAY_NUMBER_INCREMENT;

        sessions[providerAddress] = {
          session: singleConsumerSession,
          epoch: sessionEpoch,
          reportedProviders: reportedProviders,
        };

        tempIgnoredProviders.providers.add(providerAddress);

        if (Object.keys(sessions).length === wantedSessions) {
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

      if (sessionWithProvidersMap.error && Object.keys(sessions).length !== 0) {
        return sessions;
      }

      if (sessionWithProvidersMap.error) {
        throw sessionWithProvidersMap.error;
      }
    }
  }

  public onSessionUnused(consumerSession: any): void {
    throw new Error("Not implemented");
  }

  public onSessionFailure(
    consumerSession: SingleConsumerSession,
    // TODO: extract code from error
    errorReceived?: Error | null
  ): void {
    if (!consumerSession.isLocked()) {
      // TODO: improve error message
      throw new Error("Session is not locked");
    }

    if (consumerSession.blockListed) {
      throw new SessionIsAlreadyBlockListedError();
    }

    consumerSession.qoSInfo.totalRelays++;
    consumerSession.consecutiveNumberOfFailures++;

    let consumerSessionBlockListed = false;
    // TODO: verify if code == SessionOutOfSyncError.ABCICode() (from go)
    if (
      consumerSession.consecutiveNumberOfFailures >
      MAXIMUM_NUMBER_OF_FAILURES_ALLOWED_PER_CONSUMER_SESSION
    ) {
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
      // TODO: return error?
      throw error;
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
  ): void {
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
  ): void {
    if (sessionEpoch != this.currentEpoch) {
      throw new EpochMismatchError();
    }

    const result = this.removeAddressFromValidAddresses(address);
    if (
      result.error &&
      !(result.error instanceof AddressIndexWasNotFoundError)
    ) {
      throw result.error;
    }

    if (reportProvider) {
      this.addedToPurgeAndReport.add(address);
    }
  }

  private removeAddressFromValidAddresses(address: string): Result<void> {
    const idx = this.validAddresses.indexOf(address);
    if (idx === -1) {
      return {
        error: new AddressIndexWasNotFoundError(),
      };
    }

    this.validAddresses.splice(idx, 1);
    this.removeAddonAddress();
    return {
      result: undefined,
    };
  }

  private getValidConsumerSessionsWithProvider(
    ignoredProviders: IgnoredProviders,
    cuNeededForSession: number,
    requestedBlock: number,
    addon: string,
    extensions: string[]
  ): Result<SessionsWithProviderMap> {
    if (ignoredProviders.currentEpoch < this.currentEpoch) {
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

    if (providerAddresses.error) {
      return { error: providerAddresses.error };
    }

    const wantedProviders = providerAddresses.result.length;
    const sessionsWithProvider: SessionsWithProviderMap = new Map();

    while (true) {
      for (const providerAddress of providerAddresses.result) {
        const consumerSessionsWithProvider = this.pairing.get(providerAddress);
        if (consumerSessionsWithProvider === undefined) {
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
          return {
            result: sessionsWithProvider,
          };
        }
      }

      providerAddresses = this.getValidProviderAddress(
        Array.from(ignoredProviders.providers),
        cuNeededForSession,
        requestedBlock,
        addon,
        extensions
      );

      if (providerAddresses.error && sessionsWithProvider.size !== 0) {
        return {
          result: sessionsWithProvider,
        };
      }

      if (providerAddresses.error) {
        return {
          error: providerAddresses.error,
        };
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

  private getValidAddresses(addon: string, extensions: string[]): string[] {
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
  ): Result<string[]> {
    const ignoredProvidersLength = Object.keys(ignoredProviderList).length;
    const validAddresses = this.getValidAddresses(addon, extensions);
    const validAddressesLength = validAddresses.length;
    const totalValidLength = validAddressesLength - ignoredProvidersLength;

    if (totalValidLength <= 0) {
      return {
        error: new PairingListEmptyError(),
      };
    }

    const providers = this.providerOptimizer.chooseProvider(
      validAddresses,
      ignoredProviderList,
      cu,
      requestedBlock,
      0
    );

    if (providers.length === 0) {
      return {
        error: new PairingListEmptyError(),
      };
    }

    return {
      result: providers,
    };
  }

  private resetValidAddress(addon = "", extensions: string[] = []): number {
    const validAddresses = this.getValidAddresses(addon, extensions);
    if (validAddresses.length === 0) {
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
}
