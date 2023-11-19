import {
  Api,
  ApiCollection,
  BlockParser,
  CollectionData,
  PARSER_FUNC,
} from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";
import { Spec } from "../grpc_web_services/lavanet/lava/spec/spec_pb";
import {
  SendRelayOptions,
  SendRelaysBatchOptions,
  SendRestRelayOptions,
} from "./base_chain_parser";
import { BaseChainMessageContainer } from "./chain_message";
import { JsonRPCVersion } from "./chainproxy/consts";
import { JsonrpcBatchMessage } from "./chainproxy/rpcInterfaceMessages/json_rpc_message";
import { TendermintrpcMessage } from "./chainproxy/rpcInterfaceMessages/tendermint_rpc_message";
import { TendermintRpcChainParser } from "./tendermint";

describe("ParseTendermintRPCMessage", () => {
  let parser: TendermintRpcChainParser;
  let spec: Spec;

  beforeAll(() => {
    spec = new Spec();
    const apiCollection = new ApiCollection();
    const collectionData = new CollectionData();
    const api1 = new Api();
    const api2 = new Api();
    const blockParser = new BlockParser();

    blockParser.setParserFunc(PARSER_FUNC.DEFAULT);
    blockParser.setParserArgList(["latest"]);

    api1.setName("method1");
    api1.setEnabled(true);
    api1.setBlockParsing(blockParser);

    api2.setName("method2");
    api2.setEnabled(true);
    api2.setBlockParsing(blockParser);

    collectionData.setApiInterface("tendermintrpc");

    apiCollection.setApisList([api1, api2]);
    apiCollection.setCollectionData(collectionData);
    apiCollection.setEnabled(true);

    spec.setApiCollectionsList([apiCollection]);
    spec.setEnabled(true);
  });

  beforeEach(() => {
    parser = new TendermintRpcChainParser();
    parser.init(spec);
  });

  it("parses a single relay message correctly", () => {
    const id = 123;
    const options: SendRelayOptions = {
      method: "method1",
      params: ["param1", "param2"],
      id: id,
      apiInterface: "tendermintrpc",
    };

    const expectedTendermintRpcMessage = new TendermintrpcMessage();
    expectedTendermintRpcMessage.initJsonrpcMessage(
      JsonRPCVersion,
      id.toString(),
      options.method,
      options.params
    );
    expectedTendermintRpcMessage.initBaseMessage({
      headers: [],
      latestBlockHeaderSetter: undefined,
    });

    const result = parser.parseMsg(options);
    expect(result).toBeInstanceOf(BaseChainMessageContainer);
    const tendermintRpcMessage = result.getRPCMessage();
    expect(tendermintRpcMessage).toBeInstanceOf(TendermintrpcMessage);
    expect(tendermintRpcMessage).toStrictEqual(expectedTendermintRpcMessage);
  });

  it("parses a batch of relay messages correctly", () => {
    const id = 123;
    const options: SendRelaysBatchOptions = {
      relays: [
        { method: "method1", params: ["param1"], id: id },
        { method: "method2", params: ["param2"] },
      ],
      apiInterface: "tendermintrpc",
    };

    const generateTendermintrpcMsg = (
      method: string,
      params: any,
      id?: number
    ) => {
      const tendermintrpcMsg = new TendermintrpcMessage();
      tendermintrpcMsg.initJsonrpcMessage(
        JsonRPCVersion,
        id?.toString() ?? "",
        method,
        params
      );
      tendermintrpcMsg.initBaseMessage({
        headers: [],
        latestBlockHeaderSetter: undefined,
      });
      return tendermintrpcMsg;
    };

    const expectedTendermintRpcBatchMessage = new JsonrpcBatchMessage();
    expectedTendermintRpcBatchMessage.initJsonrpcBatchMessage([
      generateTendermintrpcMsg(
        options.relays[0].method,
        options.relays[0].params,
        id
      ),
      generateTendermintrpcMsg(
        options.relays[1].method,
        options.relays[1].params
      ),
    ]);

    const result = parser.parseMsg(options);
    expect(result).toBeInstanceOf(BaseChainMessageContainer);

    const jsonRpcBatchMessage = result.getRPCMessage();
    expect(jsonRpcBatchMessage).toBeInstanceOf(JsonrpcBatchMessage);
    const secondMessageId =
      (jsonRpcBatchMessage as JsonrpcBatchMessage).batch[1].id ?? "-1";

    expect(Number.parseInt(secondMessageId)).toBeGreaterThan(0);

    // Generated Id
    expectedTendermintRpcBatchMessage.batch[1].id = secondMessageId;
    expect(jsonRpcBatchMessage).toStrictEqual(
      expectedTendermintRpcBatchMessage
    );
  });

  it("throws an error with SendRestRelayOptions", () => {
    const options: SendRestRelayOptions = {
      connectionType: "GET",
      url: "http://example.com",
    };

    expect(() => parser.parseMsg(options)).toThrow();
  });
});
