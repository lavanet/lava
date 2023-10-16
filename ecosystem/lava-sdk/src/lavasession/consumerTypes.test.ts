import BigNumber from "bignumber.js";
import {
  calculateAvailabilityScore,
  ConsumerSessionsWithProvider,
  QoSReport,
  SingleConsumerSession,
} from "./consumerTypes";
import { AVAILABILITY_PERCENTAGE, DEFAULT_DECIMAL_PRECISION } from "./common";
import { AlreadyLockedError, NotLockedError } from "./errors";

describe("consumerTypes", () => {
  describe("SingleConsumerSession", () => {
    it("tests locking", () => {
      const session = new SingleConsumerSession(
        42,
        new ConsumerSessionsWithProvider("", [], {}, 0, 0),
        {
          networkAddress: "",
          enabled: true,
          extensions: new Set(),
          addons: new Set(),
          connectionRefusals: 0,
        }
      );

      let lockError = session.tryLock();
      expect(lockError).toBeUndefined();

      lockError = session.tryLock();
      expect(lockError).toBeInstanceOf(AlreadyLockedError);
    });

    it("tests unlocking", () => {
      const session = new SingleConsumerSession(
        42,
        new ConsumerSessionsWithProvider("", [], {}, 0, 0),
        {
          networkAddress: "",
          enabled: true,
          extensions: new Set(),
          addons: new Set(),
          connectionRefusals: 0,
        }
      );

      session.tryLock();

      let unlockError = session.tryUnlock();
      expect(unlockError).toBeUndefined();

      unlockError = session.tryUnlock();
      expect(unlockError).toBeInstanceOf(NotLockedError);
    });
  });

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
    expect(result.scaledAvailabilityScore).toEqual(halfDec.toFixed());
  });
});
