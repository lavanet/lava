import {
  ConsumerSessionsMap,
  ConsumerSessionsWithProvider,
  ProviderOptimizer,
  RPCEndpoint,
} from "./consumerTypes";
import { newRouterKey } from "./routerKey";
import { PairingListEmptyError } from "./errors";

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
  private addedToPurgeAndReport: Map<string, any> = new Map<string, any>();

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
        provider?.isSupportingAddon(addon) ||
        provider?.isSupportingExtensions(extensions)
      ) {
        supportingProviderAddresses.push(address);
      }
    }

    return supportingProviderAddresses;
  }

  public getSessions(
    cuNeededForSession: number,
    initUnwantedProviders: Record<string, any>,
    requestedBlock: number,
    addon: string,
    extensions: string[]
  ): ConsumerSessionsMap {
    return {};
  }

  public onSessionUnused(consumerSession: any): void {
    throw new Error("Not implemented");
  }

  public onSessionFailure(consumerSession: any, errorReceived: Error): void {
    throw new Error("Not implemented");
  }

  public onSessionDone(
    consumerSession: any,
    latestServicedBlock: number,
    specComputeUnits: number,
    currentLatency: number,
    expectedLatency: number,
    expectedBH: number,
    numOfProviders: number,
    providersCount: number,
    isHangingApi: boolean
  ): void {
    throw new Error("Not implemented");
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
    if (validAddresses === undefined) {
      return this.calculateAddonValidAddresses(addon, extensions);
    }

    return validAddresses;
  }

  private getValidProviderAddress(
    ignoredProviderList: Record<string, any>,
    cu: number,
    requestedBlock: number,
    addon: string,
    extensions: string[]
  ): string[] {
    const ignoredProvidersLength = Object.keys(ignoredProviderList).length;
    const validAddresses = this.getValidAddresses(addon, extensions);
    const validAddressesLength = validAddresses.length;
    const totalValidLength = validAddressesLength - ignoredProvidersLength;

    if (totalValidLength <= 0) {
      throw new PairingListEmptyError();
    }

    const providers = this.providerOptimizer.chooseProvider(
      validAddresses,
      ignoredProviderList,
      cu,
      requestedBlock,
      0
    );

    if (providers.length === 0) {
      throw new PairingListEmptyError();
    }

    return providers;
  }
}
