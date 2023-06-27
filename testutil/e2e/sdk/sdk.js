const { LavaSDK } = require("@lavanet/lava-sdk");

async function main() {
    // Initialize Lava SDK
    const eth = await new LavaSDK({
        privateKey: process.env.PRIVATE_KEY.trim(),
        chainID: "ETH1",
        lavaChainId:"lava",
        pairingListConfig:"testutil/e2e/sdk/pairingList.json"
    });

    const result = await eth.sendRelay({
        method: "eth_chainId",
        params: [],
    });

    const parsedResponse = JSON.parse(result);

    const chainID = parsedResponse.result;
    console.log(chainID)
    if (chainID != "0x2") {
        throw new Error(" ERR Chain ID is not equal to 0x1");
    }
}

(async () => {
    try {
        await main();
    } catch (error) {
        console.error(error.message);
        process.exit(1);
    }
})();