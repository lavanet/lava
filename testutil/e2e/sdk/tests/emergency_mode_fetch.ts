import { LavaSDK } from "../../../../ecosystem/lava-sdk/bin/src/sdk/sdk"
import { delay } from "./common";

async function main() {
    // Initialize Lava SDK
    const lavaSDKTendermint = await LavaSDK.create({
        privateKey: process.env.PRIVATE_KEY,
        chainIds: "LAV1",
        lavaChainId: "lava-local-1",
        pairingListConfig: process.env.PAIRING_LIST,
        allowInsecureTransport: true,
        logLevel: "debug",
    }).catch((e) => {
        throw new Error(" ERR [emergency_mode_fetch] failed setting lava-sdk tendermint test");
    });

    await delay(40000);

    // Fetch chain id
    for (let i = 0; i < 60; i++) { // send relays synchronously
        try {
            const result = await lavaSDKTendermint.sendRelay({
                method: "status",
                params: [],
            });

            // Parse response
            const parsedResponse = result;

            const chainID = parsedResponse.result["node_info"].network;

            // Validate chainID
            if (chainID !== "lava-local-1") {
                throw new Error(" ERR [emergency_mode_fetch] Chain ID is not equal to lava");
            } else {
                console.log(i, "[emergency_mode_fetch] Success: Fetching Lava chain ID using tendermintrpc passed. Chain ID correctly matches 'lava'");
            }
        } catch (error) {
            throw new Error(` ERR ${i} [emergency_mode_fetch] failed sending relay tendermint test: ${error.message}`);
        }
    }
}

(async () => {
    try {
        await main();
        process.exit(0);
    } catch (error) {
        console.error(" ERR [emergency_mode_fetch] " + error.message);
        process.exit(1);
    }
})();