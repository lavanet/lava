const { LavaSDK } = require("../../../../ecosystem/lava-sdk/bin/src/sdk/sdk");

async function main() {
    // Initialize Lava SDK
    const lavaSdkRest = await new LavaSDK({
        privateKey: process.env.PRIVATE_KEY,
        chainID: "LAV1",
        lavaChainId:"lava",
        rpcInterface:"rest",
        pairingListConfig:process.env.PAIRING_LIST,
        allowInsecureTransport: true,
    }).catch(e => {
        throw new Error(" ERR failed initializing lava-sdk rest test");
    });

    // Fetch chain id
    const result = await lavaSdkRest.sendRelay({
        method: "GET",
        url: "/cosmos/base/tendermint/v1beta1/node_info",
    }).catch(e => {
        throw new Error(" ERR failed sending relay rest test");
    });

    // Parse response
    const parsedResponse = JSON.parse(result);

    const chainID = parsedResponse["default_node_info"].network;

    // Validate chainID
    if (chainID != "lava") {
        throw new Error(" ERR Chain ID is not equal to lava");
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