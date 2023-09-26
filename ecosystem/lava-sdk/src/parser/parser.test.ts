import { LATEST_BLOCK } from "../common/common";
import { Metadata } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import {
  BlockParser,
  PARSER_FUNC,
  PARSER_FUNCMap,
} from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";
import { EncodingBase64, EncodingHex } from "./consts";
import { Parser } from "./parser";
import { RPCInput } from "./rpcInput";

class RPCInputTest implements RPCInput {
  constructor(
    public Params: any = null,
    public Result: any = null,
    public Headers: Metadata[] = [],
    public ParseBlockFunc: (block: string) => number | Error = (
      block: string
    ) => Parser.ParseDefaultBlockParameter(block),
    public GetHeadersFunc: () => Metadata[] = () => []
  ) {}

  GetParams(): any {
    return this.Params;
  }

  GetResult(): Uint8Array {
    return this.Result;
  }

  ParseBlock(block: string): number | Error {
    if (this.ParseBlockFunc !== null) {
      return this.ParseBlockFunc(block);
    }
    return Parser.ParseDefaultBlockParameter(block);
  }

  GetHeaders(): Metadata[] {
    return this.Headers;
  }
}

class TestParser extends Parser {
  public static ParseResponseByEncoding(
    rawResult: Uint8Array,
    encoding: string
  ): string | Error {
    return this.parseResponseByEncoding(rawResult, encoding);
  }

  public static ParseArrayOfInterfaces(
    data: any[],
    propName: string,
    innerSeparator: string
  ): any[] | null {
    return this.parseArrayOfInterfaces(data, propName, innerSeparator);
  }

  public static BlockAnyToString(block: any): string {
    return this.blockAnyToString(block);
  }
}

describe("parser", () => {
  describe("ParseArrayOfInterfaces", () => {
    const tests = [
      {
        name: "Test with matching prop name",
        data: ["name:John Doe", "age:30", "gender:male"],
        propName: "name",
        sep: ":",
        expected: ["John Doe"],
      },
      {
        name: "Test with non-matching prop name",
        data: ["name:John Doe", "age:30", "gender:male"],
        propName: "address",
        sep: ":",
        expected: null,
      },
      {
        name: "Test with empty data array",
        data: [],
        propName: "name",
        sep: ":",
        expected: null,
      },
      {
        name: "Test with non-string value in data array",
        data: ["name:John Doe", 30, "gender:male"],
        propName: "name",
        sep: ":",
        expected: ["John Doe"],
      },
    ];

    tests.forEach((test) => {
      it(test.name, () => {
        const result = TestParser.ParseArrayOfInterfaces(
          test.data,
          test.propName,
          test.sep
        );
        expect(result).toEqual(test.expected);
      });
    });
  });

  describe("ParseResponseByEncoding", () => {
    const testData = [
      {
        bytes:
          "9291EDC036AE254F9A6E0237F0EF13C452E7F08722E8DBD68B2F34CC8132C91D",
        encoding: EncodingHex,
      },
      {
        bytes: "kpHtwDauJU+abgI38O8TxFLn8Ici6NvWiy80zIEyyR0=",
        encoding: EncodingBase64,
      },
    ];

    testData.forEach((data) => {
      it(`Test with encoding: ${data.encoding}`, () => {
        const result = TestParser.ParseResponseByEncoding(
          Buffer.from(data.bytes, "hex"),
          data.encoding
        );
        expect(result).toEqual(result);
      });
    });
  });

  describe("BlockInterfaceToString", () => {
    const testCases = [
      {
        name: '"NUMBER"',
        block: "100",
        expected: "100",
      },
      {
        name: "String(NUMBER)",
        block: String(56),
        expected: "56",
      },
      {
        name: "NUMBER",
        block: 6,
        expected: "6",
      },
      {
        name: "Number(NUMBER)",
        block: Number(7878),
        expected: "7878",
      },
      {
        name: "NUMBER.0",
        block: 34.0,
        expected: "34",
      },
      {
        name: "Not a number",
        block: new BlockParser(),
        expected: "",
      },
      {
        // TODO: Is this an expected behavior?
        name: "Just a string",
        block: "This is a string",
        expected: "This is a string",
      },
    ];

    testCases.forEach((testCase) => {
      it(`Test BlockAnyToString for: ${testCase.name}`, () => {
        const result = TestParser.BlockAnyToString(testCase.block);
        expect(result).toEqual(testCase.expected);
      });
    });
  });
});

describe("TestParseBlockFromParamsHappyFlow", () => {
  class TestCase {
    constructor(
      public Name: string,
      public Message: RPCInput,
      public BlockParser: BlockParser,
      public ExpectedBlock: number
    ) {}
  }

  const createBlockParser = (
    parserArgs: string[],
    parserFunc: PARSER_FUNCMap[keyof PARSER_FUNCMap]
  ): BlockParser => {
    const blockParser: BlockParser = new BlockParser();
    blockParser.setParserArgList(parserArgs);
    blockParser.setParserFunc(parserFunc);
    return blockParser;
  };

  const testCases: TestCase[] = [
    new TestCase(
      "DefaultParsing",
      new RPCInputTest(),
      createBlockParser(["latest"], PARSER_FUNC.DEFAULT),
      LATEST_BLOCK
    ),
    new TestCase(
      "ParseByArg",
      new RPCInputTest(["1"]),
      createBlockParser(["0"], PARSER_FUNC.PARSE_BY_ARG),
      1
    ),
    new TestCase(
      "ParseCanonical__any[]__Case",
      new RPCInputTest([{ block: 6 }, { block: 25 }]),
      createBlockParser(["1", "block"], PARSER_FUNC.PARSE_CANONICAL),
      25
    ),
    new TestCase(
      "ParseCanonical__object__Case",
      new RPCInputTest({ data: { block: 1234234 } }),
      createBlockParser(["0", "data", "block"], PARSER_FUNC.PARSE_CANONICAL),
      1234234
    ),
    new TestCase(
      "ParseDictionary__any[]__Case",
      new RPCInputTest(["block=1000"]),
      createBlockParser(["block", "="], PARSER_FUNC.PARSE_DICTIONARY),
      1000
    ),
    new TestCase(
      "ParseDictionary__object__Case",
      new RPCInputTest({ block: 6 }),
      createBlockParser(["block", "unnecessary"], PARSER_FUNC.PARSE_DICTIONARY),
      6
    ),
  ];

  for (const testCase of testCases) {
    it(testCase.Name, () => {
      const block = Parser.ParseBlockFromParams(
        testCase.Message,
        testCase.BlockParser
      );
      expect(block).toBe(testCase.ExpectedBlock);
    });
  }
});
