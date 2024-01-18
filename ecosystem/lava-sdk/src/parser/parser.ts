import {
  NOT_APPLICABLE,
  LATEST_BLOCK,
  EARLIEST_BLOCK,
  PENDING_BLOCK,
  SAFE_BLOCK,
  FINALIZED_BLOCK,
  DEFAULT_PARSED_RESULT_INDEX,
  IsString,
  IsNumber,
} from "../common/common";
import {
  BlockParser,
  PARSER_FUNC,
} from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";
import { encodeUtf8 } from "../util/common";
import {
  EncodingBase64,
  EncodingHex,
  PARSE_PARAMS,
  PARSE_RESULT,
} from "./consts";
import {
  InvalidBlockValue,
  InvalidInputFormat,
  UnsupportedBlockParser,
  ValueNotSetError,
} from "./errors";
import { RPCInput } from "./rpcInput";

export class Parser {
  public static parseDefaultBlockParameter(block: string): number | Error {
    switch (block) {
      case "latest":
        return LATEST_BLOCK;
      case "earliest":
        return EARLIEST_BLOCK;
      case "pending":
        return PENDING_BLOCK;
      case "safe":
        return SAFE_BLOCK;
      case "finalized":
        return FINALIZED_BLOCK;
      default:
        // try to parse a number
        const blockNum = parseInt(block, 0);
        if (isNaN(blockNum) || blockNum < 0) {
          return new InvalidBlockValue(block);
        }
        return blockNum;
    }
  }

  public static parseBlockFromParams(
    rpcInput: RPCInput,
    blockParser: BlockParser
  ): number | Error {
    const result = this.parse(rpcInput, blockParser, PARSE_PARAMS);
    if (result == null) {
      return NOT_APPLICABLE;
    }
    if (result instanceof Error) {
      return result;
    }

    const resString = (result as any[])[0] as string;
    return rpcInput.parseBlock(resString);
  }

  public static parseFromReply(
    rpcInput: RPCInput,
    blockParser: BlockParser
  ): string | Error | null {
    const result = this.parse(rpcInput, blockParser, PARSE_RESULT);
    if (result == null) {
      return null;
    }
    if (result instanceof Error) {
      return result;
    }

    let response = (result as any[])[DEFAULT_PARSED_RESULT_INDEX] as string;
    if (response.includes('"')) {
      response = JSON.parse(response);
    }

    return response;
  }

  public static parseBlockFromReply(
    rpcInput: RPCInput,
    blockParser: BlockParser
  ): number | Error {
    const result = this.parseFromReply(rpcInput, blockParser);
    if (result == null) {
      return NOT_APPLICABLE;
    }

    if (result instanceof Error) {
      return result;
    }

    return rpcInput.parseBlock(result);
  }

  public static parseFromReplyAndDecode(
    rpcInput: RPCInput,
    resultParser: BlockParser
  ): string | Error | null {
    const result = this.parseFromReply(rpcInput, resultParser);
    if (result == null) {
      return null;
    }

    if (result instanceof Error) {
      return result;
    }

    return this.parseResponseByEncoding(
      encodeUtf8(result),
      resultParser.getEncoding()
    );
  }

  public static parseDefault(input: string[]): any[] {
    return [input[0]];
  }

  public static getDataToParse(
    rpcInput: RPCInput,
    dataSource: number
  ): any | Error {
    switch (dataSource) {
      case PARSE_PARAMS:
        return rpcInput.getParams();
      case PARSE_RESULT:
        let data: Record<string, any>;
        const rawJsonString = rpcInput.getResult();
        if (rawJsonString.length === 0) {
          return new Error("GetDataToParse failure GetResult is empty");
        }
        // Try to unmarshal, and if the data is unmarshalable, then return the data itself
        try {
          data = JSON.parse(rawJsonString);
          return [data];
        } catch (err) {
          return [rawJsonString];
        }
      default:
        return new Error("unsupported block parser parserFunc");
    }
  }

  public static parseByArg(
    rpcInput: RPCInput,
    input: string[],
    dataSource: number
  ): any[] | Error {
    // specified block is one of the direct parameters, input should be one string defining the location of the block
    if (input.length !== 1) {
      return new InvalidInputFormat(`input length: ${input.length}`);
    }
    const inp = input[0];
    const param_index = parseInt(inp, 10);
    if (isNaN(param_index)) {
      return new InvalidInputFormat(`input isn't an unsigned index: ${inp}`);
    }

    const unmarshalledData = this.getDataToParse(rpcInput, dataSource);
    if (unmarshalledData instanceof Error) {
      return new InvalidInputFormat(
        `data is not json: ${unmarshalledData.message}`
      );
    }

    if (Array.isArray(unmarshalledData)) {
      if (param_index >= unmarshalledData.length) {
        return new ValueNotSetError();
      }
      const block = unmarshalledData[param_index];
      return [this.blockAnyToString(block)];
    } else {
      return new Error(
        "Parse type unsupported in parse by arg, only list parameters are currently supported"
      );
    }
  }

  // expect input to be keys[a,b,c] and a canonical object such as
  //
  //	{
  //	  "a": {
  //	      "b": {
  //	         "c": "wanted result"
  //	       }
  //	   }
  //	}
  //
  // should output an `any` array with "wanted result" in first index 0
  public static parseCanonical(
    rpcInput: RPCInput,
    input: string[],
    dataSource: number
  ): any[] | Error {
    const inp = input[0];
    const param_index = parseInt(inp, 10);
    if (isNaN(param_index)) {
      return new Error(
        `invalid input format, input isn't an unsigned index: ${inp}`
      );
    }

    let unmarshalledData = this.getDataToParse(rpcInput, dataSource);
    if (unmarshalledData instanceof Error) {
      return new Error(
        `invalid input format, data is not json: ${unmarshalledData.message}`
      );
    }

    if (Array.isArray(unmarshalledData)) {
      if (param_index >= unmarshalledData.length) {
        return new ValueNotSetError();
      }
      let blockContainer = unmarshalledData[param_index];
      for (let i = 1; i < input.length; i++) {
        if (typeof blockContainer === "object" && blockContainer !== null) {
          const key = input[i];
          if (blockContainer.hasOwnProperty(key)) {
            blockContainer = blockContainer[key];
          } else {
            return new Error(
              `invalid input format, blockContainer does not have field inside: ${key}`
            );
          }
        } else {
          return new Error(
            `invalid parser input format, blockContainer is ${typeof blockContainer} and not an object`
          );
        }
      }

      return [this.blockAnyToString(blockContainer)];
    } else if (
      unmarshalledData !== null &&
      typeof unmarshalledData === "object"
    ) {
      for (let i = 1; i < input.length; i++) {
        const key = input[i];
        if (unmarshalledData.hasOwnProperty(key)) {
          const val = unmarshalledData[key];
          if (i === input.length - 1) {
            return [this.blockAnyToString(val)];
          } else if (typeof val === "object" && val !== null) {
            unmarshalledData = val;
          } else {
            return new Error(`failed to parse, ${key} is not of type object`);
          }
        } else {
          return new ValueNotSetError();
        }
      }
    }

    return new Error(
      `ParseCanonical not supported with other types ${typeof unmarshalledData}`
    );
  }

  public static parseDictionary(
    rpcInput: RPCInput,
    input: string[],
    dataSource: number
  ): any[] | Error {
    // Validate number of arguments
    // The number of arguments should be 2
    // [prop_name,separator]
    if (input.length !== 2) {
      return new Error(
        `invalid input format, input length: ${input.length} and needs to be 2`
      );
    }

    const unmarshalledData = this.getDataToParse(rpcInput, dataSource);
    if (unmarshalledData instanceof Error) {
      return new Error(
        `invalid input format, data is not json: ${unmarshalledData.message}`
      );
    }

    const propName = input[0];
    const innerSeparator = input[1];

    if (Array.isArray(unmarshalledData)) {
      const value = this.parseArrayOfInterfaces(
        unmarshalledData,
        propName,
        innerSeparator
      );
      if (value !== null) {
        return value;
      }
      return new ValueNotSetError();
    } else if (
      unmarshalledData !== null &&
      typeof unmarshalledData === "object"
    ) {
      if (unmarshalledData.hasOwnProperty(propName)) {
        return this.appendInterfaceToInterfaceArrayWithError(
          this.blockAnyToString(unmarshalledData[propName])
        );
      }
      return new ValueNotSetError();
    }

    return new Error(
      `ParseDictionary not supported with other types: ${typeof unmarshalledData}`
    );
  }

  public static parseDictionaryOrOrdered(
    rpcInput: RPCInput,
    input: string[],
    dataSource: number
  ): any[] | Error {
    // Validate number of arguments
    // The number of arguments should be 3
    // [prop_name,separator,parameter order if not found]
    if (input.length !== 3) {
      return new Error(
        `ParseDictionaryOrOrdered: invalid input format, input length: ${
          input.length
        } and needs to be 3: ${input.join(",")}`
      );
    }

    const unmarshalledData = this.getDataToParse(rpcInput, dataSource);
    if (unmarshalledData instanceof Error) {
      return new Error(
        `invalid input format, data is not json: ${unmarshalledData.message}`
      );
    }

    const propName = input[0];
    const innerSeparator = input[1];
    const inp = input[2];

    // Convert prop index to the uint
    const propIndex = parseInt(inp, 10);
    if (isNaN(propIndex)) {
      return new Error(
        `invalid input format, input isn't an unsigned index: ${inp}`
      );
    }

    if (Array.isArray(unmarshalledData)) {
      const value = this.parseArrayOfInterfaces(
        unmarshalledData,
        propName,
        innerSeparator
      );
      if (value !== null) {
        return value;
      }
      if (propIndex >= unmarshalledData.length) {
        return new ValueNotSetError();
      }
      return this.appendInterfaceToInterfaceArrayWithError(
        this.blockAnyToString(unmarshalledData[propIndex])
      );
    } else if (
      unmarshalledData !== null &&
      typeof unmarshalledData === "object"
    ) {
      if (unmarshalledData.hasOwnProperty(propName)) {
        return this.appendInterfaceToInterfaceArrayWithError(
          this.blockAnyToString(unmarshalledData[propName])
        );
      }
      if (unmarshalledData.hasOwnProperty(inp)) {
        return this.appendInterfaceToInterfaceArrayWithError(
          this.blockAnyToString(unmarshalledData[inp])
        );
      }
      return new ValueNotSetError();
    }

    return new Error(
      `ParseDictionaryOrOrdered not supported with other types: ${typeof unmarshalledData}`
    );
  }

  protected static parseResponseByEncoding(
    rawResult: Uint8Array,
    encoding: string
  ): string | Error {
    switch (encoding) {
      case EncodingBase64:
        return Buffer.from(rawResult).toString();
      case EncodingHex:
        let hexString = Buffer.from(rawResult).toString("hex");
        if (hexString.length % 2 !== 0) {
          hexString = "0" + hexString;
        }

        try {
          const hexBytes = Buffer.from(hexString, "hex");
          const base64String = hexBytes.toString("base64");
          return base64String;
        } catch (error) {
          return new Error(
            `Tried decoding a hex response in parseResponseByEncoding but failed: ${error}`
          );
        }
      default:
        return Buffer.from(rawResult).toString();
    }
  }

  // parseArrayOfInterfaces returns value of item with specified prop name
  // If it doesn't exist return null
  protected static parseArrayOfInterfaces(
    data: any[],
    propName: string,
    innerSeparator: string
  ): any[] | null {
    for (const val of data) {
      if (IsString(val)) {
        const valueArr = val.split(innerSeparator, 2);

        if (valueArr[0] !== propName || valueArr.length < 2) {
          continue;
        } else {
          return [valueArr[1]];
        }
      }
    }

    return null;
  }

  // AppendInterfaceToInterfaceArrayWithError appends an `any` to an array of `any`s
  // Returns an error if the value is an empty string
  protected static appendInterfaceToInterfaceArrayWithError(
    value: string
  ): any[] | Error {
    if (value === "" || value === "0") {
      return new ValueNotSetError();
    }
    return [value];
  }

  protected static blockAnyToString(block: any): string {
    if (IsString(block)) {
      return block;
    } else if (IsNumber(block)) {
      if (Number.isInteger(block)) {
        // Check if it's an integer
        return block.toString();
      } else {
        // Assuming it's a floating-point number
        return block.toFixed();
      }
    } else {
      return String(block);
    }
  }

  protected static parse(
    rpcInput: RPCInput,
    blockParser: BlockParser,
    dataSource: number
  ): any[] | null | Error {
    let retval: any[] | Error = [];

    const parserFunc = blockParser.getParserFunc();
    switch (parserFunc) {
      case PARSER_FUNC.EMPTY:
        return null;
      case PARSER_FUNC.PARSE_BY_ARG:
        retval = this.parseByArg(
          rpcInput,
          blockParser.getParserArgList(),
          dataSource
        );
        break;
      case PARSER_FUNC.PARSE_CANONICAL:
        retval = this.parseCanonical(
          rpcInput,
          blockParser.getParserArgList(),
          dataSource
        );
        break;
      case PARSER_FUNC.PARSE_DICTIONARY:
        retval = this.parseDictionary(
          rpcInput,
          blockParser.getParserArgList(),
          dataSource
        );
        break;
      case PARSER_FUNC.PARSE_DICTIONARY_OR_ORDERED:
        retval = this.parseDictionaryOrOrdered(
          rpcInput,
          blockParser.getParserArgList(),
          dataSource
        );
        break;
      case PARSER_FUNC.DEFAULT:
        retval = this.parseDefault(blockParser.getParserArgList());
        break;
      default:
        return new UnsupportedBlockParser(parserFunc);
    }

    if (
      retval instanceof ValueNotSetError &&
      blockParser.getDefaultValue() !== ""
    ) {
      // means this parsing failed because the value did not exist on an optional param
      return [blockParser.getDefaultValue()];
    }

    return retval;
  }
}
