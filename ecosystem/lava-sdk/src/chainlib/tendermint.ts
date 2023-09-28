import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceTendermintRPC,
  ChainMessage,
  HeadersPassSend,
} from "../chainlib/base_chain_parser";
import { Logger } from "../logger/logger";
import { encodeUtf8, generateRPCData } from "../util/common";
import { NOT_APPLICABLE } from "../common/common";
import { FUNCTION_TAG } from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";
import { TendermintrpcMessage } from "./chainproxy/rpcInterfaceMessages/tendermint_rpc_message";
import { Parser } from "../parser/parser";
import { ParsedMessage } from "./chain_message";

const Method = ""; // in tendermint all types are empty (in spec)
const jsonrpcVersion = "2.0";
export class TendermintRpcChainParser extends BaseChainParser {
  constructor() {
    super();
    this.apiInterface = APIInterfaceTendermintRPC;
  }
  parseMsg(options: SendRelayOptions | SendRestRelayOptions): ChainMessage {
    if (this.isRest(options)) {
      throw Logger.fatal(
        "Wrong relay options provided, expected SendRestRelayOptions got SendRelayOptions"
      );
    }

    const apiCont = this.getSupportedApi(options.method, Method);
    const apiCollection = this.getApiCollection({
      addon: apiCont.collectionKey.addon,
      connectionType: Method,
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

    const tendermintrpcMessage = new TendermintrpcMessage();
    tendermintrpcMessage.initJsonrpcMessage(
      jsonrpcVersion,
      encodeUtf8(
        String(
          options.id ?? Math.floor(Math.random() * Number.MAX_SAFE_INTEGER)
        )
      ),
      options.method,
      options.params
    );
    tendermintrpcMessage.initBaseMessage({
      headers: headerHandler.filteredHeaders,
      latestBlockHeaderSetter: settingHeaderDirective,
    });

    const blockParser = apiCont.api.getBlockParsing();
    if (!blockParser) {
      throw Logger.fatal("BlockParsing is missing");
    }

    let requestedBlock: number | Error | null;

    const overwriteRequestedBlock = headerHandler.overwriteRequestedBlock;
    if (overwriteRequestedBlock === "") {
      requestedBlock = Parser.parseBlockFromParams(
        tendermintrpcMessage,
        blockParser
      );
      if (!requestedBlock) {
        throw Logger.fatal(
          `ParseBlockFromParams failed parsing block for chain: ${this.spec?.getName()}, blockParsing: ${blockParser}`
        );
      }
    } else {
      requestedBlock = tendermintrpcMessage.parseBlock(overwriteRequestedBlock);
      if (requestedBlock instanceof Error) {
        throw Logger.fatal(
          `Failed parsing block from an overwrite header for chain: ${this.spec?.getName()}, overwriteRequestedBlock: ${overwriteRequestedBlock}`
        );
      }
    }

    const parsedMessage = new ParsedMessage(
      apiCont.api,
      requestedBlock,
      undefined,
      tendermintrpcMessage,
      apiCollection,
      undefined
    );

    // TODO: add extension parsing.

    // TODO: Change to parsedMessage when ready
    // return parsedMessage;
    const chainMessage = new ChainMessage(
      NOT_APPLICABLE,
      apiCont.api,
      apiCollection,
      generateRPCData(options.method, options.params),
      ""
    );
    return chainMessage;
  }
}
