import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceRest,
} from "../chainlib/base_chain_parser";
export class RestChainParser extends BaseChainParser {
  constructor() {
    super();
    this.apiInterface = APIInterfaceRest;
  }
  parseMsg(options: SendRelayOptions | SendRestRelayOptions): string {
    // TODO implement the parsemsg
    return "";
  }
}
