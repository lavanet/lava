const { LavaSDK } = require("../../../../ecosystem/lava-sdk/bin/src/sdk/sdk");

async function main() {
    // Initialize Lava SDK
    const cosmosHub = await new LavaSDK({
        privateKey: process.env.PRIVATE_KEY,
        chainID: "LAV1",
        lavaChainId:"lava",
        pairingListConfig:"testutil/e2e/sdk/pairingList.json"
    });

    // Fetch chain id
    const result = await cosmosHub.sendRelay({
        method: "status",
        params: [],
    });

    // Parse response
    const parsedResponse = JSON.parse(result);

    const chainID = parsedResponse.result["node_info"].network;

    // Validate chainID
    if (chainID != "lava") {
        throw new Error(" ERR Chain ID is not equal to cosmoshub-4");
    }else{
        console.log("Success: Fetching Lava chain ID using tendermintrpc passed. Chain ID correctly matches 'lava'.");
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