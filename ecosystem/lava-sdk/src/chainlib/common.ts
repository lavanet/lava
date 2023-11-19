import { JsonRpcChainParser } from "./jsonrpc";
import { RestChainParser } from "./rest";
import { TendermintRpcChainParser } from "./tendermint";
import {
  APIInterfaceJsonRPC,
  APIInterfaceRest,
  APIInterfaceTendermintRPC,
} from "./base_chain_parser";
import { Logger } from "../logger/logger";
import { EARLIEST_BLOCK } from "../common/common";

export function getChainParser(apiInterface: string) {
  switch (apiInterface) {
    case APIInterfaceJsonRPC:
      return new JsonRpcChainParser();
    case APIInterfaceRest:
      return new RestChainParser();
    case APIInterfaceTendermintRPC:
      return new TendermintRpcChainParser();
    default:
      throw Logger.fatal(
        "Couldn't find api interface in getChainParser options",
        apiInterface
      );
  }
}

// split two requested blocks to the most advanced and most behind
// the hierarchy is as follows:
// NOT_APPLICABLE
// LATEST_BLOCK
// PENDING_BLOCK
// SAFE
// FINALIZED
// numeric value (descending)
// EARLIEST
//
// returns: [latest, earliest]
export function compareRequestedBlockInBatch(
  firstRequestedBlock: number,
  second: number
): [number, number] {
  if (firstRequestedBlock === EARLIEST_BLOCK) {
    return [second, firstRequestedBlock];
  }
  if (second === EARLIEST_BLOCK) {
    return [firstRequestedBlock, second];
  }

  const biggerFirst = (x: number, y: number): [number, number] => [
    Math.max(x, y),
    Math.min(x, y),
  ];

  if (firstRequestedBlock < 0) {
    if (second < 0) {
      // both are negative
      return biggerFirst(firstRequestedBlock, second);
    }
    // first is negative non earliest, second is positive
    return [firstRequestedBlock, second];
  }
  if (second < 0) {
    // second is negative non earliest, first is positive
    return [second, firstRequestedBlock];
  }
  // both are positive
  return biggerFirst(firstRequestedBlock, second);
}
