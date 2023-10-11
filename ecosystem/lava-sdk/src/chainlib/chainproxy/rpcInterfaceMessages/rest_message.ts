import { Parser } from "../../../parser/parser";
import { RPCInput } from "../../../parser/rpcInput";
import { BaseMessage } from "../common";

export class RestMessage extends BaseMessage implements RPCInput {
  public msg: Uint8Array | undefined;
  private path = "";
  private specPath = "";

  initRestMessage(msg: Uint8Array | undefined, path: string, specPath: string) {
    this.msg = msg;
    this.path = path;
    this.specPath = specPath;
  }

  // GetParams will be deprecated after we remove old client
  // Currently needed because of parser.RPCInput interface
  getParams(): any {
    let parsedMethod: string;
    const idx = this.path.indexOf("?");
    if (idx === -1) {
      parsedMethod = this.path;
    } else {
      parsedMethod = this.path.substring(0, idx);
    }

    const objectSpec = this.specPath.split("/");
    const objectPath = parsedMethod.split("/");

    const parameters: { [key: string]: any } = {};

    for (let index = 0; index < objectSpec.length; index++) {
      const element = objectSpec[index];
      if (element.includes("{")) {
        if (element.startsWith("{") && element.endsWith("}")) {
          parameters[element.slice(1, -1)] = objectPath[index];
        }
      }
    }
    if (idx > -1) {
      const queryParams = this.path.substring(idx);
      if (queryParams != undefined && queryParams.length > 0) {
        const queryParamsList = queryParams.split("&");
        for (const queryParamNameValue of queryParamsList) {
          const queryParamNameValueSplitted = queryParamNameValue.split("=");
          if (queryParamNameValueSplitted.length !== 2) {
            continue;
          }
          const queryParamName = queryParamNameValueSplitted[0];
          const queryParamValue = queryParamNameValueSplitted[1];
          parameters.set(queryParamName, queryParamValue);
        }
      }
    }
    if (Object.keys(parameters).length === 0) {
      return null;
    }
    return parameters;
  }

  // GetResult will be deprecated after we remove old client
  // Currently needed because of parser.RPCInput interface
  getResult(): string {
    return "";
  }

  parseBlock(block: string): number | Error {
    return Parser.parseDefaultBlockParameter(block);
  }

  updateLatestBlockInMessage(
    latestBlock: number,
    modifyContent: boolean
  ): boolean {
    return this.setLatestBlockWithHeader(latestBlock, modifyContent);
  }
}
