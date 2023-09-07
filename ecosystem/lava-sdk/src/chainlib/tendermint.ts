import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
} from "../chainlib/base_chain_parser";

export class TendermintRpcChainParser extends BaseChainParser {
  constructor() {
    super();
  }
  parseMsg(options: SendRelayOptions | SendRestRelayOptions): string {
    // TODO implement the parsemsg
    return "";
  }
}
