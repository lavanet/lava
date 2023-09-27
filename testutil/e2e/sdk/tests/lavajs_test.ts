import { lavanet } from "../../../../ecosystem/lavajs/dist/codegen/lavanet/bundle";

async function main() {
    const client = await lavanet.ClientFactory.createRPCQueryClient({ rpcEndpoint: "http://127.0.0.1:26657" })
    const lavaClient = client.lavanet.lava;
    let specResult = await lavaClient.spec.spec({ ChainID: "LAV1" })
    const cosmosClient = client.cosmos;
    if (specResult.Spec.index != "LAV1") {
        console.log(specResult)
        throw new Error("Failed validating Lava spec.")
    }
    console.log("[lavajs_test] Success: Fetching LAV1 spec")

    console.log("[lavajs_test] sending queries for address", process.env.PUBLIC_KEY)

    // get subscription info (lava query)
    let subscriptionResult = await lavaClient.subscription.current({ consumer: process.env.PUBLIC_KEY })
    
    if (subscriptionResult.sub.creator != process.env.PUBLIC_KEY) {
        console.log(subscriptionResult, "VS", process.env.PUBLIC_KEY)
        throw new Error("Failed validating subscription creator.")
    }
    console.log("[lavajs_test] Success: Fetching consumer's subscription")
    
    // get a balance (cosmos sdk query)
    let balanceResult = await cosmosClient.bank.v1beta1.allBalances({ address: process.env.PUBLIC_KEY, pagination: undefined })
    if (balanceResult.balances[0].denom != "ulava") {
        console.log(balanceResult, "VS", "required denom 'ulava'")
        throw new Error("Failed validating allBalances")
    }
    console.log("[lavajs_test] Success: Fetching consumer's balance")
}

(async () => {
    try {
        await main();
    } catch (error) {
        console.error(" ERR [lavajs_test] "+error.message);
        process.exit(1);
    }
})();