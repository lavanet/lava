import { StateTracker } from "../state_tracker";
import {
  DEFAULT_GEOLOCATION,
  DEFAULT_LAVA_PAIRING_NETWORK,
} from "../../config/default";
import { RPCConsumerServer } from "../../rpcconsumer/rpcconsumer_server";
import { Relayer } from "../../relayer/relayer";
import {
  ProviderOptimizer,
  ProviderOptimizerStrategy,
} from "../../providerOptimizer/providerOptimizer";
import { ConsumerSessionManager } from "../../lavasession/consumerSessionManager";
import {
  ConsumerSessionsWithProvider,
  Endpoint,
  RPCEndpoint,
} from "../../lavasession/consumerTypes";
import { AverageWorldLatency } from "../../common/timeout";
import { ProbeReply } from "../../grpc_web_services/lavanet/lava/pairing/relay_pb";
import { TendermintRpcChainParser } from "../../chainlib/tendermint";
import { FinalizationConsensus } from "../../lavaprotocol/finalization_consensus";
import { ConsumerConsistency } from "../../rpcconsumer/consumerConsistency";
import { getDefaultLavaSpec } from "../../chainlib/default_lava_spec";
import { Params } from "../../grpc_web_services/lavanet/lava/downtime/v1/downtime_pb";
import { Duration } from "google-protobuf/google/protobuf/duration_pb";
import { StateChainQuery } from "../stateQuery/state_chain_query";

function setupConsumerSessionManager(
  relayer?: Relayer,
  optimizer?: ProviderOptimizer
) {
  if (!relayer) {
    relayer = setupRelayer();
    jest
      .spyOn(relayer, "probeProvider")
      .mockImplementation((providerAddress, apiInterface, guid, specId) => {
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
    new RPCEndpoint("", "LAV1", "", DEFAULT_GEOLOCATION),
    optimizer
  );

  return cm;
}

function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function setupRelayer(): Relayer {
  return new Relayer({
    allowInsecureTransport: true,
    lavaChainId: "LAV1",
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

describe("EmergencyTracker", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });
  jest.setTimeout(10000000);

  describe("test", () => {
    it("EmergencyTrackerVirtualEpoch", async () => {
      const relayer = setupRelayer();
      const cm = setupConsumerSessionManager(relayer);
      const pairingList = createPairingList("", true);
      await cm.updateAllProviders(1, pairingList);

      const rpcEndpoint = new RPCEndpoint(
        "", // We do no need this in sdk as we are not opening any ports
        "LAV1",
        "tendermintrpc",
        DEFAULT_GEOLOCATION // This is also deprecated
      );
      const finalizationConsensus = new FinalizationConsensus();
      const consumerConsistency = new ConsumerConsistency("LAV1");

      const rpcConsumerServerLoL = new RPCConsumerServer(
        relayer,
        cm,
        new TendermintRpcChainParser(),
        DEFAULT_GEOLOCATION,
        rpcEndpoint,
        "LAV1",
        finalizationConsensus,
        consumerConsistency
      );
      const spec = getDefaultLavaSpec();

      const stateTracker = new StateTracker(
        "",
        relayer,
        ["LAV1"],
        {
          geolocation: DEFAULT_GEOLOCATION,
          network: DEFAULT_LAVA_PAIRING_NETWORK,
        },
        rpcConsumerServerLoL,
        spec,
        {
          algo: "secp256k1",
          address: "",
          pubkey: new Uint8Array([]),
        },
        ""
      );
      rpcConsumerServerLoL.setEmergencyTracker(
        stateTracker.getEmergencyTracker()
      );

      const emergencyTracker = stateTracker["emergencyTracker"];

      const params = new Params();
      params.setEpochDuration(new Duration().setSeconds(4));
      params.setDowntimeDuration(new Duration().setSeconds(2));

      emergencyTracker["downtimeParams"] = params;

      (stateTracker["stateQuery"] as StateChainQuery)["currentEpoch"] = 10;
      await emergencyTracker.update();

      expect(emergencyTracker.getVirtualEpoch()).toEqual(0);

      // wait downtime duration
      await delay(2000);
      expect(emergencyTracker.getVirtualEpoch()).toEqual(0);

      // wait until the end of normal epoch
      await delay(2000);
      expect(emergencyTracker.getVirtualEpoch()).toEqual(1);

      // wait until the end of first virtual epoch
      await delay(4000);
      expect(emergencyTracker.getVirtualEpoch()).toEqual(2);

      // call update on the same epoch
      await emergencyTracker.update();
      expect(emergencyTracker.getVirtualEpoch()).toEqual(2);

      // increase epoch and call update to reset virtual epoch
      (stateTracker["stateQuery"] as StateChainQuery)["currentEpoch"] = 20;
      await emergencyTracker.update();

      expect(emergencyTracker.getVirtualEpoch()).toEqual(0);
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

  for (let i = 0; i < 10; i++) {
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
        20
      )
    );
  }

  return sessionsWithProvider;
}
