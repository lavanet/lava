import {
  COST_EXPLORATION_CHANCE,
  cumulativeProbabilityFunctionForPoissonDist,
  floatToBigNumber,
  perturbWithNormalGaussian,
  ProviderData,
  ProviderOptimizer,
  ProviderOptimizerStrategy,
  FLOAT_PRECISION,
} from "./providerOptimizer";
import random from "random";
import { now } from "../util/time";
import seedrandom from "seedrandom";

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
      new Set(providers),
      new Set(),
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
      new Set(providers),
      new Set(),
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
      new Set(providers),
      new Set(),
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
      new Set(providers),
      new Set(),
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
      new Set(providers),
      new Set(),
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

    // append at least one successful probe, otherwise the providers get outright ignored
    for (let i = 0; i < providersCount; i++) {
      if (i === skipIndex) {
        continue;
      }

      providerOptimizer.appendProbeRelayData(
        providers[i],
        TEST_BASE_WORLD_LATENCY,
        true
      );
    }

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
      new Set(providers),
      new Set(),
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    console.log(
      "[Debugging] expect(returnedProviders[0]).toBe(providers[skipIndex]); Optimizer Issue",
      "returnedProviders",
      returnedProviders,
      "providers",
      providers,
      "skipIndex",
      skipIndex
    );
    expect(returnedProviders[0]).toBe(providers[skipIndex]);

    returnedProviders = providerOptimizer.chooseProvider(
      new Set(providers),
      new Set([providers[skipIndex]]),
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).not.toBe(providers[skipIndex]);
  });

  it("tests provider optimizer availability relay data", () => {
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

      providerOptimizer.appendRelayFailure(providers[i]);
    }

    let returnedProviders = providerOptimizer.chooseProvider(
      new Set(providers),
      new Set(),
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).toBe(providers[skipIndex]);

    returnedProviders = providerOptimizer.chooseProvider(
      new Set(providers),
      new Set([providers[skipIndex]]),
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).not.toBe(providers[skipIndex]);
  });

  // it("tests provider optimizer availability block error", async () => {
  //   const providerOptimizer = setupProviderOptimizer();
  //   const providersCount = 100;
  //   const providers = setupProvidersForTest(providersCount);

  //   const requestCU = 10;
  //   const requestBlock = 1000;
  //   const perturbationPercentage = 0.0;
  //   const syncBlock = requestBlock;
  //   const chosenIndex = random.int(0, providersCount);

  //   for (let i = 0; i < providersCount; i++) {
  //     await sleep(4);

  //     if (i === chosenIndex) {
  //       providerOptimizer.appendRelayData(
  //         providers[i],
  //         TEST_BASE_WORLD_LATENCY + 10,
  //         false,
  //         requestCU,
  //         syncBlock
  //       );
  //       continue;
  //     }

  //     providerOptimizer.appendRelayData(
  //       providers[i],
  //       TEST_BASE_WORLD_LATENCY,
  //       false,
  //       requestCU,
  //       syncBlock - 1
  //     );
  //   }

  //   let returnedProviders = providerOptimizer.chooseProvider(
  //     new Set(providers),
  //     new Set(),
  //     requestCU,
  //     requestBlock,
  //     perturbationPercentage
  //   );
  //   expect(returnedProviders).toHaveLength(1);
  //   expect(returnedProviders[0]).toBe(providers[chosenIndex]);

  //   returnedProviders = providerOptimizer.chooseProvider(
  //     new Set(providers),
  //     new Set([providers[chosenIndex]]),
  //     requestCU,
  //     requestBlock,
  //     perturbationPercentage
  //   );
  //   expect(returnedProviders).toHaveLength(1);
  //   expect(returnedProviders[0]).not.toBe(providers[chosenIndex]);
  // });

  // this test fails statistically. we need to solve the issue also on the golang version.
  // it("tests provider optimizer updating latency", async () => {
  //   const providerOptimizer = setupProviderOptimizer();
  //   const providersCount = 2;
  //   const providers = setupProvidersForTest(providersCount);

  //   const requestCU = 10;
  //   const requestBlock = 1000;
  //   const syncBlock = requestBlock;

  //   const getProviderData = (providerAddress: string) => {
  //     // @ts-expect-error private method but we need it for testing without exposing it
  //     return providerOptimizer.getProviderData(providerAddress);
  //   };
  //   const calculateLatencyScore = (
  //     providerData: ProviderData,
  //     requestCU: number,
  //     requestBlock: number
  //   ) => {
  //     // @ts-expect-error private method but we need it for testing without exposing it
  //     return providerOptimizer.calculateLatencyScore(
  //       providerData,
  //       requestCU,
  //       requestBlock
  //     );
  //   };

  //   let providerAddress = providers[0];
  //   for (let i = 0; i < 10; i++) {
  //     let { providerData } = getProviderData(providerAddress);

  //     const currentLatencyScore = calculateLatencyScore(
  //       providerData,
  //       requestCU,
  //       requestBlock
  //     );
  //     providerOptimizer.appendProbeRelayData(
  //       providerAddress,
  //       TEST_BASE_WORLD_LATENCY,
  //       true
  //     );

  //     await sleep(4);

  //     const data = getProviderData(providerAddress);
  //     providerData = data.providerData;
  //     expect(data.found).toBe(true);

  //     const newLatencyScore = calculateLatencyScore(
  //       providerData,
  //       requestCU,
  //       requestBlock
  //     );
  //     expect(currentLatencyScore).toBeGreaterThan(newLatencyScore);
  //   }

  //   providerAddress = providers[1];
  //   for (let i = 0; i < 10; i++) {
  //     let { providerData } = getProviderData(providerAddress);
  //     const currentLatencyScore = calculateLatencyScore(
  //       providerData,
  //       requestCU,
  //       requestBlock
  //     );
  //     providerOptimizer.appendRelayData(
  //       providerAddress,
  //       TEST_BASE_WORLD_LATENCY,
  //       false,
  //       requestCU,
  //       syncBlock
  //     );

  //     await sleep(4);

  //     const data = getProviderData(providerAddress);
  //     providerData = data.providerData;
  //     expect(data.found).toBe(true);

  //     const newLatencyScore = calculateLatencyScore(
  //       providerData,
  //       requestCU,
  //       requestBlock
  //     );
  //     expect(currentLatencyScore).toBeGreaterThan(newLatencyScore);
  //   }
  // });

  it("tests provider optimizer strategies provider count", async () => {
    const providerOptimizer = setupProviderOptimizer();
    const providersCount = 5;
    const providers = setupProvidersForTest(providersCount);

    const requestCU = 10;
    const requestBlock = 1000;
    const perturbationPercentage = 0.0;
    const syncBlock = requestBlock;

    for (let i = 0; i < 10; i++) {
      for (const address of providers) {
        providerOptimizer.appendRelayData(
          address,
          TEST_BASE_WORLD_LATENCY * 2,
          false,
          requestCU,
          syncBlock
        );
      }

      await sleep(4);
    }

    const testProvidersCount = (iterations: number) => {
      let exploration = 0;
      for (let i = 0; i < iterations; i++) {
        const returnedProviders = providerOptimizer.chooseProvider(
          new Set(providers),
          new Set(),
          requestCU,
          requestBlock,
          perturbationPercentage
        );

        if (returnedProviders.length > 1) {
          exploration++;
        }
      }

      return exploration;
    };

    const setProviderOptimizerStrategy = (
      strategy: ProviderOptimizerStrategy
    ) => {
      // @ts-expect-error private property but we need it for testing without exposing it
      providerOptimizer.strategy = strategy;
    };
    const setProviderOptimizerConcurrencyProviders = (concurrency: number) => {
      // @ts-expect-error private property but we need it for testing without exposing it
      providerOptimizer.wantedNumProvidersInConcurrency = concurrency;
    };

    // with a cost strategy we expect only one provider, two with a change of 1/100
    setProviderOptimizerStrategy(ProviderOptimizerStrategy.Cost);
    setProviderOptimizerConcurrencyProviders(2);
    const iterations = 10000;
    let exploration = testProvidersCount(iterations);
    expect(exploration).toBeLessThan(
      1.3 * iterations * providersCount * COST_EXPLORATION_CHANCE
    );

    setProviderOptimizerStrategy(ProviderOptimizerStrategy.Balanced);
    exploration = testProvidersCount(iterations);
    expect(exploration).toBeLessThan(1.3 * iterations * providersCount * 0.5);

    setProviderOptimizerStrategy(ProviderOptimizerStrategy.Privacy);
    exploration = testProvidersCount(iterations);
    expect(exploration).toBe(0);
  });

  it("tests provider optimizer sync score", async () => {
    const providerOptimizer = setupProviderOptimizer();
    const providers = setupProvidersForTest(10);

    const requestCU = 10;
    const requestBlock = -2; // from golang spectypes.LATEST_BLOCK (-2)
    const perturbationPercentage = 0.0;
    const syncBlock = 1000;

    const chosenIndex = random.int(0, providers.length - 1);
    let sampleTime = now();
    const appendRelayData = (
      providerAddress: string,
      latency: number,
      syncBlock: number,
      sampleTime: number
    ) => {
      // @ts-expect-error private method but we need it for testing without exposing it
      providerOptimizer.appendRelay(
        providerAddress,
        latency,
        false,
        true,
        requestCU,
        syncBlock,
        sampleTime
      );
    };
    for (let j = 0; j < 3; j++) {
      for (let i = 0; i < providers.length; i++) {
        await sleep(4);

        if (i === chosenIndex) {
          appendRelayData(
            providers[i],
            TEST_BASE_WORLD_LATENCY * 2 + 1 / 1000, // 1 microsecond
            syncBlock + 5,
            sampleTime
          );
          continue;
        }

        appendRelayData(
          providers[i],
          TEST_BASE_WORLD_LATENCY * 2,
          syncBlock,
          sampleTime
        );
      }

      sampleTime += 5;
    }

    await sleep(4);
    let returnedProviders = providerOptimizer.chooseProvider(
      new Set(providers),
      new Set(),
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).toBe(providers[chosenIndex]);

    returnedProviders = providerOptimizer.chooseProvider(
      new Set(providers),
      new Set(),
      requestCU,
      syncBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).not.toBe(providers[chosenIndex]);
  });

  it("tests provider optimizer strategies scoring", async () => {
    const providerOptimizer = setupProviderOptimizer();
    const providersCount = 5;
    const providers = setupProvidersForTest(providersCount);

    const requestCU = 10;
    const requestBlock = -2; // from golang spectypes.LATEST_BLOCK (-2)
    const syncBlock = 1000;
    const perturbationPercentage = 0.0;

    for (let i = 0; i < 10; i++) {
      for (const address of providers) {
        providerOptimizer.appendRelayData(
          address,
          TEST_BASE_WORLD_LATENCY * 2,
          false,
          requestCU,
          syncBlock
        );
      }

      await sleep(4);
    }

    await sleep(4);

    for (let i = 0; i < providers.length; i++) {
      const address = providers[i];
      if (i !== 2) {
        providerOptimizer.appendProbeRelayData(
          address,
          TEST_BASE_WORLD_LATENCY * 2,
          false
        );
        await sleep(4);
      }

      providerOptimizer.appendProbeRelayData(
        address,
        TEST_BASE_WORLD_LATENCY * 2,
        true
      );
      await sleep(4);
      providerOptimizer.appendProbeRelayData(
        address,
        TEST_BASE_WORLD_LATENCY * 2,
        false
      );
      await sleep(4);
      providerOptimizer.appendProbeRelayData(
        address,
        TEST_BASE_WORLD_LATENCY * 2,
        true
      );
      await sleep(4);
      providerOptimizer.appendProbeRelayData(
        address,
        TEST_BASE_WORLD_LATENCY * 2,
        true
      );
      await sleep(4);
    }

    let sampleTime = now();
    const improvedLatency = 280; // milliseconds
    const normalLatency = TEST_BASE_WORLD_LATENCY * 2;
    const improvedBlock = syncBlock + 1;

    const appendRelayData = (
      providerAddress: string,
      latency: number,
      syncBlock: number,
      sampleTime: number
    ) => {
      // @ts-expect-error private method but we need it for testing without exposing it
      providerOptimizer.appendRelay(
        providerAddress,
        latency,
        false,
        true,
        requestCU,
        syncBlock,
        sampleTime
      );
    };

    // provider 0 gets a good latency
    appendRelayData(providers[0], improvedLatency, syncBlock, sampleTime);

    // providers 3, 4 get a regular entry
    appendRelayData(providers[3], normalLatency, syncBlock, sampleTime);
    appendRelayData(providers[4], normalLatency, syncBlock, sampleTime);

    // provider 1 gets a good sync
    appendRelayData(providers[1], normalLatency, improvedBlock, sampleTime);

    sampleTime = sampleTime + 10;
    appendRelayData(providers[0], improvedLatency, syncBlock, sampleTime);
    appendRelayData(providers[3], normalLatency, syncBlock, sampleTime);
    appendRelayData(providers[4], normalLatency, syncBlock, sampleTime);
    appendRelayData(providers[1], normalLatency, improvedBlock, sampleTime);

    const setProviderStrategy = (strategy: ProviderOptimizerStrategy) => {
      // @ts-expect-error private property but we need it for testing without exposing it
      providerOptimizer.strategy = strategy;
    };

    await sleep(4);

    setProviderStrategy(ProviderOptimizerStrategy.Balanced);
    // a balanced strategy should pick provider 2 because of it's high availability
    let returnedProviders = providerOptimizer.chooseProvider(
      new Set(providers),
      new Set(),
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).toBe(providers[2]);

    setProviderStrategy(ProviderOptimizerStrategy.Cost);
    // with a cost strategy we expect the same as balanced
    returnedProviders = providerOptimizer.chooseProvider(
      new Set(providers),
      new Set(),
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).toBe(providers[2]);

    setProviderStrategy(ProviderOptimizerStrategy.Latency);
    // latency strategy should pick the best latency
    returnedProviders = providerOptimizer.chooseProvider(
      new Set(providers),
      new Set([providers[2]]),
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).toBe(providers[0]);

    setProviderStrategy(ProviderOptimizerStrategy.SyncFreshness);
    // with a cost strategy we expect the same as balanced
    returnedProviders = providerOptimizer.chooseProvider(
      new Set(providers),
      new Set([providers[2]]),
      requestCU,
      requestBlock,
      perturbationPercentage
    );
    expect(returnedProviders).toHaveLength(1);
    expect(returnedProviders[0]).toBe(providers[1]);
  });

  it("tests provider optimizer perturbation", async () => {
    const providerOptimizer = setupProviderOptimizer();
    const providersCount = 100;
    const providers = setupProvidersForTest(providersCount);

    const requestCU = 10;
    const requestBlock = -2; // from golang spectypes.LATEST_BLOCK (-2)
    const syncBlock = 1000;
    const perturbationPercentage = 0.03; // this is statistical and we do not want this failing

    let sampleTime = now();

    const appendRelayData = (
      providerAddress: string,
      latency: number,
      syncBlock: number,
      sampleTime: number
    ) => {
      // @ts-expect-error private method but we need it for testing without exposing it
      providerOptimizer.appendRelay(
        providerAddress,
        latency,
        false,
        true,
        requestCU,
        syncBlock,
        sampleTime
      );
    };

    for (let i = 0; i < 10; i++) {
      providers.forEach((address, idx) => {
        if (idx < providers.length / 2) {
          appendRelayData(
            address,
            TEST_BASE_WORLD_LATENCY,
            syncBlock,
            sampleTime
          );
        } else {
          appendRelayData(
            address,
            TEST_BASE_WORLD_LATENCY * 10,
            syncBlock,
            sampleTime
          );
        }
      });

      sampleTime = sampleTime + 5;
    }

    const getProviderData = (providerAddress: string) => {
      // @ts-expect-error private method but we need it for testing without exposing it
      return providerOptimizer.getProviderData(providerAddress);
    };

    for (const address of providers) {
      const { found } = getProviderData(address);
      expect(found).toBe(true);
    }

    const seed = String(now());
    console.log("random seed", seed);
    // @ts-expect-error random has wring typings, but docs use examples with seedrandom
    random.use(seedrandom(seed));
    let same = 0;
    let pickFaults = 0;
    const chosenProvider = providerOptimizer.chooseProvider(
      new Set(providers),
      new Set(),
      requestCU,
      requestBlock,
      0
    )[0];
    const runs = 1000;

    for (let i = 0; i < runs; i++) {
      const returnedProviders = providerOptimizer.chooseProvider(
        new Set(providers),
        new Set(),
        requestCU,
        requestBlock,
        perturbationPercentage
      );

      expect(returnedProviders).toHaveLength(1);
      if (returnedProviders[0] === chosenProvider) {
        same++;
      }

      for (const [idx, address] of providers.entries()) {
        if (address === returnedProviders[0] && idx >= providers.length / 2) {
          pickFaults++;
          break;
        }
      }
    }

    expect(pickFaults).toBeLessThan(runs * 0.01);
    expect(same).toBeLessThan(runs / 2);
  });

  it("tests excellence report", async () => {
    const floatVal = 0.25;
    const floatNew = floatToBigNumber(floatVal, FLOAT_PRECISION);
    expect(floatNew.toNumber()).toEqual(floatVal);

    const providerOptimizer = setupProviderOptimizer();
    const providersCount = 5;
    const providers = setupProvidersForTest(providersCount);

    const requestCU = 10;
    const syncBlock = 1000;

    const appendRelayData = (
      providerAddress: string,
      latency: number,
      syncBlock: number,
      sampleTime: number
    ) => {
      // @ts-expect-error private method but we need it for testing without exposing it
      providerOptimizer.appendRelay(
        providerAddress,
        latency,
        false,
        true,
        requestCU,
        syncBlock,
        sampleTime
      );
    };

    let sampleTime = now();

    for (let i = 0; i < 10; i++) {
      for (const address of providers) {
        appendRelayData(
          address,
          TEST_BASE_WORLD_LATENCY * 2,
          syncBlock,
          sampleTime
        );
      }
      await sleep(4);
      sampleTime += 4;
    }
    const report = providerOptimizer.getExcellenceQoSReportForProvider(
      providers[0]
    );
    expect(report).not.toBeUndefined();
    const report2 = providerOptimizer.getExcellenceQoSReportForProvider(
      providers[1]
    );
    expect(report2).not.toBeUndefined();
    expect(report2).toEqual(report);
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

function setupProviderOptimizer(
  wantedProviders = 1,
  strategy: ProviderOptimizerStrategy = ProviderOptimizerStrategy.Balanced
) {
  return new ProviderOptimizer(
    strategy,
    TEST_AVERAGE_BLOCK_TIME,
    TEST_BASE_WORLD_LATENCY,
    wantedProviders
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
