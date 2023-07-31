const { LavaSDK } = require("../../../../ecosystem/lava-sdk/bin/src/sdk/sdk");

async function main() {
    // Initialize Lava SDK
    const eth = await new LavaSDK({
        privateKey: process.env.PRIVATE_KEY,
        chainID: "LAV1",
        lavaChainId:"lava",
        rpcInterface:"rest",
        pairingListConfig:process.env.PAIRING_LIST
    });

    // Fetch chain id
    const result = await eth.sendRelay({
        method: "GET",
        url: "/node_info",
    });

    // Parse response
    const parsedResponse = JSON.parse(result);

    const chainID = parsedResponse["node_info"].network;

    // Validate chainID
    if (chainID != "lava") {
        throw new Error(" ERR Chain ID is not equal to 0x1");
    }else{
        console.log("Success: Fetching Lava chain ID using REST passed. Chain ID correctly matches 'lava'");
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