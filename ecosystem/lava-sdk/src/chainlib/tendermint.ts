import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceTendermintRPC,
  ChainMessage,
  HeadersPassSend,
} from "../chainlib/base_chain_parser";
import { Logger } from "../logger/logger";
import { generateRPCData } from "../util/common";

const Method = ""; // in tendermint all types are empty (in spec)
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
    // TODO: implement apip.GetParsingByTag to support headers
    const chainMessage = new ChainMessage(
      -2,
      apiCont.api,
      apiCollection,
      generateRPCData(options.method, options.params),
      ""
    );
    return chainMessage;
  }
}
