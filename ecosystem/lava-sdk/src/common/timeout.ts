export const TimePerCU = 100; // ms
export const MinimumTimePerRelayDelay = 1000; // ms
export const AverageWorldLatency = 300; // ms

export function getTimePerCu(computeUnits: number): number {
  return localNodeTimePerCu(computeUnits) + MinimumTimePerRelayDelay;
}

function localNodeTimePerCu(computeUnits: number): number {
  return baseTimePerCU(computeUnits) + AverageWorldLatency;
}

export function baseTimePerCU(computeUnits: number): number {
  return computeUnits * TimePerCU;
}
