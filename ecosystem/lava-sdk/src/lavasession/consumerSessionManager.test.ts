import {
  ALLOWED_PROBE_RETRIES,
  ConsumerSessionManager,
  TIMEOUT_BETWEEN_PROBES,
} from "./consumerSessionManager";
import {
  ConsumerSessionsWithProvider,
  Endpoint,
  RPCEndpoint,
  SingleConsumerSession,
} from "./consumerTypes";
import { PairingListEmptyError, ReportAndBlockProviderError } from "./errors";
import { Relayer } from "../relayer/relayer";
import { ProbeReply } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import { sleep } from "../util/common";
import {
  ProviderOptimizer,
  ProviderOptimizerStrategy,
} from "../providerOptimizer/providerOptimizer";
import { AverageWorldLatency } from "../common/timeout";
import { CONSISTENCY_SELECT_ALLPROVIDERS, NOSTATE } from "../common/common";

const NUMBER_OF_PROVIDERS = 10;
const NUMBER_OF_RESETS_TO_TEST = 10;
const FIRST_EPOCH_HEIGHT = 20;
const SECOND_EPOCH_HEIGHT = 40;
const CU_FOR_FIRST_REQUEST = 10;
const MAX_CU_FOR_VIRTUAL_EPOCH = 200;
const FIRST_VIRTUAL_EPOCH = 1;
const SERVICED_BLOCK_NUMBER = 30;
const RELAY_NUMBER_AFTER_FIRST_CALL = 1;
const RELAY_NUMBER_AFTER_FIRST_FAIL = 1;
const LATEST_RELAY_CU_AFTER_DONE = 0;
const NUMBER_OF_ALLOWED_SESSIONS_PER_CONSUMER = 10;
const CU_SUM_ON_FAILURE = 0;

function setupConsumerSessionManager(
  relayer?: Relayer,
  optimizer?: ProviderOptimizer
) {
  if (!relayer) {
    relayer = setupRelayer();
    jest
      .spyOn(relayer, "probeProvider")
      .mockImplementation((_providerAddress, _apiInterface, guid) => {
        const response: ProbeReply = new ProbeReply();
        response.setLatestBlock(42);
        response.setLavaEpoch(20);
        response.setGuid(guid);
        return Promise.resolve(response);
      });
  }

  if (!optimizer) {
    optimizer = setupProviderOptimizer(ProviderOptimizerStrategy.Balanced);
  }

  const cm = new ConsumerSessionManager(
    relayer,
    new RPCEndpoint("stub", "stub", "stub", 1),
    optimizer
  );

  return cm;
}

function setupRelayer(): Relayer {
  return new Relayer({
    allowInsecureTransport: true,
    lavaChainId: "lava-local-1",
    privKey: "",
    secure: true,
  });
}

function setupProviderOptimizer(
  strategy: ProviderOptimizerStrategy,
  wantedNumProviders = 1
): ProviderOptimizer {
  return new ProviderOptimizer(
    strategy,
    1,
    AverageWorldLatency / 2,
    wantedNumProviders
  );
}

describe("ConsumerSessionManager", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe("getSessions", () => {
    it("happy flow", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      const consumerSessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        0
      );
      if (consumerSessions instanceof Error) {
        throw consumerSessions;
      }
      expect(consumerSessions.size).toBeGreaterThan(0);

      for (const consumerSession of consumerSessions.values()) {
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

    it("virtual epoch increase maxCu", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      let consumerSessions = cm.getSessions(
        MAX_CU_FOR_VIRTUAL_EPOCH,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        0
      );
      if (consumerSessions instanceof Error) {
        throw consumerSessions;
      }
      expect(consumerSessions.size).toBeGreaterThan(0);

      for (const consumerSession of consumerSessions.values()) {
        expect(consumerSession.epoch).toEqual(cm.getCurrentEpoch());
        expect(consumerSession.session.latestRelayCu).toEqual(
          MAX_CU_FOR_VIRTUAL_EPOCH
        );
        cm.onSessionDone(
          consumerSession.session,
          SERVICED_BLOCK_NUMBER,
          MAX_CU_FOR_VIRTUAL_EPOCH,
          0,
          consumerSession.session.calculateExpectedLatency(2),
          SERVICED_BLOCK_NUMBER - 1,
          NUMBER_OF_PROVIDERS,
          NUMBER_OF_PROVIDERS,
          false
        );
        expect(consumerSession.session.cuSum).toEqual(MAX_CU_FOR_VIRTUAL_EPOCH);
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

      //increase virtual epoch
      consumerSessions = cm.getSessions(
        MAX_CU_FOR_VIRTUAL_EPOCH,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        FIRST_VIRTUAL_EPOCH
      );
      if (consumerSessions instanceof Error) {
        throw consumerSessions;
      }
      expect(consumerSessions.size).toBeGreaterThan(0);

      for (const consumerSession of consumerSessions.values()) {
        expect(consumerSession.epoch).toEqual(cm.getCurrentEpoch());
        expect(consumerSession.session.latestRelayCu).toEqual(
          MAX_CU_FOR_VIRTUAL_EPOCH
        );
        cm.onSessionDone(
          consumerSession.session,
          SERVICED_BLOCK_NUMBER,
          MAX_CU_FOR_VIRTUAL_EPOCH,
          0,
          consumerSession.session.calculateExpectedLatency(2),
          SERVICED_BLOCK_NUMBER - 1,
          NUMBER_OF_PROVIDERS,
          NUMBER_OF_PROVIDERS,
          false
        );
        expect(consumerSession.session.cuSum).toEqual(
          MAX_CU_FOR_VIRTUAL_EPOCH * (FIRST_VIRTUAL_EPOCH + 1)
        );
        expect(consumerSession.session.latestRelayCu).toEqual(
          LATEST_RELAY_CU_AFTER_DONE
        );
        expect(consumerSession.session.relayNum).toEqual(
          RELAY_NUMBER_AFTER_FIRST_CALL + 1
        );
        expect(consumerSession.session.latestBlock).toEqual(
          SERVICED_BLOCK_NUMBER
        );
      }
    });

    it("virtual epoch exceeding maxCu with failure", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      let consumerSessions = cm.getSessions(
        MAX_CU_FOR_VIRTUAL_EPOCH,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        0
      );
      if (consumerSessions instanceof Error) {
        throw consumerSessions;
      }
      expect(consumerSessions.size).toBeGreaterThan(0);

      for (const consumerSession of consumerSessions.values()) {
        expect(consumerSession.epoch).toEqual(cm.getCurrentEpoch());
        expect(consumerSession.session.latestRelayCu).toEqual(
          MAX_CU_FOR_VIRTUAL_EPOCH
        );
        cm.onSessionDone(
          consumerSession.session,
          SERVICED_BLOCK_NUMBER,
          MAX_CU_FOR_VIRTUAL_EPOCH,
          0,
          consumerSession.session.calculateExpectedLatency(2),
          SERVICED_BLOCK_NUMBER - 1,
          NUMBER_OF_PROVIDERS,
          NUMBER_OF_PROVIDERS,
          false
        );
        expect(consumerSession.session.cuSum).toEqual(MAX_CU_FOR_VIRTUAL_EPOCH);
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

      //increase virtual epoch with more cu than allowed
      consumerSessions = cm.getSessions(
        MAX_CU_FOR_VIRTUAL_EPOCH * (FIRST_VIRTUAL_EPOCH + 1) + 10,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        FIRST_VIRTUAL_EPOCH
      );
      expect(consumerSessions).toBeInstanceOf(Error);
    });

    it("tests pairing reset", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);
      cm.validAddresses = new Set();

      const consumerSessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        0
      );
      if (consumerSessions instanceof Error) {
        throw consumerSessions;
      }
      expect(consumerSessions.size).toBeGreaterThan(0);

      for (const consumerSession of consumerSessions.values()) {
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

    it("test pairing reset with failures", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      while (true) {
        if (cm.validAddresses.size === 0) {
          break;
        }

        const consumerSessions = cm.getSessions(
          CU_FOR_FIRST_REQUEST,
          new Set(),
          SERVICED_BLOCK_NUMBER,
          "",
          [],
          NOSTATE,
          0
        );
        if (consumerSessions instanceof Error) {
          throw consumerSessions;
        }

        for (const consumerSession of consumerSessions.values()) {
          cm.onSessionFailure(consumerSession.session);
        }
      }

      const consumerSessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        0
      );
      if (consumerSessions instanceof Error) {
        throw consumerSessions;
      }

      expect(cm.validAddresses.size).toEqual(cm.getPairingAddressesLength());

      for (const consumerSession of consumerSessions.values()) {
        expect(consumerSession.epoch).toEqual(cm.getCurrentEpoch());
        expect(consumerSession.session.latestRelayCu).toEqual(
          CU_FOR_FIRST_REQUEST
        );
        expect(cm.getNumberOfResets()).toEqual(1);
      }
    });

    it("tests pairing reset with multiple failures", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      // let numberOfResets = 0;
      for (
        let numberOfResets = 0;
        numberOfResets < NUMBER_OF_RESETS_TO_TEST;
        numberOfResets++
      ) {
        while (true) {
          if (cm.validAddresses.size === 0) {
            break;
          }

          const consumerSessions = cm.getSessions(
            CU_FOR_FIRST_REQUEST,
            new Set(),
            SERVICED_BLOCK_NUMBER,
            "",
            [],
            NOSTATE,
            0
          );

          if (consumerSessions instanceof Map) {
            for (const consumerSession of consumerSessions.values()) {
              const error = cm.onSessionFailure(consumerSession.session);
              if (error) {
                throw error;
              }
            }
          }

          if (
            cm.validAddresses.size === 0 &&
            consumerSessions instanceof PairingListEmptyError
          ) {
            break;
          }
        }

        expect(cm.validAddresses.size).toEqual(0);

        const consumerSessions = cm.getSessions(
          CU_FOR_FIRST_REQUEST,
          new Set(),
          SERVICED_BLOCK_NUMBER,
          "",
          [],
          NOSTATE,
          0
        );
        if (consumerSessions instanceof Error) {
          throw consumerSessions;
        }

        expect(cm.validAddresses.size).toEqual(cm.getPairingAddressesLength());

        for (const consumerSession of consumerSessions.values()) {
          expect(consumerSession.epoch).toEqual(cm.getCurrentEpoch());
          expect(consumerSession.session.latestRelayCu).toEqual(
            CU_FOR_FIRST_REQUEST
          );
          expect(cm.getNumberOfResets()).toEqual(numberOfResets + 1);
        }
      }

      const consumerSessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        0
      );
      if (consumerSessions instanceof Error) {
        throw consumerSessions;
      }

      for (const consumerSession of consumerSessions.values()) {
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
          RELAY_NUMBER_AFTER_FIRST_FAIL
        );
        expect(consumerSession.session.latestBlock).toEqual(
          SERVICED_BLOCK_NUMBER
        );
      }
    });

    it("tests success and failure of session with update pairings in the middle", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      const sessionList: { cs: SingleConsumerSession; epoch: number }[] = [];
      const sessionListData: { relayNum: number; cuSum: number }[] = [];
      for (let i = 0; i < NUMBER_OF_ALLOWED_SESSIONS_PER_CONSUMER; i++) {
        const consumerSessions = cm.getSessions(
          CU_FOR_FIRST_REQUEST,
          new Set(),
          SERVICED_BLOCK_NUMBER,
          "",
          [],
          NOSTATE,
          0
        );
        if (consumerSessions instanceof Error) {
          throw consumerSessions;
        }

        for (const consumerSession of consumerSessions.values()) {
          expect(consumerSession.epoch).toEqual(cm.getCurrentEpoch());
          expect(consumerSession.session.latestRelayCu).toEqual(
            CU_FOR_FIRST_REQUEST
          );

          sessionList.push({
            cs: consumerSession.session,
            epoch: consumerSession.epoch,
          });
        }
      }

      for (let j = 0; j < NUMBER_OF_ALLOWED_SESSIONS_PER_CONSUMER / 2; j++) {
        const { cs, epoch } = sessionList[j];
        expect(epoch).toEqual(cm.getCurrentEpoch());

        if (Math.random() > 0.5) {
          cm.onSessionDone(
            cs,
            SERVICED_BLOCK_NUMBER,
            CU_FOR_FIRST_REQUEST,
            0,
            cs.calculateExpectedLatency(2),
            SERVICED_BLOCK_NUMBER - 1,
            NUMBER_OF_PROVIDERS,
            NUMBER_OF_PROVIDERS,
            false
          );
          expect(cs.cuSum).toEqual(CU_FOR_FIRST_REQUEST);
          expect(cs.latestRelayCu).toEqual(LATEST_RELAY_CU_AFTER_DONE);
          expect(cs.relayNum).toEqual(RELAY_NUMBER_AFTER_FIRST_CALL);
          expect(cs.latestBlock).toEqual(SERVICED_BLOCK_NUMBER);
          sessionListData.push({
            cuSum: CU_FOR_FIRST_REQUEST,
            relayNum: 1,
          });
        } else {
          cm.onSessionFailure(cs);
          expect(cs.cuSum).toEqual(0);
          expect(cs.relayNum).toEqual(RELAY_NUMBER_AFTER_FIRST_FAIL);
          expect(cs.latestRelayCu).toEqual(LATEST_RELAY_CU_AFTER_DONE);
          sessionListData.push({
            cuSum: 0,
            relayNum: 1,
          });
        }
      }

      for (let i = 0; i < NUMBER_OF_ALLOWED_SESSIONS_PER_CONSUMER; i++) {
        const consumerSessions = cm.getSessions(
          CU_FOR_FIRST_REQUEST,
          new Set(),
          SERVICED_BLOCK_NUMBER,
          "",
          [],
          NOSTATE,
          0
        );
        if (consumerSessions instanceof Error) {
          throw consumerSessions;
        }

        for (const consumerSession of consumerSessions.values()) {
          expect(consumerSession.epoch).toEqual(cm.getCurrentEpoch());
          expect(consumerSession.session.latestRelayCu).toEqual(
            CU_FOR_FIRST_REQUEST
          );
        }
      }

      await cm.updateAllProviders(
        SECOND_EPOCH_HEIGHT,
        createPairingList("test2", true)
      );

      for (
        let j = NUMBER_OF_ALLOWED_SESSIONS_PER_CONSUMER / 2;
        j < NUMBER_OF_ALLOWED_SESSIONS_PER_CONSUMER;
        j++
      ) {
        const cs = sessionList[j].cs;

        if (Math.random() > 0.5) {
          cm.onSessionDone(
            cs,
            SERVICED_BLOCK_NUMBER,
            CU_FOR_FIRST_REQUEST,
            0,
            cs.calculateExpectedLatency(2),
            SERVICED_BLOCK_NUMBER - 1,
            NUMBER_OF_PROVIDERS,
            NUMBER_OF_PROVIDERS,
            false
          );

          const cuSum = sessionListData[j]?.cuSum || 0;
          expect(cuSum + CU_FOR_FIRST_REQUEST).toEqual(cs.cuSum);
        } else {
          cm.onSessionFailure(cs);

          const cuSum = sessionListData[j]?.cuSum || 0;
          const relayNum = sessionListData[j]?.relayNum || 0;

          expect(cuSum).toEqual(cs.cuSum);
          expect(cs.relayNum).toEqual(relayNum + 1);
          expect(cs.latestRelayCu).toEqual(LATEST_RELAY_CU_AFTER_DONE);
        }
      }
    });

    it("tests session failure and get reported providers", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      const consumerSessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        0
      );
      if (consumerSessions instanceof Error) {
        throw consumerSessions;
      }

      for (const consumerSession of consumerSessions.values()) {
        expect(consumerSession.epoch).toEqual(cm.getCurrentEpoch());
        expect(consumerSession.session.latestRelayCu).toEqual(
          CU_FOR_FIRST_REQUEST
        );

        cm.onSessionFailure(
          consumerSession.session,
          new ReportAndBlockProviderError()
        );
        expect(consumerSession.session.client.usedComputeUnits).toEqual(
          CU_SUM_ON_FAILURE
        );
        expect(consumerSession.session.cuSum).toEqual(CU_SUM_ON_FAILURE);
        expect(consumerSession.session.latestRelayCu).toEqual(
          LATEST_RELAY_CU_AFTER_DONE
        );
        expect(consumerSession.session.relayNum).toEqual(
          RELAY_NUMBER_AFTER_FIRST_FAIL
        );

        const rp = cm.getReportedProviders(FIRST_EPOCH_HEIGHT);
        const allReported = [];
        for (const r of rp) {
          allReported.push(r.getAddress());
        }
        expect(allReported).toContain(
          consumerSession.session.client.publicLavaAddress
        );
        expect(cm.validAddresses).not.toContain(
          consumerSession.session.client.publicLavaAddress
        );
      }
    });

    it("tests session failure epoch mismatch", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      const consumerSessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        0
      );
      if (consumerSessions instanceof Error) {
        throw consumerSessions;
      }

      for (const consumerSession of consumerSessions.values()) {
        expect(consumerSession.epoch).toEqual(cm.getCurrentEpoch());
        expect(consumerSession.session.latestRelayCu).toEqual(
          CU_FOR_FIRST_REQUEST
        );

        const error = await cm.updateAllProviders(
          FIRST_EPOCH_HEIGHT,
          pairingList
        );
        if (error) {
          cm.onSessionFailure(
            consumerSession.session,
            new ReportAndBlockProviderError()
          );
        }
      }
    });

    it("tests all providers endpoints disabled", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", false);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);
      expect(cm.validAddresses.size).toEqual(NUMBER_OF_PROVIDERS);
      expect(cm.getPairingAddressesLength()).toEqual(NUMBER_OF_PROVIDERS);

      const sessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        0
      );

      expect(sessions).toBeInstanceOf(PairingListEmptyError);
    });

    describe("tests pairing with addons", () => {
      test.each(["", "addon"])(`addon: %s`, async (addon) => {
        const cm = setupConsumerSessionManager();
        const pairingList = createPairingList("", true);
        await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);
        expect(cm.getValidAddresses(addon, [])).not.toEqual(0);

        const initialProvidersLength = cm.getValidAddresses(addon, []).size;
        for (let i = 0; i < initialProvidersLength; i++) {
          const consumerSessions = cm.getSessions(
            CU_FOR_FIRST_REQUEST,
            new Set(),
            SERVICED_BLOCK_NUMBER,
            addon,
            [],
            NOSTATE,
            0
          );
          if (consumerSessions instanceof Error) {
            throw consumerSessions;
          }

          for (const consumerSession of consumerSessions.values()) {
            cm.onSessionFailure(
              consumerSession.session,
              new ReportAndBlockProviderError()
            );
          }
        }

        expect(cm.getValidAddresses(addon, []).size).toEqual(0);

        if (addon !== "") {
          expect(cm.getValidAddresses("addon", []).size).toEqual(0);
        }

        const consumerSessions = cm.getSessions(
          CU_FOR_FIRST_REQUEST,
          new Set(),
          SERVICED_BLOCK_NUMBER,
          addon,
          [],
          NOSTATE,
          0
        );
        if (consumerSessions instanceof Error) {
          throw consumerSessions;
        }

        for (const consumerSession of consumerSessions.values()) {
          cm.onSessionDone(
            consumerSession.session,
            SERVICED_BLOCK_NUMBER,
            CU_FOR_FIRST_REQUEST,
            1,
            consumerSession.session.calculateExpectedLatency(2),
            SERVICED_BLOCK_NUMBER - 1,
            NUMBER_OF_PROVIDERS,
            NUMBER_OF_PROVIDERS,
            false
          );
        }
      });
    });

    describe("tests pairing with extensions", () => {
      const extensionOptions = [
        {
          name: "empty",
          addon: "",
          extensions: [],
        },
        {
          name: "one ext",
          addon: "",
          extensions: ["ext1"],
        },
        {
          name: "two exts",
          addon: "",
          extensions: ["ext1", "ext2"],
        },
        {
          name: "one ext addon",
          addon: "addon",
          extensions: ["ext1"],
        },
        {
          name: "two exts addon",
          addon: "addon",
          extensions: ["ext1", "ext2"],
        },
      ];

      test.each(extensionOptions)(`$name`, async ({ addon, extensions }) => {
        const cm = setupConsumerSessionManager();
        const pairingList = createPairingList("", true);
        await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);
        expect(cm.getValidAddresses(addon, extensions)).not.toEqual(0);

        const initialProvidersLength = cm.getValidAddresses(
          addon,
          extensions
        ).size;
        for (let i = 0; i < initialProvidersLength; i++) {
          const consumerSessions = cm.getSessions(
            CU_FOR_FIRST_REQUEST,
            new Set(),
            SERVICED_BLOCK_NUMBER,
            addon,
            extensions,
            NOSTATE,
            0
          );
          if (consumerSessions instanceof Error) {
            throw consumerSessions;
          }

          for (const consumerSession of consumerSessions.values()) {
            cm.onSessionFailure(
              consumerSession.session,
              new ReportAndBlockProviderError()
            );
          }
        }

        expect(cm.getValidAddresses(addon, extensions).size).toEqual(0);

        if (extensions.length !== 0 || addon !== "") {
          expect(cm.getValidAddresses("addon", extensions).size).toEqual(0);
        }

        const consumerSessions = cm.getSessions(
          CU_FOR_FIRST_REQUEST,
          new Set(),
          SERVICED_BLOCK_NUMBER,
          addon,
          extensions,
          NOSTATE,
          0
        );
        if (consumerSessions instanceof Error) {
          throw consumerSessions;
        }

        for (const consumerSession of consumerSessions.values()) {
          cm.onSessionDone(
            consumerSession.session,
            SERVICED_BLOCK_NUMBER,
            CU_FOR_FIRST_REQUEST,
            1,
            consumerSession.session.calculateExpectedLatency(2),
            SERVICED_BLOCK_NUMBER - 1,
            NUMBER_OF_PROVIDERS,
            NUMBER_OF_PROVIDERS,
            false
          );
        }
      });
    });
  });

  describe("updateAllProviders", () => {
    it("updates providers", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      expect(cm.validAddresses.size).toEqual(NUMBER_OF_PROVIDERS);
      expect(cm.getPairingAddressesLength()).toEqual(NUMBER_OF_PROVIDERS);
      expect(cm.getCurrentEpoch()).toEqual(FIRST_EPOCH_HEIGHT);
      const validAddressesArray = Array.from(cm.validAddresses);
      for (let i = 0; i < NUMBER_OF_PROVIDERS; i++) {
        expect(validAddressesArray[i]).toEqual(`provider${i}`);
      }
    });

    it("updates all providers with same epoch", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      const err = await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);
      expect(err?.message).toMatch(
        new RegExp(`^Trying to update provider list for older epoch?`)
      );

      expect(cm.validAddresses.size).toEqual(NUMBER_OF_PROVIDERS);
      expect(cm.getPairingAddressesLength()).toEqual(NUMBER_OF_PROVIDERS);
      expect(cm.getCurrentEpoch()).toEqual(FIRST_EPOCH_HEIGHT);
      const validAddressesArray = Array.from(cm.validAddresses);
      for (let i = 0; i < NUMBER_OF_PROVIDERS; i++) {
        expect(validAddressesArray[i]).toEqual(`provider${i}`);
      }
    });

    it("retries failing providers", async () => {
      const pairingList = createPairingList("", true);
      const relayer = setupRelayer();
      let providerRetries = 0;

      jest
        .spyOn(relayer, "probeProvider")
        .mockImplementation(async (providerAddress, _apiInterface, guid) => {
          if (providerAddress === pairingList[1].publicLavaAddress) {
            providerRetries++;
            throw new Error("test");
          }

          const response: ProbeReply = new ProbeReply();
          response.setLatestBlock(42);
          response.setLavaEpoch(20);
          response.setGuid(guid);
          return Promise.resolve(response);
        });

      const cm = setupConsumerSessionManager(relayer);
      // @ts-expect-error - we are spying on a private method
      jest.spyOn(cm, "timeoutBetweenProbes").mockImplementation(() => 1); // this makes it not sleep
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      await sleep(TIMEOUT_BETWEEN_PROBES * ALLOWED_PROBE_RETRIES);

      // 1 for the initial call and 5 retries
      expect(providerRetries).toEqual(ALLOWED_PROBE_RETRIES); // after ALLOWED_PROBE_RETRIES he gets banned
    });

    it("disables provider until first successful probe", async () => {
      const pairingList = createPairingList("", true);
      const relayer = setupRelayer();
      let providerRetries = 0;

      jest
        .spyOn(relayer, "probeProvider")
        .mockImplementation(async (providerAddress, _apiInterface, guid) => {
          if (
            providerAddress === pairingList[1].publicLavaAddress &&
            providerRetries < 1
          ) {
            providerRetries++;
            throw new Error("test");
          }

          const response: ProbeReply = new ProbeReply();
          response.setLatestBlock(42);
          response.setLavaEpoch(20);
          response.setGuid(guid);
          return Promise.resolve(response);
        });

      const optimizer = setupProviderOptimizer(
        ProviderOptimizerStrategy.Latency,
        pairingList.length
      );
      const cm = setupConsumerSessionManager(relayer, optimizer);
      // @ts-expect-error - we are spying on a private method
      jest.spyOn(cm, "timeoutBetweenProbes").mockImplementation(() => 1);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      expect(cm.validAddresses).toContain(pairingList[1].publicLavaAddress);

      await sleep((TIMEOUT_BETWEEN_PROBES / 2) * ALLOWED_PROBE_RETRIES);

      expect(cm.validAddresses).toContain(pairingList[1].publicLavaAddress);
    });

    it("disables provider for failed probes", async () => {
      const pairingList = createPairingList("", true);
      const relayer = setupRelayer();

      jest
        .spyOn(relayer, "probeProvider")
        .mockImplementation(async (providerAddress, _apiInterface, guid) => {
          if (providerAddress == pairingList[1].publicLavaAddress) {
            throw new Error("test");
          }
          const response: ProbeReply = new ProbeReply();
          response.setLatestBlock(42);
          response.setLavaEpoch(20);
          response.setGuid(guid);
          return Promise.resolve(response);
        });

      const optimizer = setupProviderOptimizer(
        ProviderOptimizerStrategy.Latency,
        pairingList.length
      );
      const cm = setupConsumerSessionManager(relayer, optimizer);
      // @ts-expect-error - we are spying on a private method
      jest.spyOn(cm, "timeoutBetweenProbes").mockImplementation(() => 1);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      expect(cm.validAddresses).toContain(pairingList[1].publicLavaAddress);

      await sleep(TIMEOUT_BETWEEN_PROBES * ALLOWED_PROBE_RETRIES + 100);

      expect(cm.validAddresses).not.toContain(pairingList[1].publicLavaAddress);
    });

    it("returns the median latest block", async () => {
      const relayer = setupRelayer();
      let startEpoch = 1;
      jest
        .spyOn(relayer, "probeProvider")
        .mockImplementation((_providerAddress, _apiInterface, guid) => {
          const response: ProbeReply = new ProbeReply();
          response.setLavaEpoch(startEpoch++);
          response.setGuid(guid);
          return Promise.resolve(response);
        });

      const cm = setupConsumerSessionManager(relayer);
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      expect(cm.getEpochFromEpochTracker()).toEqual(NUMBER_OF_PROVIDERS / 2);
    });
  });
  describe("statful", () => {
    it("test pairing", async () => {
      const cm = setupConsumerSessionManager();
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(FIRST_EPOCH_HEIGHT, pairingList);

      expect(cm.validAddresses.size).toEqual(NUMBER_OF_PROVIDERS);
      expect(cm.getPairingAddressesLength()).toEqual(NUMBER_OF_PROVIDERS);
      expect(cm.getCurrentEpoch()).toEqual(FIRST_EPOCH_HEIGHT);
      let sessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        CONSISTENCY_SELECT_ALLPROVIDERS,
        0
      );
      expect(sessions).not.toBeInstanceOf(Error);
      if (!(sessions instanceof Error)) {
        expect(sessions.size).toEqual(NUMBER_OF_PROVIDERS);
      }
      sessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        new Set(),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        NOSTATE,
        0
      );
      expect(sessions).not.toBeInstanceOf(Error);
      if (!(sessions instanceof Error)) {
        expect(sessions.size).not.toEqual(NUMBER_OF_PROVIDERS);
      }
      sessions = cm.getSessions(
        CU_FOR_FIRST_REQUEST,
        new Set([Array.from(cm.validAddresses)[0]]),
        SERVICED_BLOCK_NUMBER,
        "",
        [],
        CONSISTENCY_SELECT_ALLPROVIDERS,
        0
      );
      expect(sessions).not.toBeInstanceOf(Error);
      if (!(sessions instanceof Error)) {
        expect(sessions.size).toEqual(NUMBER_OF_PROVIDERS - 1);
      }
    });
  });
});

function createPairingList(
  providerPrefixAddress: string,
  enabled: boolean
): ConsumerSessionsWithProvider[] {
  const sessionsWithProvider: ConsumerSessionsWithProvider[] = [];
  const pairingEndpoints: Endpoint[] = [
    {
      networkAddress: "",
      extensions: new Set(),
      addons: new Set(),
      connectionRefusals: 0,
      enabled,
      geolocation: 1,
    },
  ];
  const pairingEndpointsWithAddon: Endpoint[] = [
    {
      networkAddress: "",
      extensions: new Set(),
      addons: new Set(["addon"]),
      connectionRefusals: 0,
      enabled,
      geolocation: 1,
    },
  ];
  const pairingEndpointsWithExtension: Endpoint[] = [
    {
      networkAddress: "",
      extensions: new Set(["ext1"]),
      addons: new Set(["addon"]),
      connectionRefusals: 0,
      enabled,
      geolocation: 1,
    },
  ];
  const pairingEndpointsWithExtensions: Endpoint[] = [
    {
      networkAddress: "",
      extensions: new Set(["ext1", "ext2"]),
      addons: new Set(["addon"]),
      connectionRefusals: 0,
      enabled,
      geolocation: 1,
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

    sessionsWithProvider.push(
      new ConsumerSessionsWithProvider(
        "provider" + providerPrefixAddress + i,
        [
          {
            ...endpoints[0],
            networkAddress: "provider" + providerPrefixAddress + i,
          },
        ],
        {},
        200,
        FIRST_EPOCH_HEIGHT
      )
    );
  }

  return sessionsWithProvider;
}
