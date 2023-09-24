import { LavaSDK } from "../../../../ecosystem/lava-sdk/bin/src/sdk/sdk";

async function main() {
    // Initialize Lava SDK
    const lavaSdkRest = await LavaSDK.create({
        privateKey: process.env.PRIVATE_KEY,
        chainIds: ["LAV1"],
        lavaChainId:"lava",
        pairingListConfig:process.env.PAIRING_LIST,
        allowInsecureTransport: true,
        logLevel: "debug",
    }).catch(e => {
        throw new Error(" ERR [rest_chainId_fetch] failed initializing lava-sdk rest test");
    });

    // Fetch chain id
    let relayArray = [];
    for (let i = 0; i < 100; i++) { // send 100 relays asynchronously
        relayArray.push((async () => {
            // Fetch chain id
            const result = await lavaSdkRest.sendRelay({
                connectionType: "GET",
                url: "/cosmos/base/tendermint/v1beta1/node_info",
            }).catch(e => {
                throw new Error(` ERR ${i} [rest_chainId_fetch] failed sending relay rest test`);
            });

            // Parse response
            const parsedResponse = result;

            const chainID = parsedResponse["default_node_info"].network;

            // Validate chainID
            if (chainID != "lava") {
                throw new Error(" ERR [rest_chainId_fetch] Chain ID is not equal to lava");
            }else{
                console.log(i, "[rest_chainId_fetch] Success: Fetching Lava chain ID using REST passed. Chain ID correctly matches 'lava'");
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
        console.error(" ERR [rest_chainId_fetch] "+error.message);
        process.exit(1);
    }
})();