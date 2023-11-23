import { EARLIEST_BLOCK } from "../common/common";
import { compareRequestedBlockInBatch } from "./common";

describe("compareRequestedBlockInBatch", () => {
  it("should return the correct result when first is EARLIEST_BLOCK", () => {
    const result = compareRequestedBlockInBatch(EARLIEST_BLOCK, 20);
    expect(result).toEqual([20, EARLIEST_BLOCK]);
  });

  it("should return the correct result when second is EARLIEST_BLOCK", () => {
    const result = compareRequestedBlockInBatch(20, EARLIEST_BLOCK);
    expect(result).toEqual([20, EARLIEST_BLOCK]);
  });

  it("should return the correct result when both first and second are EARLIEST_BLOCK", () => {
    const result = compareRequestedBlockInBatch(EARLIEST_BLOCK, EARLIEST_BLOCK);
    expect(result).toEqual([EARLIEST_BLOCK, EARLIEST_BLOCK]);
  });

  it("should return the correct result when both first and second are positive numbers", () => {
    const result = compareRequestedBlockInBatch(10, 20);
    expect(result).toEqual([20, 10]);
  });

  it("should return the correct result when both first and second are EARLIEST_BLOCK", () => {
    const result = compareRequestedBlockInBatch(EARLIEST_BLOCK, EARLIEST_BLOCK);
    expect(result).toEqual([EARLIEST_BLOCK, EARLIEST_BLOCK]);
  });

  it("should return the correct result when both first and second are negative numbers", () => {
    const result = compareRequestedBlockInBatch(-10, -20);
    expect(result).toEqual([-10, -20]);
  });

  it("should return the correct result when first is a positive number and second is a negative number", () => {
    const result = compareRequestedBlockInBatch(10, -20);
    expect(result).toEqual([-20, 10]);
  });

  it("should return the correct result when first is a negative number and second is a positive number", () => {
    const result = compareRequestedBlockInBatch(-10, 20);
    expect(result).toEqual([-10, 20]);
  });
});
