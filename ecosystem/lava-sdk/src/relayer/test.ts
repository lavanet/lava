import { RelayPrivateData } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import Relayer from "./relayer";

describe("Test relay request", () => {
  const getPrivateDataNegativeBlock = (): RelayPrivateData => {
    const requestPrivateData = new RelayPrivateData();
    requestPrivateData.setConnectionType("GET");
    requestPrivateData.setApiUrl("");
    requestPrivateData.setData(
      '{"jsonrpc":"2.0","id":535439332833,"method":"eth_chainId","params":[]}'
    );
    requestPrivateData.setRequestBlock(-2);
    requestPrivateData.setApiInterface("jsonrpc");
    requestPrivateData.setSalt(new Uint8Array());

    return requestPrivateData;
  };

  const getPrivateDataPositiveBlock = (): RelayPrivateData => {
    const requestPrivateData = new RelayPrivateData();
    requestPrivateData.setConnectionType("GET");
    requestPrivateData.setApiUrl("");
    requestPrivateData.setData(
      '{"jsonrpc":"2.0","id":535439332833,"method":"eth_chainId","params":[]}'
    );
    requestPrivateData.setRequestBlock(9876512);
    requestPrivateData.setApiInterface("jsonrpc");
    requestPrivateData.setSalt(new Uint8Array([1, 2, 3, 4, 5, 6]));

    return requestPrivateData;
  };
  it("Test generate content hash", () => {
    const testTable = [
      {
        input: getPrivateDataNegativeBlock(),
        expectedHash: new Uint8Array([
          15, 27, 36, 51, 198, 156, 51, 188, 180, 203, 63, 64, 56, 211, 74, 231,
          112, 26, 159, 166, 168, 3, 231, 34, 37, 88, 217, 245, 29, 203, 215,
          10,
        ]),
      },
      {
        input: getPrivateDataPositiveBlock(),
        expectedHash: new Uint8Array([
          56, 112, 127, 103, 197, 229, 30, 245, 181, 92, 121, 74, 199, 160, 149,
          235, 126, 73, 219, 228, 0, 91, 30, 161, 241, 219, 192, 97, 164, 108,
          91, 12,
        ]),
      },
    ];
    const relayer = new Relayer("", "", "", false);

    for (const testCase of testTable) {
      // Test case logic goes here
      const hash = relayer.calculateContentHashForRelayData(testCase.input);
      expect(hash).toEqual(testCase.expectedHash);
    }
  });
});
