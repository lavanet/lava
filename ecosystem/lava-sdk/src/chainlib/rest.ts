import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceRest,
  ChainMessage,
  HeadersPassSend,
} from "../chainlib/base_chain_parser";
import { Logger } from "../logger/logger";
import { HttpMethod, NOT_APPLICABLE } from "../common/common";
import { Parser } from "../parser/parser";
import { FUNCTION_TAG } from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";
import { RestMessage } from "./chainproxy/rpcInterfaceMessages/rest_message";
import { ParsedMessage } from "./chain_message";
export class RestChainParser extends BaseChainParser {
  constructor() {
    super();
    this.apiInterface = APIInterfaceRest;
  }
  parseMsg(options: SendRelayOptions | SendRestRelayOptions): ChainMessage {
    if (!this.isRest(options)) {
      throw Logger.fatal(
        "Wrong relay options provided, expected SendRestRelayOptions got SendRelayOptions"
      );
    }

    const [apiCont, found] = this.matchSpecApiByName(
      options.url,
      options.connectionType
    );
    if (!found || !apiCont) {
      throw Logger.fatal("Rest api not supported", options.url);
    }

    if (!apiCont.api.getEnabled()) {
      throw Logger.fatal("API is disabled in spec", options.url);
    }

    // TODO: implement block parser
    const apiCollection = this.getApiCollection({
      addon: apiCont.collectionKey.addon,
      connectionType: options.connectionType,
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

    let restMessage = new RestMessage();
    restMessage.initBaseMessage({
      headers: headerHandler.filteredHeaders,
      latestBlockHeaderSetter: undefined,
    });

    restMessage.initRestMessage(
      options.data,
      options.url,
      apiCont.api.getName()
    );

    if (options.connectionType === HttpMethod.GET) {
      restMessage = new RestMessage();
      restMessage.initBaseMessage({
        headers: headerHandler.filteredHeaders,
        latestBlockHeaderSetter: settingHeaderDirective,
      });
      restMessage.initRestMessage(
        undefined,
        options.url + String(options.data),
        apiCont.api.getName()
      );
    }

    let requestedBlock: number | Error | null;

    const overwriteRequestedBlock = headerHandler.overwriteRequestedBlock;
    if (overwriteRequestedBlock == "") {
      const blockParser = apiCont.api.getBlockParsing();
      if (!blockParser) {
        throw Logger.fatal("BlockParsing is missing");
      }
      requestedBlock = Parser.ParseBlockFromParams(restMessage, blockParser);

      if (!requestedBlock) {
        throw Logger.fatal(
          `ParseBlockFromParams failed parsing block for chain: ${this.spec?.getName()}, blockParsing: ${blockParser}`
        );
      }
    } else {
      requestedBlock = restMessage.parseBlock(overwriteRequestedBlock);
      if (requestedBlock instanceof Error) {
        throw Logger.fatal(
          `Failed parsing block from an overwrite header for chain: ${this.spec?.getName()}, overwriteRequestedBlock: ${overwriteRequestedBlock}`
        );
      }
    }

    let data = "";
    if (options.data) {
      data = "?";
      for (const key in options.data) {
        data = data + key + "=" + options.data[key] + "&";
      }
    }

    const parsedMessage = new ParsedMessage(
      apiCont.api,
      requestedBlock,
      undefined,
      restMessage,
      apiCollection,
      undefined
    );

    // TODO: add extension parsing.

    // TODO: Change to parsedMessage when ready
    // return parsedMessage;
    return new ChainMessage(
      NOT_APPLICABLE,
      apiCont.api,
      apiCollection,
      data,
      options.url
    );
  }
}
