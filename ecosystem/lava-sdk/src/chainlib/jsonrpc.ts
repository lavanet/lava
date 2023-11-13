import {
  BaseChainParser,
  SendRelayOptions,
  SendRelaysBatchOptions,
  SendRestRelayOptions,
  APIInterfaceJsonRPC,
  HeadersPassSend,
  ApiContainer,
} from "../chainlib/base_chain_parser";
import { Logger } from "../logger/logger";
import { generateBatchRPCData, generateRPCData } from "../util/common";
import { HttpMethod, NOT_APPLICABLE } from "../common/common";
import {
  JsonrpcMessage,
  newBatchMessage,
} from "./chainproxy/rpcInterfaceMessages/json_rpc_message";
import { Parser } from "../parser/parser";
import { ParsedMessage } from "./chain_message";
import { JsonRPCVersion } from "./chainproxy/consts";
import {
  Api,
  ApiCollection,
  BlockParser,
  FUNCTION_TAG,
  PARSER_FUNC,
  SpecCategory,
} from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";
import { CombineSpecCategories } from "../util/apiCollection";
import { compareRequestedBlockInBatch } from "./common";

const SEP = "&";
export class JsonRpcChainParser extends BaseChainParser {
  constructor() {
    super();
    this.apiInterface = APIInterfaceJsonRPC;
  }
  parseMsg(
    options: SendRelayOptions | SendRelaysBatchOptions | SendRestRelayOptions
  ): ParsedMessage {
    if (this.isRest(options)) {
      throw Logger.fatal(
        "Wrong relay options provided, expected SendRestRelayOptions got SendRelayOptions"
      );
    }

    if ("relays" in options) {
      // options is SendRelaysBatchOptions
      return this.parseBatchMsg(options);
    }

    const [apiCont, apiCollection, latestRequestedBlock, jsonrpcMessage] =
      this.parseSingleMessage(options);

    // TODO: add extension parsing.

    return new ParsedMessage(
      apiCont.api,
      latestRequestedBlock,
      jsonrpcMessage,
      apiCollection,
      generateRPCData(jsonrpcMessage)
    );
  }

  private parseBatchMsg(options: SendRelaysBatchOptions) {
    let api: Api | undefined;
    let apiCollection: ApiCollection | undefined;
    let latestRequestedBlock = 0;
    let earliestRequestedBlock = 0;
    const jsonrpcMsgs: JsonrpcMessage[] = [];

    for (let idx = 0; idx < options.relays.length; idx++) {
      const relay = options.relays[idx];
      const sendRelayOptions = {
        method: relay.method,
        params: relay.params,
        id: relay.id,
        chainId: options.chainId,
        metadata: relay.metadata,
        apiInterface: options.apiInterface,
      };

      const [
        apiCont,
        apiCollectionForMessage,
        requestedBlockForMessage,
        jsonrpcMessage,
      ] = this.parseSingleMessage(sendRelayOptions);

      jsonrpcMsgs.push(jsonrpcMessage);

      if (idx === 0) {
        // on the first entry store them
        api = apiCont.api;
        apiCollection = apiCollectionForMessage;
        latestRequestedBlock = requestedBlockForMessage;
      } else {
        // on next entries we need to compare to existing data
        if (api === undefined) {
          throw Logger.fatal("Invalid parsing. First index were skipped");
        }

        let category = api.getCategory() ?? new SpecCategory();
        category = CombineSpecCategories(
          category,
          apiCont.api.getCategory() ?? new SpecCategory()
        );

        const apiObj = api.toObject();
        const apiContApi = apiCont.api.toObject();

        api = new Api();
        api.setEnabled(apiObj.enabled && apiContApi.enabled);
        api.setName(apiObj.name + SEP + apiContApi.name);
        api.setComputeUnits(apiObj.computeUnits + apiContApi.computeUnits);
        api.setExtraComputeUnits(
          apiObj.extraComputeUnits + apiContApi.extraComputeUnits
        );
        api.setCategory(category);
        const blockParser = new BlockParser();
        blockParser.setParserArgList([]);
        blockParser.setParserFunc(PARSER_FUNC.EMPTY);
        blockParser.setDefaultValue("");
        blockParser.setEncoding("");
        api.setBlockParsing(blockParser);

        [latestRequestedBlock, earliestRequestedBlock] =
          compareRequestedBlockInBatch(
            latestRequestedBlock,
            requestedBlockForMessage
          );
      }
    }

    if (!api || !apiCollection) {
      throw Logger.fatal(
        "Invalid parsing. Api and ApiCollection is not defined"
      );
    }

    const batchMsg = newBatchMessage(jsonrpcMsgs);
    if (batchMsg instanceof Error) {
      throw Logger.fatal("Error creating batch message", batchMsg);
    }

    // TODO: add extension parsing.
    return new ParsedMessage(
      api,
      latestRequestedBlock,
      batchMsg,
      apiCollection,
      generateBatchRPCData(batchMsg)
    );
  }

  parseSingleMessage(
    options: SendRelayOptions
  ): [ApiContainer, ApiCollection, number, JsonrpcMessage] {
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
      JsonRPCVersion,
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

    return [apiCont, apiCollection, requestedBlock, jsonrpcMessage];
  }
}
