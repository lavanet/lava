import { Logger } from "../logger/logger";
import { Relayer } from "../relayer/relayer";
import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";
import { SingleConsumerSession } from "../lavasession/consumerTypes";
import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  ChainMessage,
} from "../chainlib/base_chain_parser";
import {
  constructRelayRequest,
  IsFinalizedBlock,
  newRelayData,
  UpdateRequestedBlock,
  verifyFinalizationData,
  verifyRelayReply,
} from "../lavaprotocol/request_builder";
import { RPCEndpoint } from "../lavasession/consumerTypes";
import {
  RelayPrivateData,
  RelayReply,
  RelayRequest,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import SDKErrors from "../sdk/errors";
import { AverageWorldLatency, getTimePerCu } from "../common/timeout";
import { FinalizationConsensus } from "../lavaprotocol/finalization_consensus";
import { BACKOFF_TIME_ON_FAILURE, LATEST_BLOCK } from "../common/common";

const MaxRelayRetries = 4;

export class RPCConsumerServer {
  private consumerSessionManager: ConsumerSessionManager;
  private chainParser: BaseChainParser;
  private geolocation: string;
  private relayer: Relayer;
  private rpcEndpoint: RPCEndpoint;
  private lavaChainId: string;
  private consumerAddress: string;
  private finalizationConsensus: FinalizationConsensus;
  constructor(
    relayer: Relayer,
    consumerSessionManager: ConsumerSessionManager,
    chainParser: BaseChainParser,
    geolocation: string,
    rpcEndpoint: RPCEndpoint,
    lavaChainId: string,
    finalizationConsensus: FinalizationConsensus
  ) {
    this.consumerSessionManager = consumerSessionManager;
    this.geolocation = geolocation;
    this.chainParser = chainParser;
    this.relayer = relayer;
    this.rpcEndpoint = rpcEndpoint;
    this.lavaChainId = lavaChainId;
    this.consumerAddress = "TODO"; // TODO: this needs to be the public address that the provider signs finalization data with, check on badges if it's the signer or the badge project key
    this.finalizationConsensus = finalizationConsensus;
  }

  public setChainParser(chainParser: BaseChainParser) {
    this.chainParser = chainParser;
  }

  public supportedChainAndApiInterface(): SupportedChainAndApiInterface {
    return {
      specId: this.rpcEndpoint.chainId,
      apiInterface: this.rpcEndpoint.apiInterface,
    };
  }

  async sendRelay(options: SendRelayOptions | SendRestRelayOptions) {
    const chainMessage = this.chainParser.parseMsg(options);
    const unwantedProviders = new Set<string>();
    const relayData = {
      ...chainMessage.getRawRequestData(), // url and data fields
      connectionType:
        chainMessage.getApiCollection().getCollectionData()?.getType() ?? "",
      apiInterface: this.rpcEndpoint.apiInterface,
      chainId: this.rpcEndpoint.chainId,
      requestedBlock: chainMessage.getRequestedBlock(),
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
        // relayResult can be an Array of errors from relaying to multiple providers
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
          Logger.warn("relay succeeded but had some errors", ...errors);
        }
        return relayResult;
      }
    }
    // got here if didn't succeed in any of the relays
    throw new Error("failed all retries " + errors.join(","));
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
    const isHangingapi =
      chainMessage.getApi().getCategory()?.getHangingApi() == true;
    if (isHangingapi) {
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
        LATEST_BLOCK,
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
      const [providerPublicAddress, sessionInfo] = firstEntry.value;
      const relayResult: RelayResult = {
        providerAddress: providerPublicAddress,
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
        providerPublicAddress,
        singleConsumerSession,
        epoch,
        reportedProviders
      );
      relayResult.request = relayRequest;
      const relayResponse = await this.relayInner(
        singleConsumerSession,
        relayResult,
        relayTimeout
      );
      if (relayResponse.err != undefined) {
        const callSessionFailure = () => {
          const err = this.consumerSessionManager.onSessionFailure(
            singleConsumerSession,
            relayResponse.err
          );
          if (err instanceof Error) {
            Logger.error("failed on session failure %s", err);
          }
        };
        if (relayResponse.backoff) {
          const backOffDuration = BACKOFF_TIME_ON_FAILURE;
          setTimeout(callSessionFailure, backOffDuration); // call sessionFailure after a delay
        } else {
          callSessionFailure();
        }
        const relayError: RelayError = {
          providerAddress: providerPublicAddress,
          err: relayResponse.err,
        };
        return [relayError];
      }
      const reply = relayResult.reply;
      if (reply == undefined) {
        return new Error("reply is undefined");
      }

      // we got here if everything is valid
      const { expectedBlockHeight, numOfProviders } =
        this.finalizationConsensus.getExpectedBlockHeight(this.chainParser);
      const pairingAddressesLen =
        this.consumerSessionManager.getPairingAddressesLength();
      const latestBlock = reply.getLatestBlock();
      this.consumerSessionManager.onSessionDone(
        singleConsumerSession,
        latestBlock,
        chainMessage.getApi().getComputeUnits(),
        relayResponse.latency,
        singleConsumerSession.calculateExpectedLatency(relayTimeout),
        expectedBlockHeight,
        numOfProviders,
        pairingAddressesLen,
        isHangingapi
      );
      return relayResult;
    } catch (err) {
      if (err instanceof Error) {
        return err;
      }
      return new Error("unsupported error " + err);
    }
  }

  protected async relayInner(
    singleConsumerSession: SingleConsumerSession,
    relayResult: RelayResult,
    relayTimeout: number
  ): Promise<RelayResponse> {
    const relayRequest = relayResult.request;
    const response: RelayResponse = {
      relayResult: undefined,
      latency: 0,
      backoff: false,
      err: undefined,
    };
    if (relayRequest == undefined) {
      response.err = new Error("relayRequest is empty");
      return response;
    }
    const relaySession = relayRequest.getRelaySession();
    if (relaySession == undefined) {
      response.err = new Error("empty relay session");
      return response;
    }
    const relayData = relayRequest.getRelayData();
    if (relayData == undefined) {
      response.err = new Error("empty relay data");
      return response;
    }
    const providerPublicAddress = relayResult.providerAddress;
    const relayResponse = await this.sendRelayProviderInSession(
      singleConsumerSession,
      relayResult,
      relayTimeout
    );
    if (relayResponse.err != undefined) {
      return relayResponse;
    }
    if (relayResponse.relayResult == undefined) {
      relayResponse.err = new Error("empty relayResult");
      return relayResponse;
    }
    relayResult = relayResponse.relayResult;
    const reply = relayResult.reply;
    if (reply == undefined) {
      relayResponse.err = new Error("empty reply");
      return relayResponse;
    }
    const chainBlockStats = this.chainParser.chainBlockStats();
    UpdateRequestedBlock(relayData, reply);
    const finalized = IsFinalizedBlock(
      relayData.getRequestBlock(),
      reply.getLatestBlock(),
      chainBlockStats.blockDistanceForFinalizedData
    );
    relayResult.finalized = finalized;
    // TODO: when we add headers
    // filteredHeaders, _, ignoredHeaders := rpccs.chainParser.HandleHeaders(reply.Metadata, chainMessage.GetApiCollection(), spectypes.Header_pass_reply)

    const err = verifyRelayReply(reply, relayRequest, providerPublicAddress);
    if (err instanceof Error) {
      relayResponse.err = err;
      return relayResponse;
    }
    const existingSessionLatestBlock = singleConsumerSession.latestBlock;
    const dataReliabilityParams = this.chainParser.dataReliabilityParams();
    if (dataReliabilityParams.enabled) {
      const finalizationData = verifyFinalizationData(
        reply,
        relayRequest,
        providerPublicAddress,
        this.consumerAddress,
        existingSessionLatestBlock,
        chainBlockStats.blockDistanceForFinalizedData
      );
      if (finalizationData instanceof Error) {
        relayResponse.err = finalizationData;
        return relayResponse;
      }
      if (finalizationData.finalizationConflict != undefined) {
        // TODO: send a self finalization conflict
        relayResponse.err = new Error("invalid finalization data");
        return relayResponse;
      }

      const finalizationConflict =
        this.finalizationConsensus.updateFinalizedHashes(
          chainBlockStats.blockDistanceForFinalizedData,
          providerPublicAddress,
          finalizationData.finalizedBlocks,
          relaySession,
          reply
        );
      if (finalizationConflict != undefined) {
        // TODO: send a consensus finalization conflict
        relayResponse.err = new Error("conflicting finalization data");
        return relayResponse;
      }
    }
    return relayResponse;
  }

  protected async sendRelayProviderInSession(
    singleConsumerSession: SingleConsumerSession,
    relayResult: RelayResult,
    relayTimeout: number
  ): Promise<RelayResponse> {
    const endpointClient = singleConsumerSession.endpoint.client;
    const response: RelayResponse = {
      relayResult: undefined,
      latency: 0,
      backoff: false,
      err: undefined,
    };
    if (endpointClient == undefined) {
      response.err = new Error("endpointClient is undefined");
      return response;
    }
    const relayRequest = relayResult.request;
    if (relayRequest == undefined) {
      response.err = new Error("relayRequest is undefined");
      return response;
    }
    const startTime = performance.now();
    try {
      const relayReply = await this.relayer.sendRelay(
        endpointClient,
        relayRequest,
        relayTimeout
      );
      if (relayReply instanceof Error) {
        throw relayReply;
      }
      relayResult.reply = relayReply;
      const measuredLatency = performance.now() - startTime;
      const relayResponse: RelayResponse = {
        backoff: false,
        latency: measuredLatency,
        err: undefined,
        relayResult: relayResult,
      };
      return relayResponse;
    } catch (err) {
      let backoff = false;
      let castedError = new Error(
        "caught unexpected error while sending relay"
      );
      if (err instanceof Error) {
        if (err == SDKErrors.relayTimeout) {
          // timed out so we need a backoff
          backoff = true;
        }
        castedError = err;
      }
      const measuredLatency = performance.now() - startTime;
      const relayResponse: RelayResponse = {
        backoff: backoff,
        latency: measuredLatency,
        err: castedError,
        relayResult: undefined,
      };
      return relayResponse;
    }
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

export interface RelayResponse {
  relayResult: RelayResult | undefined;
  latency: number;
  backoff: boolean;
  err: Error | undefined;
}

export interface SupportedChainAndApiInterface {
  specId: string;
  apiInterface: string;
}
