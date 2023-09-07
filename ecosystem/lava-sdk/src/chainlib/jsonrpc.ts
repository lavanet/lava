import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceJsonRPC,
  ChainMessage,
} from "../chainlib/base_chain_parser";

export class JsonRpcChainParser extends BaseChainParser {
  constructor() {
    super();
    this.apiInterface = APIInterfaceJsonRPC;
  }
  parseMsg(options: SendRelayOptions | SendRestRelayOptions): ChainMessage {
    // TODO implement the parsemsg
    return "";
  }
}
