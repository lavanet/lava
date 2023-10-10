import { Logger } from "../../../logger/logger";
import { Parser } from "../../../parser/parser";
import { RPCInput } from "../../../parser/rpcInput";
import { byteArrayToString } from "../../../util/common";
import { BaseMessage } from "../common";

export interface JsonError {
  code: number;
  message: string;
  data: any;
}

export class JsonrpcMessage extends BaseMessage implements RPCInput {
  public version = "";
  public id: string | undefined;
  public method = "";
  public params: any;
  public error: JsonError | undefined;
  public result: string | undefined;

  initJsonrpcMessage(
    version: string,
    id: string,
    method: string,
    params: any,
    error?: JsonError,
    result?: string
  ) {
    this.version = version;
    this.id = id;
    this.method = method;
    this.params = params;
    this.error = error;
    this.result = result;
  }

  getParams() {
    return this.params;
  }

  getResult(): string {
    if (this.error) {
      Logger.warn(
        `GetResult() Request got an error from the node. error: ${this.error}`
      );
    }

    return this.result ?? "Failed getting result";
  }

  parseBlock(block: string): number | Error {
    return Parser.parseDefaultBlockParameter(block);
  }

  updateLatestBlockInMessage(
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    latestBlock: number,
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    modifyContent: boolean
  ): boolean {
    return false;
  }
}

interface RawJsonrpcMessage {
  jsonrpc?: string;
  id?: any;
  method?: string;
  params?: any;
  result?: any;
}

export function parseJsonRPCMsg(data: Uint8Array): JsonrpcMessage[] | Error {
  // Currently unused
  const msgs: JsonrpcMessage[] = [];
  let rawJsonObjs: RawJsonrpcMessage[];
  const dataAsString = byteArrayToString(data);
  try {
    const rawJsonObj = JSON.parse(dataAsString) as RawJsonrpcMessage;
    if (Array.isArray(rawJsonObj)) {
      rawJsonObjs = rawJsonObj;
    } else {
      rawJsonObjs = [rawJsonObj];
    }
  } catch (err) {
    return err as Error;
  }

  for (let index = 0; index < rawJsonObjs.length; index++) {
    const rawJsonObj = rawJsonObjs[index];
    const err = validateRawJsonrpcMessage(rawJsonObj);
    if (err) {
      return err;
    }

    const msg = new JsonrpcMessage();
    msg.initJsonrpcMessage(
      rawJsonObj.jsonrpc ?? "",
      rawJsonObj.id,
      rawJsonObj.method ?? "",
      rawJsonObj.params,
      undefined,
      JSON.stringify(rawJsonObj.result)
    );
    msgs.push(msg);
  }

  return msgs;
}

function validateRawJsonrpcMessage(
  rawJsonObj: RawJsonrpcMessage
): Error | undefined {
  if (!rawJsonObj.jsonrpc) {
    return new Error("Missing jsonrpc field from json");
  }
  if (!rawJsonObj.id) {
    return new Error("Missing id field from json");
  }
  if (!rawJsonObj.method) {
    return new Error("Missing method field from json");
  }
  if (!rawJsonObj.params) {
    return new Error("Missing params field from json");
  }
}
