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
  MAX_CONSECUTIVE_CONNECTION_ATTEMPTS,
  MAXIMUM_NUMBER_OF_FAILURES_ALLOWED_PER_CONSUMER_SESSION,
  RELAY_NUMBER_INCREMENT,
} from "./common";
import { Logger } from "../logger/logger";
import { Relayer } from "../relayer/relayer";
import { grpc } from "@improbable-eng/grpc-web";
import transportAllowInsecure from "../util/browserAllowInsecure";
import transport from "../util/browser";
import { secondsToMillis } from "../util/time";
import { sleep } from "../util/common";
import { ProviderEpochTracker } from "./providerEpochTracker";
import { APIInterfaceTendermintRPC } from "../chainlib/base_chain_parser";
export const ALLOWED_PROBE_RETRIES = 3;
export const TIMEOUT_BETWEEN_PROBES = secondsToMillis(1);
import { ReportedProviders } from "./reported_providers";
import { ReportedProvider } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";

export class ConsumerSessionManager {
  private rpcEndpoint: RPCEndpoint;
  private pairing: Map<string, ConsumerSessionsWithProvider> = new Map<
    string,
    ConsumerSessionsWithProvider
  >();
  private currentEpoch = 0;
  private numberOfResets = 0;
  private allowedUpdateForCurrentEpoch = true;

  private pairingAddresses: Map<number, string> = new Map<number, string>();

  public validAddresses: string[] = [];
  private addonAddresses: Map<string, string[]> = new Map<string, string[]>();
  private reportedProviders: ReportedProviders = new ReportedProviders();

  private pairingPurge: Map<string, ConsumerSessionsWithProvider> = new Map<
    string,
    ConsumerSessionsWithProvider
  >();
  private providerOptimizer: ProviderOptimizer;

  private relayer: Relayer;

  private transport: grpc.TransportFactory;
  private allowInsecureTransport = false;
  private epochTracker = new ProviderEpochTracker();

  public constructor(
    relayer: Relayer,
    rpcEndpoint: RPCEndpoint,
    providerOptimizer: ProviderOptimizer,
    opts?: {
      transport?: grpc.TransportFactory;
      allowInsecureTransport?: boolean;
    }
  ) {
    this.relayer = relayer;
    this.rpcEndpoint = rpcEndpoint;
    this.providerOptimizer = providerOptimizer;

    this.allowInsecureTransport = opts?.allowInsecureTransport ?? false;
    this.transport = opts?.transport ?? this.getTransport();
  }

  public getEpochFromEpochTracker(): number {
    return this.epochTracker.getEpoch();
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

  public async updateAllProviders(
    epoch: number,
    pairingList: ConsumerSessionsWithProvider[]
  ): Promise<Error | undefined> {
    Logger.info(
      "updateAllProviders called. epoch:",
      epoch,
      "this.currentEpoch",
      this.currentEpoch,
      "Provider list length",
      pairingList.length,
      "Api Inteface",
      this.rpcEndpoint.apiInterface,
      this.rpcEndpoint.chainId
    );

    if (epoch <= this.currentEpoch) {
      const rpcEndpoint = this.getRpcEndpoint();

      // For LAVA's initialization, we need to allow the pairing to be updated twice
      // This condition permits the pairing to be overwritten just once for the same epoch
      // After this one-time allowance, any attempt to overwrite will result in an error
      if (epoch != 0) {
        if (
          this.allowedUpdateForCurrentEpoch &&
          epoch === this.currentEpoch &&
          rpcEndpoint.chainId === "LAV1" &&
          rpcEndpoint.apiInterface === APIInterfaceTendermintRPC
        ) {
          this.allowedUpdateForCurrentEpoch = false;
        } else {
          Logger.error(
            `trying to update provider list for older epoch ${JSON.stringify({
              epoch,
              currentEpoch: this.currentEpoch,
            })}`
          );
          return new Error("Trying to update provider list for older epoch");
        }
      }
    }
    this.epochTracker.reset();
    this.currentEpoch = epoch;

    // reset states
    this.pairingAddresses.clear();
    this.reportedProviders.reset();
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
    try {
      await this.probeProviders(new Set(pairingList), epoch);
    } catch (err) {
      // TODO see what we should to
      Logger.error(err);
    }
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
    initUnwantedProviders: Set<string>,
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
          consumerSessionsWithProvider.fetchEndpointConnectionFromConsumerSessionWithProvider(
            this.transport
          );

        if (endpointConn.error) {
          // if all provider endpoints are disabled, block and report provider
          if (endpointConn.error instanceof AllProviderEndpointsDisabledError) {
            this.blockProvider(providerAddress, true, sessionEpoch, 0, 1); // endpoints are disabled
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
            this.blockProvider(providerAddress, false, sessionEpoch, 0, 0);
          } else {
            throw error;
          }

          continue;
        }

        const { singleConsumerSession, pairingEpoch } = consumerSessionInstance;

        if (pairingEpoch !== sessionEpoch && pairingEpoch != 0) {
          // if pairingEpoch == 0 its currently uninitialized so we keep the this.currentEpoch value
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
          const unlockError = singleConsumerSession.tryUnlock();
          if (unlockError) {
            Logger.error("unlock error", unlockError);
            return unlockError;
          }

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
          reportedProviders: this.reportedProviders.GetReportedProviders(),
        });

        if (singleConsumerSession.relayNum > 1) {
          singleConsumerSession.qoSInfo.lastExcellenceQoSReport =
            this.providerOptimizer.getExcellenceQoSReportForProvider(
              providerAddress
            );
        }

        tempIgnoredProviders.providers.add(providerAddress);

        if (sessions.size === wantedSessions) {
          Logger.debug(
            `returning sessions: ${JSON.stringify(sessions.values())}`,
            sessions.size
          );
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
    const lockError = consumerSession.tryLock();
    if (!lockError) {
      return new Error(
        "consumer session must be locked before accessing this method"
      );
    }

    const cuToDecrease = consumerSession.latestRelayCu;
    consumerSession.latestRelayCu = 0;
    const parentConsumerSessionsWithProvider = consumerSession.client;
    const unlockError = consumerSession.tryUnlock();
    if (unlockError) {
      Logger.error("unlock error", unlockError);
      return unlockError;
    }

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
    const unlockError = consumerSession.tryUnlock();
    if (unlockError) {
      Logger.error("unlock error", unlockError);
      return unlockError;
    }

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
      this.blockProvider(
        publicProviderAddress,
        reportProvider,
        pairingEpoch,
        1,
        0
      );
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

    const unlockError = consumerSession.tryUnlock();
    if (unlockError) {
      Logger.error("unlock error", unlockError);
      return unlockError;
    }
  }

  public getReportedProviders(epoch: number): Array<ReportedProvider> {
    if (epoch != this.currentEpoch) {
      return new Array<ReportedProvider>();
    }
    // If the addedToPurgeAndReport is empty return empty string
    // because "[]" can not be parsed
    return this.reportedProviders.GetReportedProviders();
  }

  private blockProvider(
    address: string,
    reportProvider: boolean,
    sessionEpoch: number,
    errors: number,
    disconnections: number
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
      this.reportedProviders.reportedProvider(address, errors, disconnections);
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

  public async probeProviders(
    pairingList: Set<ConsumerSessionsWithProvider>,
    epoch: number,
    retry = 0
  ): Promise<any> {
    if (retry != 0) {
      await sleep(this.timeoutBetweenProbes());
      if (this.currentEpoch != epoch) {
        // incase epoch has passed we no longer need to probe old providers
        Logger.info(
          "during old probe providers epoch passed no need to query old providers."
        );
        return;
      }
    }

    Logger.debug(`providers probe initiated`);
    let promiseProbeArray: Array<Promise<any>> = [];
    const retryProbing: Set<ConsumerSessionsWithProvider> = new Set();
    for (const consumerSessionWithProvider of pairingList) {
      promiseProbeArray = promiseProbeArray.concat(
        this.probeProvider(consumerSessionWithProvider, epoch, retryProbing)
      );
    }
    if (!retry) {
      for (const s of pairingList) {
        await Promise.race(promiseProbeArray);
        const epochFromProviders = this.epochTracker.getEpoch();
        if (epochFromProviders != -1) {
          Logger.debug(
            `providers probe done ${JSON.stringify({
              endpoint: this.rpcEndpoint,
              Epoch: epochFromProviders,
              NumberOfProvidersProbedUntilFinishedInit:
                this.epochTracker.getProviderListSize(),
            })}`
          );
          break;
        }
      }
    } else {
      await Promise.allSettled(promiseProbeArray);
    }
    // stop if we have no more providers to probe or we hit limit
    if (retryProbing.size == 0 || retry >= ALLOWED_PROBE_RETRIES) {
      return;
    }

    // launch retry probing on failed providers; this needs to run asynchronously without waiting!
    // Must NOT "await" this method.
    this.probeProviders(retryProbing, epoch, retry + 1);
  }

  private probeProvider(
    consumerSessionWithProvider: ConsumerSessionsWithProvider,
    epoch: number,
    retryProbing: Set<ConsumerSessionsWithProvider>
  ) {
    const startTime = performance.now();
    const guid = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
    const promises = [];
    for (const endpoint of consumerSessionWithProvider.endpoints) {
      if (!endpoint.enabled) {
        continue;
      }

      if (endpoint.connectionRefusals >= MAX_CONSECUTIVE_CONNECTION_ATTEMPTS) {
        endpoint.enabled = false;
        Logger.warn(
          "disabling provider endpoint for the duration of current epoch",
          endpoint.networkAddress
        );
        continue;
      }

      promises.push(
        this.relayer
          .probeProvider(
            endpoint.networkAddress,
            this.getRpcEndpoint().apiInterface,
            guid,
            this.getRpcEndpoint().chainId
          )
          .then((probeReply) => {
            const endTime = performance.now();
            const latency = endTime - startTime;
            Logger.debug(
              "Provider: " +
                consumerSessionWithProvider.publicLavaAddress +
                " chainID: " +
                this.getRpcEndpoint().chainId +
                " latency: ",
              latency + " ms"
            );

            if (guid != probeReply.getGuid()) {
              Logger.error(
                "Guid mismatch for probe request and response. requested: ",
                guid,
                "response:",
                probeReply.getGuid()
              );
            }

            const lavaEpoch = probeReply.getLavaEpoch();
            Logger.debug(
              `Probing Result for provider ${
                consumerSessionWithProvider.publicLavaAddress
              }, Epoch: ${lavaEpoch}, Lava Block: ${probeReply.getLavaLatestBlock()}`
            );
            this.epochTracker.setEpoch(
              consumerSessionWithProvider.publicLavaAddress,
              lavaEpoch
            );
            // when epoch == 0 this is the initialization of the sdk. meaning we don't have information, we will take the median
            // reported epoch from the providers probing and change the current epoch value as we probe more providers.
            if (epoch == 0) {
              this.currentEpoch = this.getEpochFromEpochTracker(); // setting the epoch for initialization.
            }
            consumerSessionWithProvider.setPairingEpoch(this.currentEpoch); // set the pairing epoch on the specific provider.
            this.providerOptimizer.appendProbeRelayData(
              consumerSessionWithProvider.publicLavaAddress,
              latency,
              true
            );

            endpoint.connectionRefusals = 0;
            retryProbing.delete(consumerSessionWithProvider);
          })
          .catch((e) => {
            Logger.warn(
              "Failed fetching probe from provider",
              consumerSessionWithProvider.getPublicLavaAddressAndPairingEpoch(),
              "Error:",
              e
            );

            this.providerOptimizer.appendProbeRelayData(
              consumerSessionWithProvider.publicLavaAddress,
              0,
              false
            );

            endpoint.connectionRefusals++;
            retryProbing.add(consumerSessionWithProvider);
          })
      );
    }

    // if no endpoints are enabled, return an error
    if (promises.length === 0) {
      // resolve instead of reject to avoid unhandled promise rejection
      // we do not want to crash the process here, just to resolve probing
      return Promise.resolve(new AllProviderEndpointsDisabledError());
    }

    return promises;
  }

  private getTransport(): grpc.TransportFactory {
    return this.allowInsecureTransport ? transportAllowInsecure : transport;
  }

  private timeoutBetweenProbes(): number {
    return TIMEOUT_BETWEEN_PROBES;
  }
}

export type ConsumerSessionManagersMap = Map<string, ConsumerSessionManager[]>;
