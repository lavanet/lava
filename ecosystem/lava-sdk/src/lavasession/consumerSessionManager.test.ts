import { ConsumerSessionManager } from "./consumerSessionManager";
import { RandomProviderOptimizer } from "./providerOptimizer";
import {
  ConsumerSessionsWithProvider,
  Endpoint,
  RPCEndpoint,
} from "./consumerTypes";

const NUMBER_OF_PROVIDERS = 10;
const FIRST_EPOCH_HEIGHT = 20;
const CU_FOR_FIRST_REQUEST = 10;
const SERVICED_BLOCK_NUMBER = 30;

describe("ConsumerSessionManager", () => {
  describe("getSessions", () => {
    it("happy flow", () => {
      const cm = new ConsumerSessionManager(
        new RPCEndpoint("stub", "stub", "stub", 0),
        new RandomProviderOptimizer()
      );
      const pairingList = createPairingList("", true);
      cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      const consumerSessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        {},
        SERVICED_BLOCK_NUMBER,
        "",
        []
      );
      expect(Object.keys(consumerSessions).length).toBeGreaterThan(0);

      for (const consumerSession of Object.values(consumerSessions)) {
        expect(consumerSession.epoch).toEqual(cm.getCurrentEpoch());
        expect(consumerSession.session.latestRelayCu).toEqual(
          CU_FOR_FIRST_REQUEST
        );
      }
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
