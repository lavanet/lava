import { millisToSeconds } from "../time";

export class ScoreStore {
  public constructor(
    public readonly num: number,
    public readonly denom: number,
    public readonly time: number
  ) {}

  /**
   * Calculates the time decayed score update between two `ScoreStore` entries.
   * It uses a decay function with a half life of `halfLife` to factor in the time
   * elapsed since the `oldScore` was recorded. Both the numerator and the denominator
   * are decayed by this function.
   *
   * Additionally, the `newScore` is factored by a weight of `updateWeight`.
   * The function returns a new `ScoreStore` entry with the updated numerator, denominator, and current time.
   *
   * The mathematical equation used to calculate the update is:
   *
   * updatedNum   = oldScore.Num * exp(-(sampleTime - oldScore.time) / halfLife) + newScore.Num * (-(sampleTime - newScore.time) / halfLife) * updateWeight
   * updatedDenom = oldScore.Denom * exp(-(sampleTime - oldScore.time) / halfLife) + newScore.Denom * (-(sampleTime - newScore.time) / halfLife) * updateWeight
   *
   * Note that the returned `ScoreStore` has a new time field set to the sample time.
   */
  public static calculateTimeDecayFunctionUpdate(
    oldScore: ScoreStore,
    newScore: ScoreStore,
    halfLife: number,
    updateWeight: number,
    sampleTime: number
  ): ScoreStore {
    const oldDecayExponent =
      (Math.LN2 * millisToSeconds(sampleTime - oldScore.time)) /
      millisToSeconds(halfLife);
    const oldDecayFactor = Math.exp(-oldDecayExponent);
    const newDecayExponent =
      (Math.LN2 * millisToSeconds(sampleTime - newScore.time)) /
      millisToSeconds(halfLife);
    const newDecayFactor = Math.exp(-newDecayExponent);
    const updatedNum =
      oldScore.num * oldDecayFactor +
      newScore.num * newDecayFactor * updateWeight;
    const updatedDenom =
      oldScore.denom * oldDecayFactor +
      newScore.denom * newDecayFactor * updateWeight;
    return new ScoreStore(updatedNum, updatedDenom, sampleTime);
  }
}
