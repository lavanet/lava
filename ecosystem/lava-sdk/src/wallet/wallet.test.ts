import { createWallet } from "./wallet";
import ClientErrors from "./errors";

describe("Fetching account from private key", () => {
  it("Successfully fetch account", async () => {
    const privateKey =
      "885c3ebe355979d68d16f51e267040eb91e39021db07a9608ad881782d546009";
    const expectedAddress = "lava@194hjlf7swpm9c0rmktswt55p6xhhj6huzxnhaj";
    // Create lava wallet instance
    const lavaWallet = await createWallet(privateKey);

    // Expect no error
    expect(async () => {
      await lavaWallet.getConsumerAccount();
    }).not.toThrow(Error);

    // Fetch account
    const accountData = await lavaWallet.getConsumerAccount();

    // Check if account address match expected address
    expect(accountData.address).toBe(expectedAddress);
  });
  it("Invalid private key, can not create wallet", async () => {
    const privateKey = "";

    // Wallet was never initialized, expect error
    try {
      // Create lava wallet instance
      await createWallet(privateKey);
    } catch (err: unknown) {
      expect(err).toBe(ClientErrors.errInvalidPrivateKey);
    }
  });
});
