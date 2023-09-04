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

type ApiInterface = string;
class RPCConsumer {
  private consumerSessionManagerMap: Map<ApiInterface, ConsumerSessionManager>;
  private chainParserMap: Map<ApiInterface, BaseChainParser>;
  private geolocation: string;
  constructor(geolocation: string) {
    this.consumerSessionManagerMap = new Map();
    this.chainParserMap = new Map();
    this.geolocation = geolocation;
  }

  static create(
    pairingResponse: PairingResponse,
    geolocation: string,
    accountAddress: string,
    relayer: Relayer
  ): RPCConsumer {
    const rpcConsumer = new RPCConsumer(geolocation);
    // step 1 set the spec.
    rpcConsumer.setSpec(pairingResponse.spec);
    // step 2 create the consumer session manager map for each api interface.
    // Create Optimizer
    const optimizer = new RandomProviderOptimizer();
    for (const apiInterfaces of rpcConsumer.getApiInterfacesSupported()) {
      const consumerSessionManager: ConsumerSessionManager =
        new ConsumerSessionManager(
          relayer,
          new RPCEndpoint(
            accountAddress,
            pairingResponse.spec.index,
            apiInterfaces,
            geolocation
          ),
          optimizer
        );

      rpcConsumer.setConsumerSessionManager(
        consumerSessionManager,
        apiInterfaces
      );
    }

    return rpcConsumer;
  }

  public getApiInterfacesSupported(): IterableIterator<string> {
    return this.chainParserMap.keys();
  }

  setConsumerSessionManager(
    consumerSessionManager: ConsumerSessionManager,
    apiInterface: ApiInterface
  ) {
    this.consumerSessionManagerMap.set(apiInterface, consumerSessionManager);
  }

  setSpec(spec: Spec) {
    // if (!apiCollection.enabled) {
    //     continue;
    //   }
    //   if (apiCollection.collectionData?.apiInterface != this.apiInterface) {
    //     continue;
    //   }
  }

  updateAllProviders(
    pairingResponse: PairingResponse
  ): Promise<Error | undefined> {}

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
}

type ChainId = string;

export class Consumer {
  private rpcConsumer: Map<ChainId, RPCConsumer>;
  private geolocation: string;
  private accountAddress: string;
  private relayer: Relayer;
  constructor(relayer: Relayer, accountAddress: string, geolocation: string) {
    this.rpcConsumer = new Map();
    this.geolocation = geolocation;
    this.accountAddress = accountAddress;
    this.relayer = relayer;
  }

  public async updateAllProviders(
    pairingResponse: PairingResponse
  ): Promise<Error | undefined> {
    const chainId = pairingResponse.spec.index;
    const rpcConsumer = this.rpcConsumer.get(chainId);
    if (!rpcConsumer) {
      // initialize the rpcConsumer
      this.rpcConsumer.set(
        chainId,
        RPCConsumer.create(
          pairingResponse,
          this.geolocation,
          this.accountAddress,
          this.relayer
        )
      );
      return;
    }
    return rpcConsumer.updateAllProviders(pairingResponse);
  }

  //   public sendRelay()
  //   implement send relay to the right chain id and rpc interface.
}
