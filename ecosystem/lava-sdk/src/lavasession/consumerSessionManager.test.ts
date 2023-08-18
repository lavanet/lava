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
const RELAY_NUMBER_AFTER_FIRST_CALL = 1;
const LATEST_RELAY_CU_AFTER_DONE = 0;

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
        [],
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
        cm.onSessionDone(
          consumerSession.session,
          SERVICED_BLOCK_NUMBER,
          CU_FOR_FIRST_REQUEST,
          0,
          consumerSession.session.calculateExpectedLatency(2),
          SERVICED_BLOCK_NUMBER - 1,
          NUMBER_OF_PROVIDERS,
          NUMBER_OF_PROVIDERS,
          false
        );
        expect(consumerSession.session.cuSum).toEqual(CU_FOR_FIRST_REQUEST);
        expect(consumerSession.session.latestRelayCu).toEqual(
          LATEST_RELAY_CU_AFTER_DONE
        );
        expect(consumerSession.session.relayNum).toEqual(
          RELAY_NUMBER_AFTER_FIRST_CALL
        );
        expect(consumerSession.session.latestBlock).toEqual(
          SERVICED_BLOCK_NUMBER
        );
      }
    });

    it("tests pairing reset", () => {
      const cm = new ConsumerSessionManager(
        new RPCEndpoint("stub", "stub", "stub", 0),
        new RandomProviderOptimizer()
      );
      const pairingList = createPairingList("", true);
      cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);
      cm.validAddresses = [];

      const consumerSessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        [],
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
        cm.onSessionDone(
          consumerSession.session,
          SERVICED_BLOCK_NUMBER,
          CU_FOR_FIRST_REQUEST,
          0,
          consumerSession.session.calculateExpectedLatency(2),
          SERVICED_BLOCK_NUMBER - 1,
          NUMBER_OF_PROVIDERS,
          NUMBER_OF_PROVIDERS,
          false
        );
        expect(consumerSession.session.cuSum).toEqual(CU_FOR_FIRST_REQUEST);
        expect(consumerSession.session.latestRelayCu).toEqual(
          LATEST_RELAY_CU_AFTER_DONE
        );
        expect(consumerSession.session.relayNum).toEqual(
          RELAY_NUMBER_AFTER_FIRST_CALL
        );
        expect(consumerSession.session.latestBlock).toEqual(
          SERVICED_BLOCK_NUMBER
        );
        expect(cm.getNumberOfResets()).toEqual(1);
      }
    });

    it("test pairing reset with failures", () => {
      const cm = new ConsumerSessionManager(
        new RPCEndpoint("stub", "stub", "stub", 0),
        new RandomProviderOptimizer()
      );
      const pairingList = createPairingList("", true);
      cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      while (true) {
        if (cm.validAddresses.length === 0) {
          break;
        }

        const consumerSessions = cm.getSessions(
          CU_FOR_FIRST_REQUEST,
          [],
          SERVICED_BLOCK_NUMBER,
          "",
          []
        );

        for (const consumerSession of Object.values(consumerSessions)) {
          cm.onSessionFailure(consumerSession.session);
        }
      }

      const consumerSessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        [],
        SERVICED_BLOCK_NUMBER,
        "",
        []
      );
      expect(cm.validAddresses.length).toEqual(cm.getPairingAddressesLength());

      for (const consumerSession of Object.values(consumerSessions)) {
        expect(consumerSession.epoch).toEqual(cm.getCurrentEpoch());
        expect(consumerSession.session.latestRelayCu).toEqual(
          CU_FOR_FIRST_REQUEST
        );
        expect(cm.getNumberOfResets()).toEqual(1);
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

      expect(cm.validAddresses.length).toEqual(NUMBER_OF_PROVIDERS);
      expect(cm.getPairingAddressesLength()).toEqual(NUMBER_OF_PROVIDERS);
      expect(cm.getCurrentEpoch()).toEqual(FIRST_EPOCH_HEIGHT);
      for (let i = 0; i < NUMBER_OF_PROVIDERS; i++) {
        expect(cm.validAddresses[i]).toEqual(`provider${i}`);
      }
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
