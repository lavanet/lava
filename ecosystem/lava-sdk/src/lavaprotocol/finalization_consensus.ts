import { BaseChainParser } from "../chainlib/base_chain_parser";
import {
  RelayReply,
  RelaySession,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import { FinalizationConflict } from "../grpc_web_services/lavanet/lava/conflict/conflict_data_pb";
import { Logger } from "../logger/logger";

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

  public updateFinalizedHashes(
    blockDistanceForFinalizedData: number,
    providerAddress: string,
    finalizedBlocks: Map<number, string>,
    req: RelaySession,
    reply: RelayReply
  ): FinalizationConflict | undefined {
    const latestBlock = reply.getLatestBlock();
    let finalizationConflict: FinalizationConflict | undefined;
    if (
      this.currentProviderHashesConsensus.length == 0 &&
      this.prevEpochProviderHashesConsensus.length == 0
    ) {
      const newHashConsensus = this.newProviderHashesConsensus(
        blockDistanceForFinalizedData,
        providerAddress,
        latestBlock,
        finalizedBlocks,
        reply,
        req
      );
      this.currentProviderHashesConsensus.push(newHashConsensus);
    } else {
      let inserted = false;
      for (const consensus of this.currentProviderHashesConsensus) {
        const err = this.discrepancyChecker(finalizedBlocks, consensus);
        if (err) {
          finalizationConflict = new FinalizationConflict();
          // TODO: Implement conflict logic from finalization_consensus
          // finalizationConflict.setRelayreply0(reply);
          continue;
        }

        if (!inserted) {
          this.insertProviderToConsensus(
            blockDistanceForFinalizedData,
            consensus,
            finalizedBlocks,
            latestBlock,
            reply,
            req,
            providerAddress
          );
          inserted = true;
        }
      }
      if (!inserted) {
        const newHashConsensus = this.newProviderHashesConsensus(
          blockDistanceForFinalizedData,
          providerAddress,
          latestBlock,
          finalizedBlocks,
          reply,
          req
        );
        this.currentProviderHashesConsensus.push(newHashConsensus);
      }
      if (finalizationConflict) {
        Logger.info("Simulation: Conflict found in discrepancyChecker");
        return finalizationConflict;
      }

      for (const idx in this.prevEpochProviderHashesConsensus) {
        const err = this.discrepancyChecker(
          finalizedBlocks,
          this.prevEpochProviderHashesConsensus[idx]
        );
        if (err) {
          finalizationConflict = new FinalizationConflict();
          // TODO: Implement conflict logic from finalization_consensus
          // finalizationConflict.setRelayreply0(reply);
          Logger.info(
            "Simulation: prev epoch Conflict found in discrepancyChecker",
            "Consensus idx",
            idx,
            "provider",
            providerAddress
          );
          return finalizationConflict;
        }
      }
    }
    return undefined;
  }

  private discrepancyChecker(
    finalizedBlocksA: Map<number, string>,
    consensus: ProviderHashesConsensus
  ): Error | undefined {
    let toIterate: Map<number, string> = new Map();
    let otherBlocks: Map<number, string> = new Map();

    if (finalizedBlocksA.size < consensus.finalizedBlocksHashes.size) {
      toIterate = finalizedBlocksA;
      otherBlocks = consensus.finalizedBlocksHashes;
    } else {
      toIterate = consensus.finalizedBlocksHashes;
      otherBlocks = finalizedBlocksA;
    }

    for (const [blockNum, blockHash] of toIterate.entries()) {
      const otherHash = otherBlocks.get(blockNum);
      if (otherHash) {
        if (blockHash != otherHash) {
          return Logger.fatal(
            "Simulation: reliability discrepancy, different hashes detected for block"
          );
        }
      }
    }
    return undefined;
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

  public getExpectedBlockHeight(
    chainParser: BaseChainParser
  ): ExpectedBlockHeight {
    const chainBlockStats = chainParser.chainBlockStats();
    let highestBlockNumber = 0;
    const findAndUpdateHighestBlockNumber = (
      listProviderHashesConsensus: ProviderHashesConsensus[]
    ) => {
      for (const providerHashesConsensus of listProviderHashesConsensus) {
        for (const [
          ,
          providerDataContainer,
        ] of providerHashesConsensus.agreeingProviders) {
          if (highestBlockNumber < providerDataContainer.latestFinalizedBlock) {
            highestBlockNumber = providerDataContainer.latestFinalizedBlock;
          }
        }
      }
    };

    findAndUpdateHighestBlockNumber(this.prevEpochProviderHashesConsensus);
    findAndUpdateHighestBlockNumber(this.currentProviderHashesConsensus);
    const now = performance.now();
    const calcExpectedBlocks = function (
      mapExpectedBlockHeights: Map<string, number>,
      listProviderHashesConsensus: ProviderHashesConsensus[]
    ) {
      for (const providerHashesConsensus of listProviderHashesConsensus) {
        for (const [
          providerAddress,
          providerDataContiner,
        ] of providerHashesConsensus.agreeingProviders) {
          const interpolation = interpolateBlocks(
            now,
            providerDataContiner.latestBlockTime,
            chainBlockStats.averageBlockTime
          );
          let expected =
            providerDataContiner.latestFinalizedBlock + interpolation;
          if (expected > highestBlockNumber) {
            expected = highestBlockNumber;
          }
          mapExpectedBlockHeights.set(providerAddress, expected);
        }
      }
    };

    const mapExpectedBlockHeights: Map<string, number> = new Map();
    calcExpectedBlocks(
      mapExpectedBlockHeights,
      this.prevEpochProviderHashesConsensus
    );
    calcExpectedBlocks(
      mapExpectedBlockHeights,
      this.currentProviderHashesConsensus
    );

    const median = function (dataMap: Map<string, number>): number {
      const data: Array<number> = [];
      for (const latestBlock of dataMap.values()) {
        data.push(latestBlock);
      }
      data.sort((a, b) => a - b);
      let medianResult = 0;
      if (data.length == 0) {
        return 0;
      } else if (data.length % 2 == 0) {
        medianResult =
          (data[data.length / 2 - 1] + data[data.length / 2]) / 2.0;
      } else {
        medianResult = data[(data.length - 1) / 2];
      }
      return medianResult;
    };
    const medianOfExpectedBlocks = median(mapExpectedBlockHeights);
    const providersMedianOfLatestBlock =
      medianOfExpectedBlocks + chainBlockStats.blockDistanceForFinalizedData;
    if (
      medianOfExpectedBlocks > 0 &&
      providersMedianOfLatestBlock > this.latestBlock
    ) {
      if (
        providersMedianOfLatestBlock > this.latestBlock + 1000 &&
        this.latestBlock > 0
      ) {
        Logger.error(
          "Uncontinuous jump in finalization data, latestBlock",
          this.latestBlock,
          "providersMedianOfLatestBlock",
          providersMedianOfLatestBlock
        );
      }
      this.latestBlock = providersMedianOfLatestBlock;
    }
    return {
      expectedBlockHeight:
        providersMedianOfLatestBlock -
        chainBlockStats.allowedBlockLagForQosSync,
      numOfProviders: mapExpectedBlockHeights.size,
    };
  }
}

function interpolateBlocks(
  timeNow: number,
  latestBlockTime: number,
  averageBlockTime: number
): number {
  if (timeNow < latestBlockTime) {
    return 0;
  }
  return Math.floor((timeNow - latestBlockTime) / averageBlockTime);
}

interface ExpectedBlockHeight {
  expectedBlockHeight: number;
  numOfProviders: number;
}
