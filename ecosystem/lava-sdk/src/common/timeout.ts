import { BaseChainParser } from "../chainlib/base_chain_parser";
import { BaseChainMessageContainer } from "../chainlib/chain_message";
import {
  IsHangingApi,
  GetComputeUnits,
} from "../chainlib/chain_message_queries";

export const TimePerCU = 100; // ms
export const MinimumTimePerRelayDelay = 1000; // ms
export const AverageWorldLatency = 300; // ms

export function getTimePerCu(computeUnits: number): number {
  if (localNodeTimePerCu(computeUnits) < MinimumTimePerRelayDelay) {
    return MinimumTimePerRelayDelay;
  }
  return localNodeTimePerCu(computeUnits);
}

function localNodeTimePerCu(computeUnits: number): number {
  return baseTimePerCU(computeUnits) + AverageWorldLatency;
}

export function baseTimePerCU(computeUnits: number): number {
  return computeUnits * TimePerCU;
}

export function GetRelayTimeout(
  chainMessage: BaseChainMessageContainer,
  chainParser: BaseChainParser,
  timeouts: number
): number {
  let extraRelayTimeout = 0;
  if (IsHangingApi(chainMessage)) {
    const chainStats = chainParser.chainBlockStats();
    extraRelayTimeout = chainStats.averageBlockTime;
  }
  let relayTimeAddition = getTimePerCu(GetComputeUnits(chainMessage));
  if (chainMessage.getApi().getTimeoutMs() > 0) {
    relayTimeAddition = chainMessage.getApi().getTimeoutMs();
  }
  // Set relay timeout, increase it every time we fail a relay on timeout
  return (
    extraRelayTimeout + (timeouts + 1) * relayTimeAddition + AverageWorldLatency
  );
}
