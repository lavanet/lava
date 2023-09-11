import { Logger } from "../logger/logger";
import { Relayer } from "../relayer/relayer";
import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";
import { SessionInfo } from "../lavasession/consumerTypes";
import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  ChainMessage,
} from "../chainlib/base_chain_parser";
import { constructRelayRequest, newRelayData } from "./lavaprotocol";
import { RPCEndpoint } from "../lavasession/consumerTypes";
import {
  RelayPrivateData,
  RelayReply,
  RelayRequest,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import SDKErrors from "../sdk/errors";
import { AverageWorldLatency, getTimePerCu } from "../common/timeout";

const MaxRelayRetries = 4;

export class RPCConsumerServer {
  private consumerSessionManager: ConsumerSessionManager;
  private chainParser: BaseChainParser;
  private geolocation: string;
  private relayer: Relayer;
  private rpcEndpoint: RPCEndpoint;
  private lavaChainId: string;
  constructor(
    relayer: Relayer,
    consumerSessionManager: ConsumerSessionManager,
    chainParser: BaseChainParser,
    geolocation: string,
    rpcEndpoint: RPCEndpoint,
    lavaChainId: string
  ) {
    this.consumerSessionManager = consumerSessionManager;
    this.geolocation = geolocation;
    this.chainParser = chainParser;
    this.relayer = relayer;
    this.rpcEndpoint = rpcEndpoint;
    this.lavaChainId = lavaChainId;
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
      const relayResult = await this.sendRelayToProvider(
        chainMessage,
        relayPrivateData,
        unwantedProviders
      );

      if (relayResult instanceof Array) {
        for (const oneResult of relayResult) {
          if (blockOnSyncLoss && oneResult.err == SDKErrors.sessionSyncLoss) {
            Logger.debug(
              "Identified SyncLoss in provider, not removing it from list for another attempt"
            );
            blockOnSyncLoss = false;
          } else {
            unwantedProviders.add(oneResult.providerAddress);
          }
          errors.push(oneResult.err);
        }
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
    //

    //TODO after reply if resolved, parse to json
    // // Decode response
    // const dec = new TextDecoder();
    // const decodedResponse = dec.decode(response.getData_asU8());

    // // Parse response
    // const jsonResponse = JSON.parse(decodedResponse);

    // // Return response
    // return jsonResponse;
  }

  private async sendRelayToProvider(
    chainMessage: ChainMessage,
    relayData: RelayPrivateData,
    unwantedProviders: Set<string>
  ): Promise<RelayResult | Array<RelayError> | Error> {
    if (chainMessage.getApi().getCategory()?.getSubscription() == true) {
      return new Error("subscription currently not supported");
    }
    const chainID = this.rpcEndpoint.chainId;
    const lavaChainId = this.lavaChainId;

    let extraRelayTimeout = 0;
    if (chainMessage.getApi().getCategory()?.getHangingApi() == true) {
      const { averageBlockTime } = this.chainParser.chainBlockStats();
      extraRelayTimeout = averageBlockTime;
    }
    const relayTimeout =
      extraRelayTimeout +
      getTimePerCu(chainMessage.getApi().getComputeUnits()) +
      AverageWorldLatency;
    try {
      const consumerSessionsMap = this.consumerSessionManager.getSessions(
        chainMessage.getApi().getComputeUnits(),
        unwantedProviders,
        chainMessage.getRequestedBlock(),
        "",
        []
      );
      if (consumerSessionsMap instanceof Error) {
        return consumerSessionsMap;
      }
      // TODO: send to several
      // return this.sendRelayToAllProvidersAndRace(
      //   consumerSessionsMap,
      //   extraRelayTimeout
      // );
      const firstEntry = consumerSessionsMap.entries().next();
      if (firstEntry.done) {
        return new Error("returned empty consumerSessionsMap");
      }
      const [providerAddress, sessionInfo] = firstEntry.value;
      const relayResult: RelayResult = {
        providerAddress: providerAddress,
        request: undefined,
        reply: undefined,
        finalized: false,
      };

      const singleConsumerSession = sessionInfo.session;
      const epoch = sessionInfo.epoch;
      const reportedProviders = sessionInfo.reportedProviders;

      const relayRequest = constructRelayRequest(
        lavaChainId,
        chainID,
        relayData,
        providerAddress,
        singleConsumerSession,
        epoch,
        reportedProviders
      );

      relayResult.request = relayRequest;
      return await this.sendRelayProviderInSession(
        sessionInfo,
        extraRelayTimeout,
        chainMessage,
        relayData
      );
    } catch (err) {
      if (err instanceof Error) {
        return err;
      }
      return new Error("unsupported error " + err);
    }
  }
  protected async sendRelayProviderInSession(
    sessionInfo: SessionInfo,
    relayTimeout: number,
    chainMessage: ChainMessage,
    relayData: RelayPrivateData
  ): Promise<RelayResult | Array<RelayError> | Error> {
    return {
      request: undefined,
      reply: undefined,
      providerAddress: "",
      finalized: false,
    };
  }

  // use this as an initial scaffold to send to several providers
  // protected async sendRelayToAllProvidersAndRace(
  //   consumerSessionsMap: ConsumerSessionsMap,
  //   timeoutMs: number
  // ): Promise<any> {
  //   let lastError;
  //   const allRelays: Map<string, Promise<any>> = new Map();
  //   function addTimeoutToPromise(
  //     promise: Promise<any>,
  //     timeoutMs: number
  //   ): Promise<any> {
  //     return Promise.race([
  //       promise,
  //       new Promise((_, reject) =>
  //         setTimeout(() => reject(new Error("Timeout")), timeoutMs)
  //       ),
  //     ]);
  //   }
  //   for (const [providerAddress, sessionInfo] of consumerSessionsMap) {
  //     const providerRelayPromise = this.relayer.sendRelay(
  //       provider.options,
  //       sessionInfo.session
  //     );
  //     allRelays.set(
  //       providerAddress,
  //       addTimeoutToPromise(providerRelayPromise, timeoutMs)
  //     );
  //   }

  //   while (allRelays.size > 0) {
  //     const returnedResponse = await Promise.race([...allRelays.values()]);
  //     if (returnedResponse) { // maybe change this if the promise returns an Error and not throws it
  //       console.log("Ended sending to all providers and race");
  //       return returnedResponse;
  //     }
  //     // Handle removal of completed promises separately (Optional and based on your needs)
  //     allRelays.forEach((promise, key) => {
  //       promise
  //         .then(() => allRelays.delete(key))
  //         .catch(() => allRelays.delete(key));
  //     });
  //   }

  //   throw new Error(
  //     "Failed all promises SendRelayToAllProvidersAndRace: " + String(lastError)
  //   );
  // }
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
