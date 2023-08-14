import { ConsumerSessionManager } from "./consumerSessionManager";
import { RandomProviderOptimizer } from "./providerOptimizer";
import {
  ConsumerSessionsWithProvider,
  Endpoint,
  RPCEndpoint,
} from "./consumerTypes";

const NUMBER_OF_PROVIDERS = 10;
const FIRST_EPOCH_HEIGHT = 20;

describe("ConsumerSessionManager", () => {
  describe("getSessions", () => {
    it("will return an array with one entry", () => {
      const cm = new ConsumerSessionManager(
        new RPCEndpoint("stub", "stub", "stub", 0),
        new RandomProviderOptimizer()
      );

      const sessions = cm.getSessions(0, {}, 0, "");
      expect(Object.keys(sessions).length).toEqual(1);
    });
  });

  describe("updateAllProviders", () => {
    it("updates providers", () => {
      const cm = new ConsumerSessionManager(
        new RPCEndpoint("stub", "stub", "stub", 0),
        new RandomProviderOptimizer()
      );

      const pairingList = createPairingList("", true);
      cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      expect(cm.getPairingAddressesLength()).toEqual(NUMBER_OF_PROVIDERS);
    });
  });
});

function createPairingList(
  providerPrefixAddress: string,
  enabled: boolean
): Map<number, ConsumerSessionsWithProvider> {
  const sessionsWithProvider = new Map<number, ConsumerSessionsWithProvider>();
  const pairingEndpoints: Endpoint[] = [
    {
      networkAddress: "",
      extensions: new Set(),
      addons: new Set(),
      connectionRefusals: 0,
      enabled,
    },
  ];
  const pairingEndpointsWithAddon: Endpoint[] = [
    {
      networkAddress: "",
      extensions: new Set(),
      addons: new Set(["addon"]),
      connectionRefusals: 0,
      enabled,
    },
  ];
  const pairingEndpointsWithExtension: Endpoint[] = [
    {
      networkAddress: "",
      extensions: new Set(["ext1"]),
      addons: new Set(["addon"]),
      connectionRefusals: 0,
      enabled,
    },
  ];
  const pairingEndpointsWithExtensions: Endpoint[] = [
    {
      networkAddress: "",
      extensions: new Set(["ext1", "ext2"]),
      addons: new Set(["addon"]),
      connectionRefusals: 0,
      enabled,
    },
  ];

  for (let i = 0; i < NUMBER_OF_PROVIDERS; i++) {
    let endpoints: Endpoint[];

    switch (i) {
      case 0:
      case 1:
        endpoints = pairingEndpointsWithAddon;
        break;
      case 2:
        endpoints = pairingEndpointsWithExtension;
        break;
      case 3:
        endpoints = pairingEndpointsWithExtensions;
        break;
      default:
        endpoints = pairingEndpoints;
    }

    sessionsWithProvider.set(
      i,
      new ConsumerSessionsWithProvider(
        "provider" + providerPrefixAddress + i,
        endpoints,
        {},
        200,
        FIRST_EPOCH_HEIGHT
      )
    );
  }

  return sessionsWithProvider;
}
