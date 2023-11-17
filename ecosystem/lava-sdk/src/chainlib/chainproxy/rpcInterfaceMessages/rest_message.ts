import { DUMMY_URL } from "../../../common/common";
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
    const urlObj = new URL(this.path, DUMMY_URL);
    const parsedMethod = urlObj.pathname;
    const queryParams = urlObj.searchParams;

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

    queryParams.forEach((value, key) => {
      parameters[key] = value;
    });

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
