import { newRouterKey } from "./routerKey";

describe("routerKey", () => {
  it("has no repetitions", () => {
    const unsorted = ["a", "a", "b", "b", "c", "c"];
    const routerKey = newRouterKey(unsorted);
    expect(routerKey).toEqual("a|b|c");
  });

  it("sorts correctly", () => {
    const unsorted = ["c", "a", "b"];
    const routerKey = newRouterKey(unsorted);
    expect(routerKey).toEqual("a|b|c");
  });
});
