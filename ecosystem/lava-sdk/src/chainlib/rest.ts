import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceRest,
  ChainMessage,
} from "../chainlib/base_chain_parser";
export class RestChainParser extends BaseChainParser {
  constructor() {
    super();
    this.apiInterface = APIInterfaceRest;
  }
  parseMsg(options: SendRelayOptions | SendRestRelayOptions): ChainMessage {
    // TODO implement the parsemsg
    return "";
  }
}
