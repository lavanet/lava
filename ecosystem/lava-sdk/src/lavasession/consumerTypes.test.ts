import BigNumber from "bignumber.js";
import { calculateAvailabilityScore, QoSReport } from "./consumerTypes";
import { AVAILABILITY_PERCENTAGE, DEFAULT_DECIMAL_PRECISION } from "./common";

describe("consumerTypes", () => {
  it("should test calculate availability score", () => {
    const precision = 10000;
    let qosReport: QoSReport = {
      latencyScoreList: [],
      totalRelays: precision,
      answeredRelays: precision - AVAILABILITY_PERCENTAGE * precision,
      syncScoreSum: 0,
      totalSyncScore: 0,
    };

    let result = calculateAvailabilityScore(qosReport);
    expect(BigNumber(result.downtimePercentage).toNumber()).toEqual(
      AVAILABILITY_PERCENTAGE
    );
    expect(BigNumber(result.scaledAvailabilityScore).toNumber()).toEqual(0);

    qosReport = {
      latencyScoreList: [],
      totalRelays: 2 * precision,
      answeredRelays: 2 * precision - AVAILABILITY_PERCENTAGE * precision,
      syncScoreSum: 0,
      totalSyncScore: 0,
    };

    const halfDec = BigNumber("0.5");
    result = calculateAvailabilityScore(qosReport);
    expect(BigNumber(result.downtimePercentage).toNumber() * 2).toEqual(
      AVAILABILITY_PERCENTAGE
    );
    expect(result.scaledAvailabilityScore).toEqual(
      halfDec.toPrecision(DEFAULT_DECIMAL_PRECISION)
    );
  });
});
