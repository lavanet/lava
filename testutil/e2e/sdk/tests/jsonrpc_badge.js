const { LavaSDK } = require("../../../../ecosystem/lava-sdk/bin/src/sdk/sdk");

async function main() {
    // Initialize Lava SDK
    const eth = await new LavaSDK({
        badge: {
            badgeServerAddress: process.env.BADGE_SERVER_ADDR,
            projectId: process.env.BADGE_PROJECT_ID,
        },
        chainID: "ETH1",
        lavaChainId:"lava",
        pairingListConfig:process.env.PAIRING_LIST, 
        allowInsecureTransport: true,
    }).catch(e => {
        throw new Error(" ERR failed initializing lava-sdk jsonrpc badge test");
    });

    // Fetch chain id
    const result = await eth.sendRelay({
        method: "eth_chainId",
        params: [],
    }).catch(e => {
        throw new Error(" ERR failed sending relay jsonrpc badge test");
    });

    // Parse response
    const parsedResponse = JSON.parse(result);

    const chainID = parsedResponse.result;

    // Validate chainID
    if (chainID != "0x1") {
        throw new Error(" ERR Chain ID is not equal to 0x1");
    } else{
        console.log("[jsonrpc_badge] Success: Fetching ETH chain ID using jsonrpc passed. Chain ID correctly matches '0x1'");
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