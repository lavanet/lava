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

    const parameters: any[] = [];

    for (let index = 0; index < objectSpec.length; index++) {
      const element = objectSpec[index];
      if (element.includes("{")) {
        parameters.push(objectPath[index]);
      }
    }

    if (parameters.length === 0) {
      return null;
    }
    return parameters;
  }

  // GetResult will be deprecated after we remove old client
  // Currently needed because of parser.RPCInput interface
  getResult(): Uint8Array {
    return new Uint8Array();
  }

  parseBlock(block: string): number | Error {
    return Parser.ParseDefaultBlockParameter(block);
  }

  updateLatestBlockInMessage(
    latestBlock: number,
    modifyContent: boolean
  ): boolean {
    return this.setLatestBlockWithHeader(latestBlock, modifyContent);
  }
}
