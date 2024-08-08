import { DEFAULT_LAVA_PAIRING_LIST } from "../../config/default";
import { Config } from "../state_tracker";
import { fetchLavaPairing } from "../../util/lavaPairing";
import { StateTrackerErrors } from "../errors";
import { PairingResponse } from "./state_query";
import { AccountData } from "@cosmjs/proto-signing";
import { base64ToUint8Array, generateRandomInt } from "../../util/common";
import {
  QueryGetPairingRequest,
  QuerySdkPairingResponse,
} from "../../grpc_web_services/lavanet/lava/pairing/query_pb";
import { ProvidersErrors } from "../errors";
import {
  ConsumerSessionsWithProvider,
  SingleConsumerSession,
  Endpoint,
} from "../../lavasession/consumerTypes";
import { Logger } from "../../logger/logger";
import { RPCConsumerServer } from "../../rpcconsumer/rpcconsumer_server";
import { SendRelayOptions } from "../../chainlib/base_chain_parser";
import { Spec } from "../../grpc_web_services/lavanet/lava/spec/spec_pb";
import { StakeEntry } from "../../grpc_web_services/lavanet/lava/epochstorage/stake_entry_pb";
import { Endpoint as PairingEndpoint } from "../../grpc_web_services/lavanet/lava/epochstorage/endpoint_pb";
import { GeolocationFromString } from "../../lavasession/geolocation";
import { Params as DowntimeParams } from "../../grpc_web_services/lavanet/lava/downtime/v1/downtime_pb";

interface PairingList {
  stakeEntry: StakeEntry[];
  consumerSessionsWithProvider: ConsumerSessionsWithProvider[];
}

export class StateChainQuery {
  private pairingListConfig: string;
  private chainIDs: string[];
  private rpcConsumer: RPCConsumerServer;
  private config: Config;
  private pairing: Map<string, PairingResponse | undefined>;
  private account: AccountData;
  private latestBlockNumber = 0;
  private lavaSpec: Spec;
  private csp: ConsumerSessionsWithProvider[] = [];
  private currentEpoch: number | undefined;
  private downtimeParams: DowntimeParams | undefined;

  constructor(
    pairingListConfig: string,
    chainIDs: string[],
    rpcConsumer: RPCConsumerServer,
    config: Config,
    account: AccountData,
    lavaSpec: Spec
  ) {
    Logger.debug("Initialization of State Chain Query started");

    // Save arguments
    this.pairingListConfig = pairingListConfig;
    this.chainIDs = chainIDs;
    this.rpcConsumer = rpcConsumer;
    this.config = config;
    this.account = account;
    this.lavaSpec = lavaSpec;

    // Initialize pairing to an empty map
    this.pairing = new Map<string, PairingResponse>();

    Logger.debug("Initialization of State Chain Query ended");
  }

  public async init(): Promise<void> {
    const pairing = await this.fetchLavaProviders(this.pairingListConfig);
    this.csp = pairing.consumerSessionsWithProvider;
    // Save pairing response for chainID
    this.pairing.set("LAV1", {
      providers: pairing.stakeEntry,
      maxCu: 10000,
      currentEpoch: 0,
      spec: this.lavaSpec,
    });
  }

  // fetchPairing fetches pairing for all chainIDs we support
  public async fetchPairing(): Promise<number> {
    try {
      Logger.debug("Fetching pairing from chain started");
      // Save time till next epoch
      let timeLeftToNextPairing;
      let currentEpoch;
      let downtimeParams;

      const lavaPairing = this.getPairing("LAV1");

      // Reset pairing

      this.pairing = new Map<string, PairingResponse>();

      // Save lava pairing
      // as if we do not have lava in chainID it can fail updating list
      this.pairing.set("LAV1", lavaPairing);

      // Iterate over chain and construct pairing
      for (const chainID of this.chainIDs) {
        const request = new QueryGetPairingRequest();
        request.setChainid(chainID);
        request.setClient(this.account.address);

        // Fetch pairing for specified chainID
        const pairingResponse = await this.getPairingFromChain(request);

        // If pairing is undefined set to empty object
        if (!pairingResponse) {
          this.pairing.set(chainID, undefined);
          continue;
        }

        const pairing = pairingResponse.getPairing();

        if (!pairing) {
          this.pairing.set(chainID, undefined);
          continue;
        }

        const spec = pairingResponse.getSpec();

        if (!spec) {
          this.pairing.set(chainID, undefined);
          continue;
        }

        downtimeParams = pairingResponse.getDowntimeParams();

        const providers = pairing.getProvidersList();
        timeLeftToNextPairing = pairing.getTimeLeftToNextPairing();
        currentEpoch = pairingResponse.getPairing()?.getCurrentEpoch();
        if (!currentEpoch) {
          throw Logger.fatal(
            "Failed fetching current epoch from pairing request."
          );
        }

        // Save pairing response for chainID
        this.pairing.set(chainID, {
          providers: providers,
          maxCu: pairingResponse.getMaxCu(),
          currentEpoch: currentEpoch,
          spec: spec,
        });
      }

      // If timeLeftToNextPairing undefined return an error
      if (timeLeftToNextPairing == undefined) {
        throw StateTrackerErrors.errTimeTillNextEpochMissing;
      }

      this.currentEpoch = currentEpoch;
      this.downtimeParams = downtimeParams;

      Logger.debug("Fetching pairing from chain ended", timeLeftToNextPairing);

      // Return timeLeftToNextPairing
      return timeLeftToNextPairing;
    } catch (err) {
      throw err;
    }
  }

  public getCurrentEpoch(): number | undefined {
    return this.currentEpoch;
  }

  public getDowntimeParams(): DowntimeParams | undefined {
    return this.downtimeParams;
  }

  // getPairing return pairing list for specific chainID
  public getPairing(chainID: string): PairingResponse | undefined {
    // Return pairing for the specific chainId from the map
    return this.pairing.get(chainID);
  }

  //fetchLavaProviders fetches lava providers from different sources
  private async fetchLavaProviders(
    pairingListConfig: string
  ): Promise<PairingList> {
    try {
      Logger.debug("Fetching lava providers started");

      // Else if pairingListConfig exists use it to fetch lava providers from local file
      if (pairingListConfig != "") {
        const pairingList = await this.fetchLocalLavaPairingList(
          pairingListConfig
        );

        const providers = this.constructLavaPairing(pairingList);
        // Construct lava providers from pairing list and return it
        return providers;
      }

      // Fetch pairing from default lava pairing list
      const pairingList = await this.fetchDefaultLavaPairingList();

      const providers = this.constructLavaPairing(pairingList);

      // Construct lava providers from pairing list and return it
      return providers;
    } catch (err) {
      throw err;
    }
  }

  // getPairingFromChain fetch pairing response from lava providers
  private async getPairingFromChain(
    request: QueryGetPairingRequest
  ): Promise<QuerySdkPairingResponse | undefined> {
    try {
      Logger.debug("Get pairing for:" + request.getChainid() + " started");

      // Serialize request to binary format
      const requestData = request.serializeBinary();

      // Create hex from data
      const hexData = Buffer.from(requestData).toString("hex");

      // Init send relay options
      const sendRelayOptions: SendRelayOptions = {
        method: "abci_query",
        params: ["/lavanet.lava.pairing.Query/SdkPairing", hexData, "0", false],
      };

      const response = await this.rpcConsumer.sendRelay(sendRelayOptions);

      const reply = response.reply;

      if (reply == undefined) {
        throw new Error("Reply undefined");
      }

      // Decode response
      const dec = new TextDecoder();
      const decodedResponse = dec.decode(reply.getData_asU8());

      // Parse response
      const jsonResponse = JSON.parse(decodedResponse);

      // If log is not empty
      // return an error
      if (jsonResponse.result.response.log != "") {
        Logger.error(
          "Failed fetching pairing list for: ",
          request.getChainid()
        );
        throw new Error(jsonResponse.result.response.log);
      }

      const byteArrayResponse = base64ToUint8Array(
        jsonResponse.result.response.value
      );

      // Deserialize the Uint8Array to obtain the protobuf message
      const decodedResponse2 =
        QuerySdkPairingResponse.deserializeBinary(byteArrayResponse);

      // If response undefined throw an error
      if (decodedResponse2.getPairing() == undefined) {
        throw ProvidersErrors.errProvidersNotFound;
      }

      Logger.debug("Get pairing for:" + request.getChainid() + " ended");
      // Return decoded response
      return decodedResponse2;
    } catch (err) {
      // Console log the error
      Logger.error(err);

      // Return empty object
      // We do not want to return error because it will stop the state tracker for other chains
      return undefined;
    }
  }

  // fetchLocalLavaPairingList uses local pairingList.json file to load lava providers
  private async fetchLocalLavaPairingList(path: string): Promise<any> {
    Logger.debug("Fetch pairing list from local config");

    try {
      const data = await fetchLavaPairing(path);
      return this.validatePairingData(data);
    } catch (err) {
      Logger.debug("Error happened in fetchLocalLavaPairingList", err);
      throw err;
    }
  }

  // fetchLocalLavaPairingList fetch lava pairing from seed providers list
  private async fetchDefaultLavaPairingList(): Promise<any> {
    Logger.debug("Fetch pairing list from seed providers in github");

    // Fetch lava providers from seed list
    const response = await fetch(DEFAULT_LAVA_PAIRING_LIST);

    // Validate response
    if (!response.ok) {
      throw Logger.fatal(
        `Unable to fetch pairing list: ${response.statusText}`
      );
    }

    try {
      // Parse response
      const data = await response.json();

      return this.validatePairingData(data);
    } catch (error) {
      throw Logger.fatal(
        "Error happened in fetchDefaultLavaPairingList",
        error
      );
    }
  }

  // constructLavaPairing constructs consumer session with provider list from pairing list
  private constructLavaPairing(pairingList: any): PairingList {
    try {
      // Initialize ConsumerSessionWithProvider array
      const pairing: Array<StakeEntry> = [];
      const pairingEndpoints: Array<PairingEndpoint> = [];
      // Initialize ConsumerSessionWithProvider array
      const csmArr: Array<ConsumerSessionsWithProvider> = [];
      for (const provider of pairingList) {
        const pairingEndpoint = new PairingEndpoint();
        pairingEndpoint.setIpport(provider.rpcAddress);
        pairingEndpoint.setApiInterfacesList(["tendermintrpc"]);
        pairingEndpoint.setGeolocation(
          GeolocationFromString(this.config.geolocation)
        );
        // Add newly created endpoint in the pairing endpoint list
        pairingEndpoints.push(pairingEndpoint);

        const endpoint: Endpoint = {
          networkAddress: provider.rpcAddress,
          enabled: true,
          connectionRefusals: 0,
          addons: new Set<string>(),
          extensions: new Set<string>(),
          geolocation: GeolocationFromString(this.config.geolocation),
        };

        // Create a new pairing object
        const newPairing = new ConsumerSessionsWithProvider(
          provider.publicAddress,
          [endpoint],
          {},
          1000,
          0
        );

        const stakeEntry = new StakeEntry();
        stakeEntry.setEndpointsList([pairingEndpoint]);
        stakeEntry.setAddress(provider.publicAddress);

        pairing.push(stakeEntry);

        const randomSessionId = generateRandomInt();
        const singleConsumerSession = new SingleConsumerSession(
          randomSessionId,
          newPairing,
          endpoint
        );

        newPairing.sessions[0] = singleConsumerSession;

        // Add newly created pairing in the pairing list
        csmArr.push(newPairing);
      }

      return {
        stakeEntry: pairing,
        consumerSessionsWithProvider: csmArr,
      };
    } catch (err) {
      throw err;
    }
  }

  // validatePairingData validates pairing data
  private validatePairingData(data: any): any {
    if (data[this.config.network] == undefined) {
      throw Logger.fatal(
        `Unsupported network (${
          this.config.network
        }), supported networks: ${Object.keys(data)}`
      );
    }

    if (data[this.config.network][this.config.geolocation] == undefined) {
      throw Logger.fatal(
        `Unsupported geolocation (${this.config.geolocation}) for network (${
          this.config.network
        }). Supported geolocations: ${Object.keys(data[this.config.network])}`
      );
    }
    return data[this.config.network][this.config.geolocation];
  }
}
