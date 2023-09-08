import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceJsonRPC,
  ChainMessage,
  HeadersPassSend,
} from "../chainlib/base_chain_parser";
import { Logger } from "../logger/logger";
import { generateRPCData } from "../util/common";
import { NOT_APPLICABLE } from "../common/common";

const MethodPost = "POST";
export class JsonRpcChainParser extends BaseChainParser {
  constructor() {
    super();
    this.apiInterface = APIInterfaceJsonRPC;
  }
  parseMsg(options: SendRelayOptions | SendRestRelayOptions): ChainMessage {
    if (this.isRest(options)) {
      throw Logger.fatal(
        "Wrong relay options provided, expected SendRestRelayOptions got SendRelayOptions"
      );
    }
    const apiCont = this.getSupportedApi(options.method, MethodPost);
    const apiCollection = this.getApiCollection({
      addon: apiCont.collectionKey.addon,
      connectionType: MethodPost,
      internalPath: apiCont.collectionKey.internalPath,
    });
    const headerHandler = this.handleHeaders(
      options.metadata,
      apiCollection,
      HeadersPassSend
    );
    // TODO: implement apip.GetParsingByTag to support headers
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
