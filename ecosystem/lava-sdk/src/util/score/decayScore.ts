import { millisToSeconds } from "../time";

export class ScoreStore {
  public constructor(
    public readonly num: number,
    public readonly denom: number,
    public readonly time: number
  ) {}

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
