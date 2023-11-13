import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceJsonRPC,
  HeadersPassSend,
} from "../chainlib/base_chain_parser";
import { Logger } from "../logger/logger";
import { generateRPCData } from "../util/common";
import { HttpMethod, NOT_APPLICABLE } from "../common/common";
import { FUNCTION_TAG } from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";
import { JsonrpcMessage } from "./chainproxy/rpcInterfaceMessages/json_rpc_message";
import { Parser } from "../parser/parser";
import { BaseChainMessageContainer } from "./chain_message";

const jsonrpcVersion = "2.0";
export class JsonRpcChainParser extends BaseChainParser {
  constructor() {
    super();
    this.apiInterface = APIInterfaceJsonRPC;
  }
  parseMsg(
    options: SendRelayOptions | SendRestRelayOptions
  ): BaseChainMessageContainer {
    if (this.isRest(options)) {
      throw Logger.fatal(
        "Wrong relay options provided, expected SendRestRelayOptions got SendRelayOptions"
      );
    }

    const apiCont = this.getSupportedApi(options.method, HttpMethod.POST);
    const apiCollection = this.getApiCollection({
      addon: apiCont.collectionKey.addon,
      connectionType: HttpMethod.POST,
      internalPath: apiCont.collectionKey.internalPath,
    });

    const headerHandler = this.handleHeaders(
      options.metadata,
      apiCollection,
      HeadersPassSend
    );

    const [settingHeaderDirective] = this.getParsingByTag(
      FUNCTION_TAG.SET_LATEST_IN_METADATA
    );

    const jsonrpcMessage = new JsonrpcMessage();
    jsonrpcMessage.initJsonrpcMessage(
      jsonrpcVersion,
      String(options.id ?? Math.floor(Math.random() * Number.MAX_SAFE_INTEGER)),
      options.method,
      options.params
    );
    jsonrpcMessage.initBaseMessage({
      headers: headerHandler.filteredHeaders,
      latestBlockHeaderSetter: settingHeaderDirective,
    });

    const blockParser = apiCont.api.getBlockParsing();
    if (!blockParser) {
      throw Logger.fatal("BlockParsing is missing");
    }

    let requestedBlock: number | Error;

    const overwriteRequestedBlock = headerHandler.overwriteRequestedBlock;
    if (overwriteRequestedBlock === "") {
      requestedBlock = Parser.parseBlockFromParams(jsonrpcMessage, blockParser);
      if (requestedBlock instanceof Error) {
        Logger.error(
          `ParseBlockFromParams failed parsing block for chain: ${this.spec?.getName()}`,
          blockParser,
          requestedBlock
        );
        requestedBlock = NOT_APPLICABLE;
      }
    } else {
      requestedBlock = jsonrpcMessage.parseBlock(overwriteRequestedBlock);
      if (requestedBlock instanceof Error) {
        Logger.error(
          `Failed parsing block from an overwrite header for chain: ${this.spec?.getName()}, overwriteRequestedBlock: ${overwriteRequestedBlock}`,
          requestedBlock
        );
        requestedBlock = NOT_APPLICABLE;
      }
    }

    // TODO: add extension parsing.

    return new BaseChainMessageContainer(
      apiCont.api,
      requestedBlock,
      jsonrpcMessage,
      apiCollection,
      generateRPCData(jsonrpcMessage)
    );
  }
}
