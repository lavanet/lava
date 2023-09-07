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
import {
  RelayPrivateData,
  RelayReply,
  RelayRequest,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import SDKErrors from "../sdk/errors";

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
      ...chainMessage.getRawRequestData(), // url and data fields
      connectionType:
        chainMessage.getApiCollection().getCollectionData()?.getType() ?? "",
      apiInterface: this.rpcEndpoint.apiInterface,
      chainId: this.rpcEndpoint.chainId,
    };
    const relayPrivateData = newRelayData(relayData);
    let blockOnSyncLoss = true;
    const errors = new Array<Error>();
    for (let retries = 0; retries < MaxRelayRetries; retries++) {
      const relayResult = this.sendRelayToProvider(
        chainMessage,
        relayPrivateData,
        unwantedProviders
      );

      if (relayResult instanceof RelayError) {
        if (blockOnSyncLoss && relayResult.err == SDKErrors.sessionSyncLoss) {
          Logger.debug(
            "Identified SyncLoss in provider, not removing it from list for another attempt"
          );
          blockOnSyncLoss = false;
        } else {
          unwantedProviders.add(relayResult.providerAddress);
        }
      }
      if (relayResult instanceof RelayError) {
        errors.push(relayResult.err);
      } else if (relayResult instanceof Error) {
        errors.push(relayResult);
      } else {
        if (errors.length > 0) {
          Logger.debug("relay succeeded but had some errors", ...errors);
        }
        return relayResult;
      }
    }
    // got here if didn't succeed in any of the relays
    throw new Error("failed all retries " + errors.join(","));
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
  ): RelayResult | RelayError | Error {
    return {
      request: undefined,
      reply: undefined,
      providerAddress: "",
      finalized: false,
    };
  }
}

class RelayError {
  public providerAddress: string;
  public err: Error;
  constructor(address: string, err: Error) {
    this.providerAddress = address;
    this.err = err;
  }
}

interface RelayResult {
  request: RelayRequest | undefined;
  reply: RelayReply | undefined;
  providerAddress: string;
  finalized: boolean;
}
