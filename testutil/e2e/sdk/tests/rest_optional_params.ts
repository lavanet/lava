import { LavaSDK } from "../../../../ecosystem/lava-sdk/bin/src/sdk/sdk";

async function main() {
    // Initialize Lava SDK
    const lavaSdkRest = await LavaSDK.create({
        privateKey: process.env.PRIVATE_KEY,
        chainIds: ["LAV1"],
        lavaChainId: "lava",
        pairingListConfig: process.env.PAIRING_LIST,
        allowInsecureTransport: true,
        logLevel: "debug",
    }).catch(e => {
        throw new Error(" ERR [rest_optional_params] failed initializing lava-sdk rest test");
    });

    // Fetch chain id
    let relayArray = [];
    for (let i = 0; i < 100; i++) { // send 100 relays asynchronously
        relayArray.push((async () => {
            // Fetch chain id
            const result = await lavaSdkRest.sendRelay({
                connectionType: "GET",
                url: "/cosmos/base/tendermint/v1beta1/blocks/5",
            }).catch(e => {
                throw new Error(` ERR ${i} [rest_optional_params] failed sending relay rest test`);
            });

            // Parse response
            const chainID = result["block"]["header"]["chain_id"];

            // Validate chainID
            if (chainID != "lava") {
                throw new Error(" ERR [rest_optional_params] Chain ID is not equal to lava");
            } else {
                console.log(i, "[rest_optional_params] Success: Fetching Lava chain ID using REST passed. Chain ID correctly matches 'lava'");
            } 
        })().catch(err => {throw err;}));
    }
    // wait for all relays to finish;
    await Promise.allSettled(relayArray);
}

(async () => {
    try {
        await main();
        process.exit(0);
    } catch (error) {
        console.error(" ERR [rest_optional_params] " + error.message);
        process.exit(1);
    }
})();