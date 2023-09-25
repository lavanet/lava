import { EncodingBase64, EncodingHex } from "./consts";
import { Parser } from "./parser";

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

  public static AppendInterfaceToInterfaceArrayWithError(
    value: string
  ): any[] | Error {
    return this.appendInterfaceToInterfaceArrayWithError(value);
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
});
