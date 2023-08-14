import { DEFAULT_LAVA_PAIRING_LIST } from "../config/default";
import {
  ConsumerSessionWithProvider,
  SingleConsumerSession,
  Endpoint,
  SessionManager,
} from "../types/types";
import {
  QueryGetPairingRequest,
  QueryGetPairingResponse,
  QueryUserEntryRequest,
  QueryUserEntryResponse,
} from "../codec/lavanet/lava/pairing/query";
import {
  QueryGetSpecRequest,
  QueryGetSpecResponse,
} from "../codec/lavanet/lava/spec/query";
import { fetchLavaPairing } from "../util/lavaPairing";
import Relayer from "../relayer/relayer";
import ProvidersErrors from "./errors";
import { base64ToUint8Array, generateRPCData, parseLong } from "../util/common";
import { Badge } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import { QueryShowAllChainsResponse } from "../codec/lavanet/lava/spec/query";

const BOOT_RETRY_ATTEMPTS = 2;
export interface LavaProvidersOptions {
  accountAddress: string;
  network: string;
  relayer: Relayer | null;
  geolocation: string;
  debug?: boolean;
}

export class LavaProviders {
  private providers: ConsumerSessionWithProvider[];
  private network: string;
  private index = 0;
  private accountAddress: string;
  private relayer: Relayer | null;
  private geolocation: string;
  private debugMode: boolean;

  constructor(options: LavaProvidersOptions) {
    this.providers = [];
    this.network = options.network;
    this.accountAddress = options.accountAddress;
    this.relayer = options.relayer;
    this.geolocation = options.geolocation;
    this.debugMode = options.debug ? options.debug : false;
  }

  public updateLavaProvidersRelayersBadge(badge: Badge | undefined) {
    if (this.relayer) {
      this.relayer.setBadge(badge);
    }
  }

  async init(pairingListConfig: string) {
    let data;

    // if no pairing list config use default
    if (pairingListConfig == "") {
      data = await this.initDefaultConfig();
    } else {
      // Else use local config file
      data = await this.initLocalConfig(pairingListConfig);
    }
    // Initialize ConsumerSessionWithProvider array
    const pairing: Array<ConsumerSessionWithProvider> = [];

    for (const provider of data) {
      const singleConsumerSession = new SingleConsumerSession(
        0, // cuSum
        0, // latestRelayCuSum
        1, // relayNumber
        new Endpoint(provider.rpcAddress, true, 0),
        -1, //invalid epoch
        provider.publicAddress
      );

      // Create a new pairing object
      const newPairing = new ConsumerSessionWithProvider(
        this.accountAddress,
        [],
        singleConsumerSession,
        100000, // invalid max cu
        0, // used compute units
        false
      );

      // Add newly created pairing in the pairing list
      pairing.push(newPairing);
    }
    // Save providers as local attribute
    this.providers = pairing;
  }

  public async showAllChains(): Promise<QueryShowAllChainsResponse> {
    const sendRelayOptions = {
      data: generateRPCData("abci_query", [
        "/lavanet.lava.spec.Query/ShowAllChains",
        "",
        "0",
        false,
      ]),
      url: "",
      connectionType: "",
    };

    const info = await this.SendRelayToAllProvidersAndRace(
      sendRelayOptions,
      10,
      "tendermintrpc"
    );
    const byteArrayResponse = base64ToUint8Array(info.result.response.value);
    const response = QueryShowAllChainsResponse.decode(byteArrayResponse);
    return response;
  }

  async initDefaultConfig(): Promise<any> {
    // Fetch config from github repo
    const response = await fetch(DEFAULT_LAVA_PAIRING_LIST);

    // Validate response
    if (!response.ok) {
      throw new Error(`Unable to fetch pairing list: ${response.statusText}`);
    }

    try {
      // Parse response
      const data = await response.json();

      if (data[this.network] == undefined) {
        throw new Error(
          `Unsupported network (${
            this.network
          }), supported networks: ${Object.keys(data)}, seed pairing list used`
        );
      }

      if (data[this.network][this.geolocation] == undefined) {
        throw new Error(
          `Unsupported geolocation (${this.geolocation}) for network (${
            this.network
          }). Supported geolocations: ${Object.keys(
            data[this.network]
          )}, seed pairing list used`
        );
      }
      // Return data array
      return data[this.network][this.geolocation];
    } catch (error) {
      throw error;
    }
  }

  async initLocalConfig(path: string): Promise<any> {
    try {
      const data = await fetchLavaPairing(path);
      if (data[this.network] == undefined) {
        throw new Error(
          `Unsupported network (${
            this.network
          }), supported networks: ${Object.keys(data)}, local pairing list used`
        );
      }

      if (data[this.network][this.geolocation] == undefined) {
        throw new Error(
          `Unsupported geolocation (${this.geolocation}) for network (${
            this.network
          }). Supported geolocations: ${Object.keys(
            data[this.network]
          )}, local pairing list used`
        );
      }
      return data[this.network][this.geolocation];
    } catch (err) {
      throw err;
    }
  }

  // private getLavaProvidersInBatches() {

  // }

  // GetLavaProviders returns lava providers list
  GetLavaProviders(): ConsumerSessionWithProvider[] {
    if (this.providers.length == 0) {
      throw ProvidersErrors.errNoProviders;
    }

    return this.providers;
  }

  // GetNextLavaProvider returns lava providers used for fetching epoch
  // in round-robin fashion
  GetNextLavaProvider(): ConsumerSessionWithProvider {
    if (this.providers.length == 0) {
      throw ProvidersErrors.errNoProviders;
    }

    const rpcAddress = this.providers[this.index];
    this.index = (this.index + 1) % this.providers.length;
    return rpcAddress;
  }

  // getSession returns providers for current epoch
  async getSession(
    chainID: string,
    rpcInterface: string,
    badge?: Badge
  ): Promise<SessionManager> {
    if (this.providers == null) {
      throw ProvidersErrors.errLavaProvidersNotInitialized;
    }

    // Create request for fetching api methods for LAV1
    const lavaApis = await this.getServiceApis(
      { ChainID: "LAV1" },
      "grpc",
      new Map([["lavanet.lava.spec.Query/Spec", 10]])
    );

    // Create request for getServiceApis method for chainID
    const apis = await this.getServiceApis(
      { ChainID: chainID },
      rpcInterface,
      lavaApis
    );

    // Create pairing request for getPairing method
    const pairingRequest = {
      chainID: chainID,
      client: this.accountAddress,
    };

    // Get pairing from the chain
    const pairingResponse = await this.getPairingFromChain(
      pairingRequest,
      lavaApis
    );

    // Set when will next epoch start
    const nextEpochStart = new Date();
    nextEpochStart.setSeconds(
      nextEpochStart.getSeconds() +
        parseLong(pairingResponse.timeLeftToNextPairing)
    );

    // Extract providers from pairing response
    const providers = pairingResponse.providers;

    // Create request for getting userEntity
    const userEntityRequest = {
      address: this.accountAddress,
      chainID: chainID,
      block: pairingResponse.currentEpoch,
    };

    // Fetch max compute units
    let maxcu: number;
    if (badge) {
      maxcu = badge.getCuAllocation();
    } else {
      maxcu = await this.getMaxCuForUser(userEntityRequest, lavaApis);
    }

    // Initialize ConsumerSessionWithProvider array
    const pairing: Array<ConsumerSessionWithProvider> = [];
    const pairingFromDifferentGeolocation: Array<ConsumerSessionWithProvider> =
      [];
    // Iterate over providers to populate pairing list
    for (const provider of providers) {
      // Skip providers with no endpoints
      this.debugPrint("parsing provider", provider);
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
        const convertedEndpoint = new Endpoint(endpoint.iPPORT, true, 0);
        if (parseLong(endpoint.geolocation) == Number(this.geolocation)) {
          sameGeoEndpoints.push(convertedEndpoint); // set same geo location provider endpoint
        } else {
          differntGeoEndpoints.push(convertedEndpoint); // set different geo location provider endpoint
        }
      }

      // skip if we have no endpoints at all.
      if (sameGeoEndpoints.length == 0 && differntGeoEndpoints.length == 0) {
        continue;
      }

      let sameGeoOptions = false; // if we have same geolocation options or not
      let endpointListToStore: Endpoint[] = differntGeoEndpoints;
      if (sameGeoEndpoints.length > 0) {
        sameGeoOptions = true;
        endpointListToStore = sameGeoEndpoints;
      }

      // create single consumer session from pairing.
      const singleConsumerSession = new SingleConsumerSession(
        0, // cuSum
        0, // latestRelayCuSum
        1, // relayNumber
        endpointListToStore[0],
        parseLong(pairingResponse.currentEpoch),
        provider.address
      );

      // Create a new pairing object
      const newPairing = new ConsumerSessionWithProvider(
        this.accountAddress,
        endpointListToStore,
        singleConsumerSession,
        maxcu,
        0, // used compute units
        false
      );

      // Add newly created pairing in the pairing list
      if (sameGeoOptions) {
        pairing.push(newPairing);
      } else {
        pairingFromDifferentGeolocation.push(newPairing);
      }
    }

    if (pairing.length == 0 && pairingFromDifferentGeolocation.length == 0) {
      throw new Error("Could not find any relevant pairing");
    }
    // Create session object from both pairing lists.
    const sessionManager = new SessionManager(
      pairing,
      pairingFromDifferentGeolocation,
      nextEpochStart,
      apis
    );
    this.debugPrint("SessionManager Object", sessionManager);

    return sessionManager;
  }

  private debugPrint(message?: any, ...optionalParams: any[]) {
    if (this.debugMode) {
      console.log(message, ...optionalParams);
    }
  }

  pickRandomProviders(
    providers: Array<ConsumerSessionWithProvider>
  ): ConsumerSessionWithProvider[] {
    // Remove providers which does not match criteria
    this.debugPrint("pickRandomProviders Providers list", providers);
    const validProviders = providers.filter(
      (item) => item.MaxComputeUnits > item.UsedComputeUnits
    );

    if (validProviders.length <= 1) {
      return validProviders; // returning the array as it is.
    }

    // Fisher-Yates shuffle
    for (let i = validProviders.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [validProviders[i], validProviders[j]] = [
        validProviders[j],
        validProviders[i],
      ];
    }

    return validProviders;
  }

  pickRandomProvider(
    providers: Array<ConsumerSessionWithProvider>
  ): ConsumerSessionWithProvider {
    this.debugPrint("pickRandomProvider Providers list", providers);
    // Remove providers which does not match criteria
    const validProviders = providers.filter(
      (item) => item.MaxComputeUnits > item.UsedComputeUnits
    );

    if (validProviders.length === 0) {
      throw ProvidersErrors.errNoValidProvidersForCurrentEpoch;
    }

    // Pick random provider
    const random = Math.floor(Math.random() * validProviders.length);

    return validProviders[random];
  }

  private async getPairingFromChain(
    request: QueryGetPairingRequest,
    lavaApis: Map<string, number>
  ): Promise<QueryGetPairingResponse> {
    const requestData = QueryGetPairingRequest.encode(request).finish();

    const hexData = Buffer.from(requestData).toString("hex");

    const sendRelayOptions = {
      data: generateRPCData("abci_query", [
        "/lavanet.lava.pairing.Query/GetPairing",
        hexData,
        "0",
        false,
      ]),
      url: "",
      connectionType: "",
    };
    const relayCu = lavaApis.get("lavanet.lava.pairing.Query/GetPairing");

    if (relayCu == undefined) {
      throw ProvidersErrors.errApiNotFound;
    }

    const jsonResponse = await this.SendRelayToAllProvidersAndRace(
      sendRelayOptions,
      relayCu,
      "tendermintrpc"
    );
    const byteArrayResponse = base64ToUint8Array(
      jsonResponse.result.response.value
    );
    const decodedResponse = QueryGetPairingResponse.decode(byteArrayResponse);

    if (decodedResponse.providers == undefined) {
      throw ProvidersErrors.errProvidersNotFound;
    }

    return decodedResponse;
  }

  private async getMaxCuForUser(
    request: QueryUserEntryRequest,
    lavaApis: Map<string, number>
  ): Promise<number> {
    const requestData = QueryUserEntryRequest.encode(request).finish();

    const hexData = Buffer.from(requestData).toString("hex");

    const sendRelayOptions = {
      data: generateRPCData("abci_query", [
        "/lavanet.lava.pairing.Query/UserEntry",
        hexData,
        "0",
        false,
      ]),
      url: "",
      connectionType: "",
    };
    const relayCu = lavaApis.get("lavanet.lava.pairing.Query/UserEntry");

    if (relayCu == undefined) {
      throw ProvidersErrors.errApiNotFound;
    }

    const jsonResponse = await this.SendRelayToAllProvidersAndRace(
      sendRelayOptions,
      relayCu,
      "tendermintrpc"
    );
    const byteArrayResponse = base64ToUint8Array(
      jsonResponse.result.response.value
    );
    const response = QueryUserEntryResponse.decode(byteArrayResponse);

    if (response.maxCU == undefined) {
      throw ProvidersErrors.errMaxCuNotFound;
    }

    // return maxCu from userEntry
    return parseLong(response.maxCU);
  }

  private async getServiceApis(
    request: QueryGetSpecRequest,
    rpcInterface: string,
    lavaApis: Map<string, number>
  ): Promise<Map<string, number>> {
    const requestData = QueryGetSpecRequest.encode(request).finish();

    const hexData = Buffer.from(requestData).toString("hex");

    const sendRelayOptions = {
      data: generateRPCData("abci_query", [
        "/lavanet.lava.spec.Query/Spec",
        hexData,
        "0",
        false,
      ]),
      url: "",
      connectionType: "",
    };
    const relayCu = lavaApis.get("lavanet.lava.spec.Query/Spec");

    if (relayCu == undefined) {
      throw ProvidersErrors.errApiNotFound;
    }

    const jsonResponse = await this.SendRelayToAllProvidersAndRace(
      sendRelayOptions,
      relayCu,
      "tendermintrpc"
    );
    const byteArrayResponse = base64ToUint8Array(
      jsonResponse.result.response.value
    );
    const response = QueryGetSpecResponse.decode(byteArrayResponse);

    if (response.Spec == undefined) {
      throw ProvidersErrors.errSpecNotFound;
    }

    const apis = new Map<string, number>();

    // Extract apis from response
    for (const element of response.Spec.apiCollections) {
      if (!element.enabled) {
        continue;
      }
      // Skip if interface which does not match
      if (element.collectionData?.apiInterface != rpcInterface) continue;

      for (const api of element.apis) {
        if (element.collectionData?.apiInterface == "rest") {
          // handle REST apis
          const name = this.convertRestApiName(api.name);
          apis.set(name, parseLong(api.computeUnits));
        } else {
          // Handle RPC apis
          apis.set(api.name, parseLong(api.computeUnits));
        }
      }
    }

    return apis;
  }

  convertRestApiName(name: string): string {
    const regex = /\{\s*[^}]+\s*\}/g;
    return name.replace(regex, "[^/s]+");
  }

  async SendRelayToAllProvidersAndRace(
    options: any,
    relayCu: number,
    rpcInterface: string
  ): Promise<any> {
    let lastError;
    // let lavaProviders = this.GetLavaProviders();
    for (
      let retryAttempt = 0;
      retryAttempt < BOOT_RETRY_ATTEMPTS;
      retryAttempt++
    ) {
      const allRelays: Map<string, Promise<any>> = new Map();
      let response;
      for (const provider of this.GetLavaProviders()) {
        const uniqueKey =
          provider.Session.ProviderAddress +
          String(Math.floor(Math.random() * 10000000));
        const providerRelayPromise = this.SendRelayWithRetry(
          options,
          provider,
          relayCu,
          rpcInterface
        )
          .then((result) => {
            this.debugPrint("response succeeded", result);
            response = result;
            allRelays.delete(uniqueKey);
          })
          .catch((err: any) => {
            this.debugPrint(
              "one of the promises failed in SendRelayToAllProvidersAndRace reason:",
              err,
              uniqueKey
            );
            lastError = err;
            allRelays.delete(uniqueKey);
          });
        allRelays.set(uniqueKey, providerRelayPromise);
      }
      const promisesToWait = allRelays.size;
      for (let i = 0; i < promisesToWait; i++) {
        const returnedPromise = await Promise.race(allRelays);
        await returnedPromise[1];
        if (response) {
          this.debugPrint(
            "SendRelayToAllProvidersAndRace, got response from one provider",
            response
          );
          return response;
        }
      }
      this.debugPrint(
        "Failed all promises SendRelayToAllProvidersAndRace, trying again",
        retryAttempt,
        "out of",
        BOOT_RETRY_ATTEMPTS
      );
    }
    throw new Error(
      "Failed all promises SendRelayToAllProvidersAndRace: " + String(lastError)
    );
  }

  async SendRelayWithRetry(
    options: any,
    lavaRPCEndpoint: ConsumerSessionWithProvider,
    relayCu: number,
    rpcInterface: string
  ): Promise<any> {
    let response;
    if (this.relayer == null) {
      throw ProvidersErrors.errNoRelayer;
    }
    try {
      // For now we have hardcode relay cu
      response = await this.relayer.sendRelay(
        options,
        lavaRPCEndpoint,
        relayCu,
        rpcInterface
      );
    } catch (error) {
      // If error is instace of Error
      if (error instanceof Error) {
        // If error is not old blokc height throw and error
        // Extract current block height from error
        const currentBlockHeight = this.extractBlockNumberFromError(error);

        // If current block height equal nill throw an error
        if (currentBlockHeight == null) {
          throw error;
        }

        // Save current block height
        lavaRPCEndpoint.Session.PairingEpoch = parseInt(currentBlockHeight);

        // Retry same relay with added block height
        try {
          response = await this.relayer.sendRelay(
            options,
            lavaRPCEndpoint,
            relayCu,
            rpcInterface
          );
        } catch (error) {
          throw error;
        }
      }
    }

    // Validate that response is not undefined
    if (response == undefined) {
      return "";
    }

    // Decode response
    const dec = new TextDecoder();
    const decodedResponse = dec.decode(response.getData_asU8());

    // Parse response
    const jsonResponse = JSON.parse(decodedResponse);

    // Return response
    return jsonResponse;
  }

  private extractBlockNumberFromError(error: Error): string | null {
    let currentBlockHeightRegex = /current epoch Value:(\d+)/;
    let match = error.message.match(currentBlockHeightRegex);

    // Retry with new error
    if (match == null) {
      currentBlockHeightRegex = /current epoch: (\d+)/; // older epoch parsing

      match = error.message.match(currentBlockHeightRegex);
      return match ? match[1] : null;
    }
    return match ? match[1] : null;
  }
}
