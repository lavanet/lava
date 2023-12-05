import { Logger } from "../logger/logger";
import { Relayer } from "../relayer/relayer";
import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";
import {
  RPCEndpoint,
  SingleConsumerSession,
} from "../lavasession/consumerTypes";
import { ConsumerConsistency } from "./consumerConsistency";
import {
  BaseChainParser,
  SendRelayOptions,
  SendRelaysBatchOptions,
  SendRestRelayOptions,
} from "../chainlib/base_chain_parser";
import {
  // GetAddon,
  IsSubscription,
  IsHangingApi,
  GetComputeUnits,
  GetStateful,
} from "../chainlib/chain_message_queries";
import {
  constructRelayRequest,
  IsFinalizedBlock,
  newRelayData,
  UpdateRequestedBlock,
  verifyFinalizationData,
  verifyRelayReply,
} from "../lavaprotocol/request_builder";
import {
  RelayPrivateData,
  RelayReply,
  RelayRequest,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import SDKErrors from "../sdk/errors";
import { GetRelayTimeout } from "../common/timeout";
import { FinalizationConsensus } from "../lavaprotocol/finalization_consensus";
import { BACKOFF_TIME_ON_FAILURE, LATEST_BLOCK } from "../common/common";
import { BaseChainMessageContainer } from "../chainlib/chain_message";
import { Header } from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";
import { promiseAny } from "../util/common";
import { EmergencyTrackerInf } from "../stateTracker/updaters/emergency_tracker";

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
  private consumerConsistency: ConsumerConsistency;
  private emergencyTracker: EmergencyTrackerInf | undefined;
  constructor(
    relayer: Relayer,
    consumerSessionManager: ConsumerSessionManager,
    chainParser: BaseChainParser,
    geolocation: string,
    rpcEndpoint: RPCEndpoint,
    lavaChainId: string,
    finalizationConsensus: FinalizationConsensus,
    consumerConsistency: ConsumerConsistency
  ) {
    this.consumerSessionManager = consumerSessionManager;
    this.geolocation = geolocation;
    this.chainParser = chainParser;
    this.relayer = relayer;
    this.rpcEndpoint = rpcEndpoint;
    this.lavaChainId = lavaChainId;
    this.consumerAddress = "TODO"; // TODO: this needs to be the public address that the provider signs finalization data with, check on badges if it's the signer or the badge project key
    this.finalizationConsensus = finalizationConsensus;
    this.consumerConsistency = consumerConsistency;
  }

  // returning getSessionManager for debugging / data reading.
  public getSessionManager(): ConsumerSessionManager {
    return this.consumerSessionManager;
  }

  public setChainParser(chainParser: BaseChainParser) {
    this.chainParser = chainParser;
  }

  public setEmergencyTracker(emergencyTracker: EmergencyTrackerInf) {
    this.emergencyTracker = emergencyTracker;
  }

  public supportedChainAndApiInterface(): SupportedChainAndApiInterface {
    return {
      specId: this.rpcEndpoint.chainId,
      apiInterface: this.rpcEndpoint.apiInterface,
    };
  }

  async sendRelay(
    options: SendRelayOptions | SendRelaysBatchOptions | SendRestRelayOptions
  ) {
    const chainMessage = this.chainParser.parseMsg(options);
    const unwantedProviders = new Set<string>();
    const relayData = {
      ...chainMessage.getRawRequestData(), // url and data fields
      connectionType:
        chainMessage.getApiCollection().getCollectionData()?.getType() ?? "",
      apiInterface: this.rpcEndpoint.apiInterface,
      chainId: this.rpcEndpoint.chainId,
      seenBlock: this.consumerConsistency.getSeenBlock(),
      requestedBlock: chainMessage.getRequestedBlock(),
      headers: chainMessage.getRPCMessage().getHeaders(),
    };
    const relayPrivateData = newRelayData(relayData);
    let blockOnSyncLoss = true;
    const errors = new Array<Error>();
    // we use the larger value between MaxRelayRetries and valid addresses in the case we have blocked a few providers and get to 0 number of providers
    // we want to reset the list and try again otherwise we will be left with no providers at all for the remaining of the epoch.
    const maxRetriesAsSizeOfValidAddressesList = Math.max(
      this.consumerSessionManager.getValidAddresses("", []).size,
      MaxRelayRetries
    );

    let timeouts = 0;
    for (
      let retries = 0;
      retries < maxRetriesAsSizeOfValidAddressesList;
      retries++
    ) {
      const relayResult = await this.sendRelayToProvider(
        chainMessage,
        relayPrivateData,
        unwantedProviders,
        timeouts
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
          if (oneResult.err == SDKErrors.relayTimeout) {
            timeouts++;
          }
          errors.push(oneResult.err);
        }
      } else if (relayResult instanceof Error) {
        errors.push(relayResult);
      } else {
        if (errors.length > 0) {
          Logger.warn("Relay succeeded but had some errors", ...errors);
        }
        const latestBlock = relayResult.reply?.getLatestBlock();
        if (latestBlock) {
          this.consumerConsistency.setSeenBlock(latestBlock);
        }
        return relayResult;
      }
    }
    // got here if didn't succeed in any of the relays
    throw new Error("failed all retries " + errors.join(","));
  }

  private async sendRelayToProvider(
    chainMessage: BaseChainMessageContainer,
    relayData: RelayPrivateData,
    unwantedProviders: Set<string>,
    timeouts: number
  ): Promise<RelayResult | Array<RelayError> | Error> {
    if (IsSubscription(chainMessage)) {
      return new Error("subscription currently not supported");
    }
    const chainID = this.rpcEndpoint.chainId;
    const lavaChainId = this.lavaChainId;
    const relayTimeout = GetRelayTimeout(
      chainMessage,
      this.chainParser,
      timeouts
    );
    let virtualEpoch = 0;
    if (this.emergencyTracker) {
      virtualEpoch = this.emergencyTracker.getVirtualEpoch();
    }
    const consumerSessionsMap = this.consumerSessionManager.getSessions(
      GetComputeUnits(chainMessage),
      unwantedProviders,
      LATEST_BLOCK,
      "",
      [],
      GetStateful(chainMessage),
      virtualEpoch
    );
    if (consumerSessionsMap instanceof Error) {
      return consumerSessionsMap;
    }

    if (consumerSessionsMap.size == 0) {
      return new Error("returned empty consumerSessionsMap");
    }

    let finalRelayResult: RelayResult | RelayError[] | Error | undefined;
    let responsesReceived = 0;

    const trySetFinalRelayResult = (
      res: RelayResult | RelayError[] | Error
    ) => {
      const isError = res instanceof Error || Array.isArray(res);
      if (!isError && finalRelayResult === undefined) {
        finalRelayResult = res;
      }

      if (
        finalRelayResult === undefined &&
        responsesReceived == consumerSessionsMap.size
      ) {
        finalRelayResult = res;
      }
    };

    const promises = [];
    for (const [providerPublicAddress, sessionInfo] of consumerSessionsMap) {
      const relayResult: RelayResult = {
        providerAddress: providerPublicAddress,
        request: undefined,
        reply: undefined,
        finalized: false,
      };

      const singleConsumerSession = sessionInfo.session;
      const epoch = sessionInfo.epoch;
      const reportedProviders = sessionInfo.reportedProviders;
      Logger.debug(
        `Before Construct: ${relayData.getRequestBlock()}, address: ${providerPublicAddress}, session: ${
          singleConsumerSession.sessionId
        }`
      );
      relayResult.request = constructRelayRequest(
        lavaChainId,
        chainID,
        relayData.clone(), // clone here so we can modify the query without affecting retries
        providerPublicAddress,
        singleConsumerSession,
        epoch,
        reportedProviders
      );

      Logger.info(`Sending relay to provider ${providerPublicAddress}`);
      Logger.debug(
        `Relay stats sessionId:${
          singleConsumerSession.sessionId
        }, guid:${relayResult.request
          .getRelayData()
          ?.getSalt_asB64()}, requestedBlock: ${relayResult.request
          .getRelayData()
          ?.getRequestBlock()}, apiInterface:${relayResult.request
          .getRelayData()
          ?.getApiInterface()}, seenBlock: ${relayResult.request
          .getRelayData()
          ?.getSeenBlock()}`
      );
      const promise = this.relayInner(
        singleConsumerSession,
        relayResult,
        chainMessage,
        relayTimeout
      )
        .catch((err: any) => {
          responsesReceived++;
          throw err;
        })
        .then((relayResponse: RelayResponse) => {
          responsesReceived++;

          if (relayResponse.err != undefined) {
            const callSessionFailure = () => {
              const err = this.consumerSessionManager.onSessionFailure(
                singleConsumerSession,
                relayResponse.err
              );
              if (err instanceof Error) {
                Logger.error("Failed on session failure %s", err);
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

            const response = [relayError];
            trySetFinalRelayResult(response);

            return response;
          }
          const reply = relayResult.reply;
          if (reply == undefined) {
            const err = new Error("reply is undefined");

            trySetFinalRelayResult(err);

            return err;
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
            GetComputeUnits(chainMessage),
            relayResponse.latency,
            singleConsumerSession.calculateExpectedLatency(relayTimeout),
            expectedBlockHeight,
            numOfProviders,
            pairingAddressesLen,
            IsHangingApi(chainMessage)
          );

          trySetFinalRelayResult(relayResult);

          return relayResult;
        })
        .catch((err: unknown) => {
          if (err instanceof Error) {
            return err;
          }

          return new Error("unsupported error " + err);
        });

      promises.push(promise);
    }

    const response = await promiseAny(promises);
    if (response instanceof Error) {
      return response;
    }

    // this should never happen, but we need to satisfy the typescript compiler
    if (finalRelayResult === undefined) {
      return new Error("UnreachableCode finalRelayResult is undefined");
    }

    return finalRelayResult;
  }

  protected async relayInner(
    singleConsumerSession: SingleConsumerSession,
    relayResult: RelayResult,
    chainMessage: BaseChainMessageContainer,
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
    Logger.debug("Updating requested Block", singleConsumerSession.sessionId);
    UpdateRequestedBlock(relayData, reply);
    Logger.debug("after Updating", relayData.getRequestBlock());
    Logger.debug("did errored", relayResponse.err);
    const finalized = IsFinalizedBlock(
      relayData.getRequestBlock(),
      reply.getLatestBlock(),
      chainBlockStats.blockDistanceForFinalizedData
    );
    relayResult.finalized = finalized;
    const headersHandler = this.chainParser.handleHeaders(
      reply.getMetadataList(),
      chainMessage.getApiCollection(),
      Header.HeaderType.PASS_REPLY
    );

    reply.setMetadataList(headersHandler.filteredHeaders);

    const err = verifyRelayReply(reply, relayRequest, providerPublicAddress);
    if (err instanceof Error) {
      relayResponse.err = err;
      return relayResponse;
    }

    reply.setMetadataList([
      ...headersHandler.filteredHeaders,
      ...headersHandler.ignoredMetadata,
    ]);

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
