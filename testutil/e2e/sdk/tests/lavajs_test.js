const { lavajs } = require("../../../../ecosystem/lavajs/dist/codegen");

async function main() {
    const client = await lavajs.lavanet.ClientFactory.createRPCQueryClient({ rpcEndpoint: "http://127.0.0.1:26657" })
    const lavaClient = client.lavanet.lava;
    let res3 = await lavaClient.spec.spec({ ChainID: "LAV1" })
    const cosmosClient = client.cosmos;
    console.log(res3)
    
    // get subscription info (lava query)
    let res = await lavaClient.subscription.current({ consumer: process.env.PUBLIC_KEY })
    console.log(res)
    
    // get a balance (cosmos sdk query)
    let res2 = await cosmosClient.bank.v1beta1.allBalances({ address: process.env.PUBLIC_KEY })
    console.log(res2)

    // Validate chainID
    if (chainID != "lava") {
        throw new Error(" ERR Chain ID is not equal to lava");
    }else{
        console.log("Success: Fetching Lava chain ID using tendermintrpc passed. Chain ID correctly matches 'lava'");
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