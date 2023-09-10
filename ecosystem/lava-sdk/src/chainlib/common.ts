import { JsonRpcChainParser } from "./jsonrpc";
import { RestChainParser } from "./rest";
import { TendermintRpcChainParser } from "./tendermint";
import {
  APIInterfaceJsonRPC,
  APIInterfaceRest,
  APIInterfaceTendermintRPC,
} from "./base_chain_parser";
import { Logger } from "../logger/logger";

export function getChainParser(apiInterface: string) {
  switch (apiInterface) {
    case APIInterfaceJsonRPC:
      return new JsonRpcChainParser();
    case APIInterfaceRest:
      return new RestChainParser();
    case APIInterfaceTendermintRPC:
      return new TendermintRpcChainParser();
    default:
      throw Logger.fatal(
        "Couldn't find api interface in getChainParser options",
        apiInterface
      );
  }
}
