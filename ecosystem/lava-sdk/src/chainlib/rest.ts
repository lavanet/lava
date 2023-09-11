import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceRest,
  ChainMessage,
  HeadersPassSend,
} from "../chainlib/base_chain_parser";
import { Logger } from "../logger/logger";
import { NOT_APPLICABLE } from "../common/common";
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
    // TODO: implement apip.GetParsingByTag to support headers
    let data = "";
    if (options.data) {
      data = "?";
      for (const key in options.data) {
        data = data + key + "=" + options.data[key] + "&";
      }
    }
    // TODO: add block parsing and use header overwrite.
    const chainMessage = new ChainMessage(
      NOT_APPLICABLE,
      apiCont.api,
      apiCollection,
      data,
      options.url
    );
    // TODO: add extension parsing.
    return chainMessage;
  }
}
