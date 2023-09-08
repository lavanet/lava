import { BaseChainParser } from "../chainlib/base_chain_parser";
import {
  RelayRequest,
  RelayReply,
  RelaySession,
  RelayPrivateData,
  Badge,
  ProbeRequest,
  ProbeReply,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb";

interface ProviderDataContainer {
  latestFinalizedBlock: number;
  latestBlockTime: number;
  finalizedBlockHashes: Map<number, string>;
  sigBlocks: string | Uint8Array;
  sessionId: number;
  blockHeight: number;
  relayNum: number;
  latestBlock: number;
}

interface ProviderHashesConsensus {
  finalizedBlocksHashes: Map<number, string>;
  agreeingProviders: Map<string, ProviderDataContainer>;
}

export function GetLatestFinalizedBlock(
  latestBlock: number,
  blockDistanceForFinalizedData: number
) {
  return latestBlock - blockDistanceForFinalizedData;
}

export class FinalizationConsensus {
  private currentProviderHashesConsensus: Array<ProviderHashesConsensus>;
  private prevEpochProviderHashesConsensus: Array<ProviderHashesConsensus>;
  private currentEpoch: number;
  private latestBlock: number;
  constructor() {
    this.currentProviderHashesConsensus = [];
    this.prevEpochProviderHashesConsensus = [];
    this.currentEpoch = 0;
    this.latestBlock = 0;
  }

  private newProviderHashesConsensus(
    blockDistanceForFinalizedData: number,
    providerAcc: string,
    latestBlock: number,
    finalizedBlocks: Map<number, string>,
    reply: RelayReply,
    req: RelaySession
  ): ProviderHashesConsensus {
    const newProviderDataContainer: ProviderDataContainer = {
      latestFinalizedBlock: GetLatestFinalizedBlock(
        latestBlock,
        blockDistanceForFinalizedData
      ),
      latestBlockTime: performance.now(),
      finalizedBlockHashes: finalizedBlocks,
      sigBlocks: reply.getSigBlocks(),
      sessionId: req.getSessionId(),
      relayNum: req.getRelayNum(),
      blockHeight: req.getEpoch(),
      latestBlock: latestBlock,
    };
    const providerDataContiners: Map<string, ProviderDataContainer> = new Map([
      [providerAcc, newProviderDataContainer],
    ]);
    return {
      finalizedBlocksHashes: finalizedBlocks,
      agreeingProviders: providerDataContiners,
    };
  }

  private insertProviderToConsensus(
    blockDistanceForFinalizedData: number,
    consensus: ProviderHashesConsensus,
    finalizedBlocks: Map<number, string>,
    latestBlock: number,
    reply: RelayReply,
    req: RelaySession,
    providerAcc: string
  ) {
    const newProviderDataContainer: ProviderDataContainer = {
      latestFinalizedBlock: GetLatestFinalizedBlock(
        latestBlock,
        blockDistanceForFinalizedData
      ),
      latestBlockTime: performance.now(),
      finalizedBlockHashes: finalizedBlocks,
      sigBlocks: reply.getSigBlocks(),
      sessionId: req.getSessionId(),
      relayNum: req.getRelayNum(),
      blockHeight: req.getEpoch(),
      latestBlock: latestBlock,
    };
    consensus.agreeingProviders.set(providerAcc, newProviderDataContainer);
    for (const [blockNum, blockHash] of finalizedBlocks.entries()) {
      consensus.finalizedBlocksHashes.set(blockNum, blockHash);
    }
  }

  public updateFinalizedHashes() {
    // TODO: implement for DR.
  }

  public newEpoch(epoch: number) {
    if (this.currentEpoch < epoch) {
      // means it's time to refresh the epoch
      this.prevEpochProviderHashesConsensus =
        this.currentProviderHashesConsensus;
      this.currentProviderHashesConsensus = [];
      this.currentEpoch = epoch;
    }
  }

  public getLatestBlock(): number {
    return this.latestBlock;
  }

  public getExpectedBlockHeight(chainParser: BaseChainParser) {}
}

interface ExpectedBlockHeight {
  expectedBlockHeight: number;
  numOfProviders: number;
}
