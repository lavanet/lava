import { LavaSDK } from "../../../../ecosystem/lava-sdk/bin/src/sdk/sdk";

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
        throw new Error(" ERR [tendermintrpc_named_params] failed setting lava-sdk tendermint test");
    });

    const blockHeight = 1

    const block = await lavaSDKTendermint.sendRelay({
        method: "block",
        params: { height: blockHeight },
    }).catch(e => {
        throw new Error(` ERR [tendermintrpc_named_params] failed fetching block using tendermint named params`);
    });

    // Validate block number
    if (block.result.block.header.height != blockHeight) {
        throw new Error(" ERR [tendermintrpc_named_params] Block number is not equal to the fetched block");
    }else{
        console.log("[tendermintrpc_named_params] Success: Fetching Lava block using tendermintrpc named params passed");
    }
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