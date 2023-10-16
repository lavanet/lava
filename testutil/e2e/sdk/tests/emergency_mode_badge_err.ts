const { LavaSDK } = require("../../../../ecosystem/lava-sdk/bin/src/sdk/sdk");

async function main() {
    // Initialize Lava SDK
    const lavaSDKTendermint = await LavaSDK.create({
        badge: {
            badgeServerAddress: process.env.BADGE_SERVER_ADDR,
            projectId: process.env.BADGE_PROJECT_ID,
        },
        chainIds: ["LAV1"],
        lavaChainId:"lava",
        pairingListConfig:process.env.PAIRING_LIST,
        allowInsecureTransport: true,
        logLevel: "debug",
    }).catch(e => {
        throw new Error(" ERR failed initializing lava-sdk jsonrpc badge test");
    });

    // Fetch chain id
    for (let i = 0; i < 200; i++) { // send relays synchronously
        try {
            const result = await lavaSDKTendermint.sendRelay({
                method: "status",
                params: [],
            });

            // Parse response
            const parsedResponse = result;

            const chainID = parsedResponse.result["node_info"].network;

            // Validate chainID
            if (chainID !== "lava") {
                throw new Error(" ERR [tendermintrpc_chainid_fetch_badge] Chain ID is not equal to lava");
            } else {
                console.log(i, "[tendermintrpc_chainid_fetch_badge_1] Success: Fetching Lava chain ID using tendermintrpc passed. Chain ID correctly matches 'lava'");
            }
        } catch (error) {
            throw new Error(` ERR ${i} [tendermintrpc_chainid_fetch_badge] failed sending relay tendermint test: ${error.message}`);
        }
    }
}

(async () => {
    try {
        await main();
        process.exit(0);
    } catch (error) {
        //console.error(" ERR [tendermintrpc_chainid_fetch_badge] " + error.message);
        process.exit(1);
    }
})();