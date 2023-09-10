import { Logger } from "../logger/logger";
import { Relayer } from "../relayer/relayer";
import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";
import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  ChainMessage,
} from "../chainlib/base_chain_parser";
import { stringToArrayBuffer } from "@improbable-eng/grpc-web/dist/typings/transports/http/xhr";
import { newRelayData, SendRelayData } from "./lavaprotocol";
import { RPCEndpoint } from "../lavasession/consumerTypes";
import { RelayPrivateData } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";

const MaxRelayRetries = 4;

export class RPCConsumerServer {
  private consumerSessionManager: ConsumerSessionManager;
  private chainParser: BaseChainParser;
  private geolocation: string;
  private relayer: Relayer;
  private rpcEndpoint: RPCEndpoint;
  constructor(
    relayer: Relayer,
    consumerSessionManager: ConsumerSessionManager,
    chainParser: BaseChainParser,
    geolocation: string,
    rpcEndpoint: RPCEndpoint
  ) {
    this.consumerSessionManager = consumerSessionManager;
    this.geolocation = geolocation;
    this.chainParser = chainParser;
    this.relayer = relayer;
    this.rpcEndpoint = rpcEndpoint;
  }

  async sendRelay(options: SendRelayOptions | SendRestRelayOptions) {
    const chainMessage = this.chainParser.parseMsg(options);
    const unwantedProviders = new Set<string>();
    // TODO: fill data url, connectionType
    const relayData = {
      data: "",
      url: "",
      connectionType: "",
      apiInterface: this.rpcEndpoint.apiInterface,
      chainId: this.rpcEndpoint.chainId,
    };
    const relayPrivateData = newRelayData(relayData);
    const blockOnSyncLoss = true;
    for (let retries = 0; retries < MaxRelayRetries; retries++) {
      const relayResult = this.sendRelayToProvider(
        chainMessage,
        relayPrivateData,
        unwantedProviders
      );
    }
    // this.consumerSessionManager.getSessions()

    //TODO after reply if resolved, parse to json
    // // Decode response
    // const dec = new TextDecoder();
    // const decodedResponse = dec.decode(response.getData_asU8());

    // // Parse response
    // const jsonResponse = JSON.parse(decodedResponse);

    // // Return response
    // return jsonResponse;
  }

  private sendRelayToProvider(
    chainMessage: ChainMessage,
    relayData: RelayPrivateData,
    unwantedProviders: Set<string>
  ): RelayResult {
    return { request: "", reply: "", providerAddress: "", finalized: false };
  }
}

interface RelayResult {
  request: string;
  reply: string;
  providerAddress: string;
  finalized: boolean;
}
