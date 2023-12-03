import { LavaSDK } from "../../../../ecosystem/lava-sdk/bin/src/sdk/sdk"

function delay(ms: number) {
    return new Promise( resolve => setTimeout(resolve, ms) );
}

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

    await delay(45000);

    // Fetch chain id
    for (let i = 0; i < 80; i++) { // send relays synchronously
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
                throw new Error(" ERR [emergency_mode_badge] Chain ID is not equal to lava");
            } else {
                console.log(i, "[emergency_mode_badge] Success: Fetching Lava chain ID using tendermintrpc passed. Chain ID correctly matches 'lava'");
            }
        } catch (error) {
            throw new Error(` ERR ${i} [emergency_mode_badge] failed sending relay tendermint test: ${error.message}`);
        }
    }
}

(async () => {
    try {
        await main();
        process.exit(0);
    } catch (error) {
        console.error(" ERR [emergency_mode_badge] " + error.message);
        process.exit(1);
    }
})();