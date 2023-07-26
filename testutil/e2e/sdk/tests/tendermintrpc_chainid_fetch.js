const { LavaSDK } = require("../../../../ecosystem/lava-sdk/bin/src/sdk/sdk");

async function main() {
    // Initialize Lava SDK
    const cosmosHub = await new LavaSDK({
        privateKey: process.env.PRIVATE_KEY.trim(),
        chainID: "LAV1",
        lavaChainId:"lava",
        pairingListConfig:"testutil/e2e/sdk/pairingList.json"
    });

    const result = await cosmosHub.sendRelay({
        method: "status",
        params: [],
    });

    const parsedResponse = JSON.parse(result);

    const chainID = parsedResponse.result["node_info"].network;
    console.log(chainID)
    if (chainID != "lava") {
        throw new Error(" ERR Chain ID is not equal to cosmoshub-4");
    }
}

(async () => {
    try {
        await main();
    } catch (error) {
        console.error(" ERR "+error.message);
        process.exit(1);
    }
})();