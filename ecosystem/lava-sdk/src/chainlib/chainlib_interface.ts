import { BaseChainParser } from "../chainlib/base_chain_parser";
import { ConsumerSessionManager } from "../lavasession/consumerSessionManager";
import { ConsumerSessionsWithProvider } from "../lavasession/consumerTypes";
import { Logger } from "../logger/logger";
import { Relayer } from "../relayer/relayer";

export abstract class ChainParser {
  protected consumerSessionManager: ConsumerSessionManager | undefined;
  protected baseChainParser: BaseChainParser | undefined;
  protected apiInterface: string;
  protected relayer: Relayer;
  protected constructor(apiInterface: string, relayer: Relayer) {
    this.apiInterface = apiInterface;
    this.relayer = relayer;
  }
  public setBaseChainParser(baseChainParser: BaseChainParser) {
    this.baseChainParser = baseChainParser;
  }

  public setConsumerSessionManager(
    consumerSessionManager: ConsumerSessionManager
  ) {
    this.consumerSessionManager = consumerSessionManager;
  }

  public updateAllProviders(
    epoch: number,
    pairing: ConsumerSessionsWithProvider[]
  ) {
    return (
      this.consumerSessionManager?.updateAllProviders(epoch, pairing) ??
      Logger.fatal(
        "Consumer session manager is not set for chain Parser:",
        this.apiInterface
      )
    );
  }

  abstract parseMsg(): string;
  abstract sendRelay(
    relayOptions: SendRelayOptions | SendRestRelayOptions
  ): Promise<string>;
}

/**
 * Options for sending RPC relay.
 */
export interface SendRelayOptions {
  method: string; // Required: The RPC method to be called
  params: Array<any>; // Required: An array of parameters to be passed to the RPC method
  chainId?: string; // Optional: the chain id to send the request to, if only one chain is initialized it will be chosen by default
}

/**
 * Options for sending Rest relay.
 */
export interface SendRestRelayOptions {
  connectionType: string; // Required: The HTTP method to be used (e.g., "GET", "POST")
  url: string; // Required: The API Path (URL) (e.g Cosmos: "/cosmos/base/tendermint/v1beta1/blocks/latest", Aptos: "/transactions" )
  // eslint-disable-next-line
    data?: Record<string, any>; // Optional: An object containing data to be sent in the request body (applicable for methods like "POST" and "PUT")
  chainId?: string; // Optional: the chain id to send the request to, if only one chain is initialized it will be chosen by default
}
