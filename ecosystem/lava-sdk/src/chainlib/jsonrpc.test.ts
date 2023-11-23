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
import {
  JsonrpcBatchMessage,
  JsonrpcMessage,
} from "./chainproxy/rpcInterfaceMessages/json_rpc_message";
import { JsonRpcChainParser } from "./jsonrpc";

describe("ParseJsonRPCMessage", () => {
  let parser: JsonRpcChainParser;
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

    collectionData.setType("POST");
    collectionData.setApiInterface("jsonrpc");

    apiCollection.setApisList([api1, api2]);
    apiCollection.setCollectionData(collectionData);
    apiCollection.setEnabled(true);

    spec.setApiCollectionsList([apiCollection]);
    spec.setEnabled(true);
  });

  beforeEach(() => {
    parser = new JsonRpcChainParser();
    parser.init(spec);
  });

  it("parses a single relay message correctly", () => {
    const id = 123;
    const options: SendRelayOptions = {
      method: "method1",
      params: ["param1", "param2"],
      id: id,
      apiInterface: "jsonrpc",
    };

    const expectedJsonRpcMessage = new JsonrpcMessage();
    expectedJsonRpcMessage.initJsonrpcMessage(
      JsonRPCVersion,
      id.toString(),
      options.method,
      options.params
    );
    expectedJsonRpcMessage.initBaseMessage({
      headers: [],
      latestBlockHeaderSetter: undefined,
    });

    const result = parser.parseMsg(options);
    expect(result).toBeInstanceOf(BaseChainMessageContainer);
    const jsonRpcMessage = result.getRPCMessage();
    expect(jsonRpcMessage).toBeInstanceOf(JsonrpcMessage);
    expect(jsonRpcMessage).toStrictEqual(expectedJsonRpcMessage);
  });

  it("parses a batch of relay messages correctly", () => {
    const id = 123;
    const options: SendRelaysBatchOptions = {
      relays: [
        { method: "method1", params: ["param1"], id: id },
        { method: "method2", params: ["param2"] },
      ],
      apiInterface: "jsonrpc",
    };

    const generateJsonpcMsg = (method: string, params: any, id?: number) => {
      const jsonrpcMsg = new JsonrpcMessage();
      jsonrpcMsg.initJsonrpcMessage(
        JsonRPCVersion,
        id?.toString() ?? "",
        method,
        params
      );
      jsonrpcMsg.initBaseMessage({
        headers: [],
        latestBlockHeaderSetter: undefined,
      });
      return jsonrpcMsg;
    };

    const expectedJsonRpcBatchMessage = new JsonrpcBatchMessage();
    expectedJsonRpcBatchMessage.initJsonrpcBatchMessage([
      generateJsonpcMsg(options.relays[0].method, options.relays[0].params, id),
      generateJsonpcMsg(options.relays[1].method, options.relays[1].params),
    ]);

    const result = parser.parseMsg(options);
    expect(result).toBeInstanceOf(BaseChainMessageContainer);

    const jsonRpcBatchMessage = result.getRPCMessage();
    expect(jsonRpcBatchMessage).toBeInstanceOf(JsonrpcBatchMessage);
    const secondMessageId =
      (jsonRpcBatchMessage as JsonrpcBatchMessage).batch[1].id ?? "-1";

    expect(Number.parseInt(secondMessageId)).toBeGreaterThan(0);

    // Generated Id
    expectedJsonRpcBatchMessage.batch[1].id = secondMessageId;
    expect(jsonRpcBatchMessage).toStrictEqual(expectedJsonRpcBatchMessage);
  });

  it("throws an error with SendRestRelayOptions", () => {
    const options: SendRestRelayOptions = {
      connectionType: "GET",
      url: "http://example.com",
    };

    expect(() => parser.parseMsg(options)).toThrow();
  });
});
