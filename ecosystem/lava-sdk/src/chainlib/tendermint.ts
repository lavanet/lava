import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
  APIInterfaceTendermintRPC,
  ChainMessage,
} from "../chainlib/base_chain_parser";

export class TendermintRpcChainParser extends BaseChainParser {
  constructor() {
    super();
    this.apiInterface = APIInterfaceTendermintRPC;
  }
  parseMsg(options: SendRelayOptions | SendRestRelayOptions): ChainMessage {
    // TODO implement the parsemsg
    return "";
  }
}
