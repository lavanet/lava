import { JsonRpcChainParser } from "./jsonrpc";
import { RestChainParser } from "./rest";
import { TendermintRpcChainParser } from "./tendermint";
import {
  APIInterfaceJsonRPC,
  APIInterfaceRest,
  APIInterfaceTendermintRPC,
} from "./base_chain_parser";
import { Logger } from "../logger/logger";
import { Relayer } from "../relayer/relayer";

export function getChainParser(apiInterface: string, relayer: Relayer) {
  switch (apiInterface) {
    case APIInterfaceJsonRPC:
      return new JsonRpcChainParser(apiInterface, relayer);
    case APIInterfaceRest:
      return new RestChainParser(apiInterface, relayer);
    case APIInterfaceTendermintRPC:
      return new TendermintRpcChainParser(apiInterface, relayer);
    default:
      throw Logger.fatal(
        "Couldn't find api interface in getChainParser options",
        apiInterface
      );
  }
}
