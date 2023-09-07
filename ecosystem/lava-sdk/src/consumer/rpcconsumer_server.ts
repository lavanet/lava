import { Logger } from "../logger/logger";
import { Relayer } from "../relayer/relayer";
import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";
import {
  BaseChainParser,
  SendRelayOptions,
  SendRestRelayOptions,
} from "../chainlib/base_chain_parser";

export class RPCConsumerServer {
  private consumerSessionManager: ConsumerSessionManager;
  private chainParser: BaseChainParser;
  private geolocation: string;
  private relayer: Relayer;
  constructor(
    relayer: Relayer,
    consumerSessionManager: ConsumerSessionManager,
    chainParser: BaseChainParser,
    geolocation: string
  ) {
    this.consumerSessionManager = consumerSessionManager;
    this.geolocation = geolocation;
    this.chainParser = chainParser;
    this.relayer = relayer;
  }

  async sendRelay(options: SendRelayOptions | SendRestRelayOptions) {
    const craftMessage = this.chainParser.parseMsg(options);
    // this.consumerSessionManager.getSessions()

    //TODO after reply if resolved, parse to json
    // // Decode response
    // const dec = new TextDecoder();
    // const decodedResponse = dec.decode(response.getData_asU8());

    // // Parse response
    // const jsonResponse = JSON.parse(decodedResponse);

    // // Return response
    // return jsonResponse;
  }
}
