import {
  cumulativeProbabilityFunctionForPoissonDist,
  perturbWithNormalGaussian,
  ProviderOptimizer,
  Strategy,
} from "./providerOptimizer";
import random from "random";

const TEST_AVERAGE_BLOCK_TIME = 10 * 1000; // 10 seconds in milliseconds
const TEST_BASE_WORLD_LATENCY = 150; // in milliseconds

describe("ProviderOptimizer", () => {
  it("tests provider optimizer basic", async () => {
    const providerOptimizer = setupProviderOptimizer();
    const providers = setupProvidersForTest(10);

    const requestCU = 10;
    const requestBlock = 1000;
    const perturbationPercentage = 0.0;

    let returnedProviders = providerOptimizer.chooseProvider(
      providers,
      [],
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);

    providerOptimizer.appendProbeRelayData(
      providers[1],
      TEST_BASE_WORLD_LATENCY * 3,
      true
    );

    await sleep(4);

    returnedProviders = providerOptimizer.chooseProvider(
      providers,
      [],
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).not.toBe(providers[1]);

    providerOptimizer.appendProbeRelayData(
      providers[0],
      TEST_BASE_WORLD_LATENCY / 2,
      true
    );

    await sleep(4);

    returnedProviders = providerOptimizer.chooseProvider(
      providers,
      [],
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).toBe(providers[0]);
  });

  it("tests provider optimizer basic relay data", () => {
    const providerOptimizer = setupProviderOptimizer();
    const providers = setupProvidersForTest(10);

    const requestCU = 1;
    const requestBlock = 1000;
    const perturbationPercentage = 0.0;
    const syncBlock = requestBlock;

    providerOptimizer.appendRelayData(
      providers[1],
      TEST_BASE_WORLD_LATENCY * 4,
      false,
      requestCU,
      syncBlock
    );
    let returnedProviders = providerOptimizer.chooseProvider(
      providers,
      [],
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).not.toBe(providers[1]);

    providerOptimizer.appendRelayData(
      providers[0],
      TEST_BASE_WORLD_LATENCY / 4,
      false,
      requestCU,
      syncBlock
    );
    returnedProviders = providerOptimizer.chooseProvider(
      providers,
      [],
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).toBe(providers[0]);
  });

  it("tests provider optimizer availability", () => {
    const providerOptimizer = setupProviderOptimizer();
    const providersCount = 100;
    const providers = setupProvidersForTest(providersCount);

    const requestCU = 10;
    const requestBlock = 1000;
    const perturbationPercentage = 0.0;
    const skipIndex = random.int(0, providersCount);

    for (let i = 0; i < providersCount; i++) {
      if (i === skipIndex) {
        continue;
      }

      providerOptimizer.appendProbeRelayData(
        providers[i],
        TEST_BASE_WORLD_LATENCY,
        false
      );
    }

    let returnedProviders = providerOptimizer.chooseProvider(
      providers,
      [],
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).toBe(providers[skipIndex]);

    returnedProviders = providerOptimizer.chooseProvider(
      providers,
      [providers[skipIndex]],
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).not.toBe(providers[skipIndex]);
  });

  describe("perturbWithNormalGaussian", () => {
    it("tests perturbation", () => {
      const originalValue1 = 1.0;
      const originalValue2 = 0.5;
      const perturbationPercentage = 0.03;

      const runs = 100000;
      let success = 0;
      for (let i = 0; i < runs; i++) {
        const res1 = perturbWithNormalGaussian(
          originalValue1,
          perturbationPercentage
        );
        const res2 = perturbWithNormalGaussian(
          originalValue2,
          perturbationPercentage
        );

        if (res1 > res2) {
          success++;
        }
      }

      expect(success).toBeGreaterThanOrEqual(runs * 0.9);
    });
  });

  describe("cumulativeProbabilityFunctionForPoissonDist", () => {
    it("tests probabilities", () => {
      const value1 = cumulativeProbabilityFunctionForPoissonDist(1, 10);
      const value2 = cumulativeProbabilityFunctionForPoissonDist(10, 10);

      expect(value2).toBeGreaterThan(value1);
    });
  });
});

function setupProviderOptimizer() {
  return new ProviderOptimizer(
    Strategy.Balanced,
    TEST_AVERAGE_BLOCK_TIME,
    TEST_BASE_WORLD_LATENCY,
    1
  );
}

function setupProvidersForTest(count: number): string[] {
  const providers: string[] = [];
  for (let i = 0; i < count; i++) {
    providers.push(`lava@test_${i}`);
  }
  return providers;
}

async function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
