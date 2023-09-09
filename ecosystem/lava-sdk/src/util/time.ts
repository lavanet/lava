export const hourInMillis = 60 * 60 * 1000;

export function now(): number {
  return performance.timeOrigin + performance.now();
}

export function millisToSeconds(millis: number): number {
  return millis / 1000;
}
