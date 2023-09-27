import { RestMessage } from "./rest_message";

describe("RestMessage", () => {
  it("should return null params", () => {
    const restMessage = new RestMessage(
      undefined,
      "blocks/latest",
      "blocks/latest",
      {
        headers: [],
        latestBlockHeaderSetter: undefined,
      }
    );

    const params = restMessage.getParams();
    expect(params).toBeNull();
  });

  it("should return empty result", () => {
    const restMessage = new RestMessage(
      undefined,
      "blocks/latest",
      "blocks/latest",
      {
        headers: [],
        latestBlockHeaderSetter: undefined,
      }
    );

    const result = restMessage.getResult();
    expect(result.length).toBe(0);
  });
});

describe("RestParseBlock", () => {
  const testTable = [
    {
      name: "Default block param",
      input: "latest",
      expected: -2,
    },
    {
      name: "String representation of int64",
      input: "80",
      expected: 80,
    },
    {
      name: "Hex representation of int64",
      input: "0x26D",
      expected: 621,
    },
  ];

  for (const testCase of testTable) {
    it(`should parse ${testCase.name}`, () => {
      const restMessage = new RestMessage(undefined, "", "", {
        headers: [],
        latestBlockHeaderSetter: undefined,
      });

      const block = restMessage.parseBlock(testCase.input);
      expect(block).toEqual(testCase.expected);
    });
  }
});
