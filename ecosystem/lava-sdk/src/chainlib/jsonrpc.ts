import {
  ChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
} from "./chainlib_interface";
import { Relayer } from "../relayer/relayer";

export class JsonRpcChainParser extends ChainParser {
  constructor(apiInterface: string, relayer: Relayer) {
    super(apiInterface, relayer);
  }
  parseMsg(): string {
    // TODO implement the parsemsg
    return "";
  }
  async sendRelay(
    relayOptions: SendRelayOptions | SendRestRelayOptions
  ): Promise<string> {
    // TODO implement the send relay
    return "";
  }
}
