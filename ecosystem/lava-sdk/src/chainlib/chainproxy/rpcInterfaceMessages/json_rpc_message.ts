import { Logger } from "../../../logger/logger";
import { Parser } from "../../../parser/parser";
import { RPCInput } from "../../../parser/rpcInput";
import { byteArrayToString, encodeUtf8 } from "../../../util/common";
import { BaseMessage } from "../common";

export interface JsonError {
  code: number;
  message: string;
  data: any;
}

export class JsonrpcMessage extends BaseMessage implements RPCInput {
  public version = "";
  public id: Uint8Array | undefined;
  public method = "";
  public params: any;
  public error: JsonError | undefined;
  public result: Uint8Array | undefined;

  initJsonrpcMessage(
    version: string,
    id: Uint8Array,
    method: string,
    params: any,
    error?: JsonError,
    result?: Uint8Array
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

  getResult(): Uint8Array {
    if (this.error) {
      Logger.warn(
        `GetResult() Request got an error from the node. error: ${this.error}`
      );
    }

    return this.result ?? new Uint8Array();
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
      encodeUtf8(rawJsonObj.id),
      rawJsonObj.method ?? "",
      rawJsonObj.params,
      undefined,
      encodeUtf8(JSON.stringify(rawJsonObj.result))
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
