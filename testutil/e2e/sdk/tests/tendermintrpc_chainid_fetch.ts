const { LavaSDK } = require("../../../../ecosystem/lava-sdk/bin/src/sdk/sdk");

async function main() {
    // Initialize Lava SDK
    const lavaSDKTendermint = await LavaSDK.create({
        privateKey: process.env.PRIVATE_KEY,
        chainIds: "LAV1",
        lavaChainId:"lava",
        pairingListConfig:process.env.PAIRING_LIST, 
        allowInsecureTransport: true,
        logLevel: "debug",
    }).catch(e => {
        throw new Error(" ERR [tendermintrpc_chainid_fetch] failed setting lava-sdk tendermint test");
    });

    // Fetch chain id
    let relayArray = [];
    for (let i = 0; i < 100; i++) { // send 100 relays asynchronously
        relayArray.push((async () => {
            const result = await lavaSDKTendermint.sendRelay({
                method: "status",
                params: [],
            }).catch(e => {
                throw new Error(` ERR ${i} [tendermintrpc_chainid_fetch] failed sending relay tendermint test`);
            });
        
            // Parse response
            const parsedResponse = result;
        
            const chainID = parsedResponse.result["node_info"].network;
        
            // Validate chainID
            if (chainID != "lava") {
                throw new Error(" ERR [tendermintrpc_chainid_fetch] Chain ID is not equal to lava");
            }else{
                console.log(i, "[tendermintrpc_chainid_fetch] Success: Fetching Lava chain ID using tendermintrpc passed. Chain ID correctly matches 'lava'");
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
        console.error(" ERR [tendermintrpc_chainid_fetch] "+error.message);
        process.exit(1);
    }
})();