import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceJsonRPC,
} from "../chainlib/base_chain_parser";

export class JsonRpcChainParser extends BaseChainParser {
  constructor() {
    super();
    this.apiInterface = APIInterfaceJsonRPC;
  }
  parseMsg(options: SendRelayOptions | SendRestRelayOptions): string {
    // TODO implement the parsemsg
    return "";
  }
}
