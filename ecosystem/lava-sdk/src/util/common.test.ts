import { promiseAny } from "./common";

describe("common", () => {
  describe("promiseAny", () => {
    it("should return the first resolved promise", async () => {
      const promises = [
        new Promise((resolve) => setTimeout(() => resolve(1), 100)),
        new Promise((resolve) => setTimeout(() => resolve(2), 200)),
        new Promise((resolve) => setTimeout(() => resolve(3), 300)),
      ];

      const result = await promiseAny(promises);
      expect(result).toEqual(1);
    });

    it("throws an error if all promises are rejected", async () => {
      const promises = [
        new Promise((_, reject) => setTimeout(() => reject(1), 100)),
        new Promise((_, reject) => setTimeout(() => reject(2), 200)),
        new Promise((_, reject) => setTimeout(() => reject(3), 300)),
      ];

      await expect(promiseAny(promises)).rejects.toEqual([1, 2, 3]);
    });
  });
});
