import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";
import {
  ConsumerSessionsWithProvider,
  Endpoint,
} from "../lavasession/consumerTypes";
import { BaseChainParser } from "../chainlib/base_chain_parser";
import { Spec } from "../codec/lavanet/lava/spec/spec";
import { PairingResponse } from "../stateTracker/stateQuery/state_query";
import { Logger } from "../logger/logger";
import { parseLong } from "../util/common";
import { RPCEndpoint } from "../lavasession/consumerTypes";
import { RandomProviderOptimizer } from "../lavasession/providerOptimizer";
import { Relayer } from "../relayer/relayer";
import {
  SendRelayOptions,
  SendRestRelayOptions,
  ChainParser,
} from "../chainlib/chainlib_interface";
import { getChainParser } from "../chainlib/utils";

type ApiInterface = string;
export class RPCConsumer {
  private chainParser: Map<ApiInterface, ChainParser>;
  private geolocation: string;
  private relayer: Relayer;
  constructor(geolocation: string, relayer: Relayer) {
    this.chainParser = new Map();
    this.geolocation = geolocation;
    this.relayer = relayer;
  }

  static async create(
    pairingResponse: PairingResponse,
    geolocation: string,
    relayer: Relayer
  ): Promise<RPCConsumer> {
    const rpcConsumer = new RPCConsumer(geolocation, relayer);
    // step 1 set the spec.
    rpcConsumer.setSpec(pairingResponse.spec);
    // step 2 create the consumer session manager map for each api interface.
    // Create Optimizer
    const optimizer = new RandomProviderOptimizer();
    for (const apiInterfaces of rpcConsumer.getApiInterfacesSupported()) {
      Logger.info(
        "Creating RPCConsumer for",
        pairingResponse.spec.index,
        apiInterfaces
      );
      const consumerSessionManager: ConsumerSessionManager =
        new ConsumerSessionManager(
          relayer,
          new RPCEndpoint(
            "",
            pairingResponse.spec.index,
            apiInterfaces,
            geolocation
          ),
          optimizer
        );
      // setup the providers for this api interface.
      await consumerSessionManager.updateAllProviders(
        pairingResponse.currentEpoch,
        rpcConsumer.filterPairingListByEndpoint(pairingResponse, apiInterfaces)
      );
      rpcConsumer.setConsumerSessionManager(
        consumerSessionManager,
        apiInterfaces
      );
    }
    return rpcConsumer;
  }

  public getApiInterfacesSupported(): IterableIterator<string> {
    return this.chainParser.keys();
  }

  setConsumerSessionManager(
    consumerSessionManager: ConsumerSessionManager,
    apiInterface: ApiInterface
  ) {
    let chainParser = this.chainParser.get(apiInterface);
    if (!chainParser) {
      chainParser = getChainParser(apiInterface, this.relayer); // create a new chain parser if missing.
    }
    chainParser.setConsumerSessionManager(consumerSessionManager);
    this.chainParser.set(apiInterface, chainParser);
  }

  setSpec(spec: Spec) {
    for (const apiCollection of spec.apiCollections) {
      if (!apiCollection.enabled) {
        continue;
      }
      const apiInterface = apiCollection.collectionData?.apiInterface;
      if (!apiInterface) {
        continue;
      }
      // reset / set the new spec.
      const baseChainParser = new BaseChainParser();
      baseChainParser.init(spec, apiInterface);

      let chainParser = this.chainParser.get(apiInterface);
      if (!chainParser) {
        chainParser = getChainParser(apiInterface, this.relayer); // create a new chain parser if missing.
      }
      chainParser.setBaseChainParser(baseChainParser);
      this.chainParser.set(apiInterface, chainParser);
    }
  }

  async updateAllProviders(pairingResponse: PairingResponse) {
    for (const apiInterfaces of this.getApiInterfacesSupported()) {
      Logger.info("Updating provider list for", apiInterfaces);
      const chainParser = this.chainParser.get(apiInterfaces);
      if (!chainParser) {
        throw Logger.fatal(
          "Consumer session manager was not found for an expected api interface",
          apiInterfaces
        );
      }
      const filteredEndpoints: ConsumerSessionsWithProvider[] =
        this.filterPairingListByEndpoint(pairingResponse, apiInterfaces);
      const err = await chainParser.updateAllProviders(
        pairingResponse.currentEpoch,
        filteredEndpoints
      );
      if (err) {
        throw Logger.fatal(
          "Received an error while updating provider list",
          err,
          apiInterfaces,
          pairingResponse.spec.index
        );
      }
    }
  }

  // filterPairingListByEndpoint filters pairing list and return only the once for rpcInterface
  private filterPairingListByEndpoint(
    pairing: PairingResponse,
    rpcInterface: string
  ): ConsumerSessionsWithProvider[] {
    // Initialize ConsumerSessionWithProvider array
    const pairingForSameGeolocation: Array<ConsumerSessionsWithProvider> = [];
    const pairingFromDifferentGeolocation: Array<ConsumerSessionsWithProvider> =
      [];
    // Iterate over providers to populate pairing list
    for (const provider of pairing.providers) {
      Logger.debug("parsing provider", provider);
      // Skip providers with no endpoints
      if (provider.endpoints.length == 0) {
        continue;
      }

      // Initialize relevantEndpoints array
      const sameGeoEndpoints: Array<Endpoint> = [];
      const differntGeoEndpoints: Array<Endpoint> = [];

      // Only take into account endpoints that use the same api interface
      // And geolocation
      for (const endpoint of provider.endpoints) {
        if (!endpoint.apiInterfaces.includes(rpcInterface)) {
          continue;
        }
        const convertedEndpoint = {
          networkAddress: endpoint.iPPORT,
          enabled: true,
          connectionRefusals: 0,
          addons: new Set(endpoint.addons),
          extensions: new Set(endpoint.extensions),
        };
        if (parseLong(endpoint.geolocation) == Number(this.geolocation)) {
          sameGeoEndpoints.push(convertedEndpoint); // set same geo location provider endpoint
        } else {
          differntGeoEndpoints.push(convertedEndpoint); // set different geo location provider endpoint
        }
      }

      // skip if we have no endpoints at all.
      if (sameGeoEndpoints.length == 0 && differntGeoEndpoints.length == 0) {
        Logger.debug("No endpoints found");
        continue;
      }

      let sameGeoOptions = false; // if we have same geolocation options or not
      let endpointListToStore: Endpoint[] = differntGeoEndpoints;
      if (sameGeoEndpoints.length > 0) {
        sameGeoOptions = true;
        endpointListToStore = sameGeoEndpoints;
      }

      const newPairing = new ConsumerSessionsWithProvider(
        provider.address,
        endpointListToStore,
        {},
        pairing.maxCu,
        pairing.currentEpoch
      );

      // Add newly created pairing in the pairing list
      if (sameGeoOptions) {
        pairingForSameGeolocation.push(newPairing);
      } else {
        pairingFromDifferentGeolocation.push(newPairing);
      }
    }

    if (
      pairingForSameGeolocation.length == 0 &&
      pairingFromDifferentGeolocation.length == 0
    ) {
      Logger.debug("No relevant providers found");
    }
    // Return providers list [pairingForSameGeolocation,pairingFromDifferentGeolocation]
    // TODO: might need to fix this, check after optimizer.
    return pairingForSameGeolocation.concat(pairingFromDifferentGeolocation);
  }

  public async sendRelay(
    relayOptions: SendRelayOptions | SendRestRelayOptions
  ) {
    // Todo
  }
}
