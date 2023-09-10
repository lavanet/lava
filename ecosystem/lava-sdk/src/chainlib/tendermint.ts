import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceTendermintRPC,
} from "../chainlib/base_chain_parser";

export class TendermintRpcChainParser extends BaseChainParser {
  constructor() {
    super();
    this.apiInterface = APIInterfaceTendermintRPC;
  }
  parseMsg(options: SendRelayOptions | SendRestRelayOptions): string {
    // TODO implement the parsemsg
    return "";
  }
}
