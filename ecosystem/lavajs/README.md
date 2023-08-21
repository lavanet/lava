# lava

<p align="center">
  <img src="https://user-images.githubusercontent.com/2770565/223762290-44afc792-8ad4-4dbb-b2c2-532780d6c5de.png" alt="Logo" width="80" height="80"><br>
   <h3 align="center">Lava JS </h3>
</p>


## install

```sh
npm install @lavanet/lavajs
```
## Table of contents

- [lava](#lava)
  - [Install](#install)
  - [Table of contents](#table-of-contents)
- [Usage](#usage)
    - [Full Flow Example](#full-flow-example)
    - [RPC Clients](#rpc-clients)           
- [Wallets and Signers](#connecting-with-wallets-and-signing-messages)
    - [Stargate Client](#initializing-the-stargate-client)
    - [Creating Signers](#creating-signers)
    - [Broadcasting Messages](#broadcasting-messages)
- [Advanced Usage](#advanced-usage)
- [Developing](#developing)
- [Credits](#credits)

## Usage

### Full Flow Example
```js
import * as lavajs from '@lavanet/lavajs';
import { Secp256k1Wallet } from '@cosmjs/amino'
import { fromHex } from "@cosmjs/encoding";
import Long from 'long';

const publicRpc = "<RPC-HERE>"
const privKey = "<PRIV-KEY-HERE>"

// interacting specifically with lava
async function run() {
  // create a lava client factory  
  const client = await lavajs.lavanet.ClientFactory.createRPCQueryClient({ rpcEndpoint: publicRpc})
  const lavaClient = client.lavanet.lava;
  const cosmosClient = client.cosmos;
  
  // create a wallet from a private key
  let wallet = await Secp256k1Wallet.fromKey(fromHex(privKey), "lava@")
  const [firstAccount] = await wallet.getAccounts();
  console.log(firstAccount)
  let signingClient = await lavajs.getSigningLavanetClient({
    rpcEndpoint: publicRpc,
    signer: wallet,
  })

  // get subscription info (lava query)
  let res = await lavaClient.subscription.current({ consumer: firstAccount.address })
  console.log(res)
  
  // get a balance (cosmos sdk query)
  let res2 = await cosmosClient.bank.v1beta1.allBalances({ address: firstAccount.address })
  console.log(res2)
  
  // buy a subscription (lava tx)
  const msg = lavajs.lavanet.lava.subscription.MessageComposer.withTypeUrl.buy({
    creator: firstAccount.address,
    consumer: firstAccount.address,
    index: "explorer",
    duration: new Long(1), /* in months */
  })

  const fee = {
    amount: [{ amount: "1", denom: "ulava" }], // Replace with the desired fee amount and token denomination
    gas: "50000000", // Replace with the desired gas limit
  }
  
  // careful here, uncommenting this code will launch a transaction. 
  //   await signingClient.signAndBroadcast(firstAccount.address,[msg], fee, "Buying subscription on Lava blockchain!")
}

run()

```

### RPC Clients

```js
import { lava } from '@lavanet/lavajs';

const { createRPCQueryClient } = lava.ClientFactory; 
const client = await createRPCQueryClient({ rpcEndpoint: RPC_ENDPOINT });

// now you can query the cosmos modules
const balance = await client.cosmos.bank.v1beta1
    .allBalances({ address: 'lava@1addresshere' });

// you can also query the lava modules
const balances = await client.lava.exchange.v1beta1
    .exchangeBalances()
```

## Connecting with Wallets and Signing Messages

⚡️ For web interfaces, we recommend using [cosmos-kit](https://github.com/cosmology-tech/cosmos-kit). Continue below to see how to manually construct signers and clients.

Here are the docs on [creating signers](https://github.com/cosmology-tech/cosmos-kit/tree/main/packages/react#signing-clients) in cosmos-kit that can be used with Keplr and other wallets.

### Initializing the Stargate Client

Use `getSigningLavanetClient` to get your `SigningStargateClient`, with the proto/amino messages full-loaded. No need to manually add amino types, just require and initialize the client:

```js
import { getSigningLavanetClient } from '@lavanet/lavajs';

const stargateClient = await getSigningLavanetClient({
  rpcEndpoint,
  signer // OfflineSigner
});
```
### Creating Signers

To broadcast messages, you can create signers with a variety of options:

* [cosmos-kit](https://github.com/cosmology-tech/cosmos-kit/tree/main/packages/react#signing-clients) (recommended)
* [keplr](https://docs.keplr.app/api/cosmjs.html)
* [cosmjs](https://gist.github.com/webmaster128/8444d42a7eceeda2544c8a59fbd7e1d9)
### Amino Signer

Likely you'll want to use the Amino, so unless you need proto, you should use this one:

```js
import { getOfflineSignerAmino as getOfflineSigner } from 'cosmjs-utils';
```
### Proto Signer

```js
import { getOfflineSignerProto as getOfflineSigner } from 'cosmjs-utils';
```

WARNING: NOT RECOMMENDED TO USE PLAIN-TEXT MNEMONICS. Please take care of your security and use best practices such as AES encryption and/or methods from 12factor applications.

```js
import { chains } from 'chain-registry';

const mnemonic =
  'unfold client turtle either pilot stock floor glow toward bullet car science';
  const chain = chains.find(({ chain_name }) => chain_name === 'lava');
  const signer = await getOfflineSigner({
    mnemonic,
    chain
  });
```
### Broadcasting Messages

Now that you have your `stargateClient`, you can broadcast messages:

```js
const { send } = cosmos.bank.v1beta1.MessageComposer.withTypeUrl;

const msg = send({
    amount: [
    {
        denom: 'coin',
        amount: '1000'
    }
    ],
    toAddress: address,
    fromAddress: address
});

const fee: StdFee = {
    amount: [
    {
        denom: 'coin',
        amount: '864'
    }
    ],
    gas: '86364'
};
const response = await stargateClient.signAndBroadcast(address, [msg], fee);
```

## Advanced Usage


If you want to manually construct a stargate client

```js
import { OfflineSigner, GeneratedType, Registry } from "@cosmjs/proto-signing";
import { AminoTypes, SigningStargateClient } from "@cosmjs/stargate";

import { 
    cosmosAminoConverters,
    cosmosProtoRegistry,
    cosmwasmAminoConverters,
    cosmwasmProtoRegistry,
    ibcProtoRegistry,
    ibcAminoConverters,
    lavaAminoConverters,
    lavaProtoRegistry
} from '@lavanet/lavajs';

const signer: OfflineSigner = /* create your signer (see above)  */
const rpcEndpint = 'https://rpc.cosmos.directory/lava'; // or another URL

const protoRegistry: ReadonlyArray<[string, GeneratedType]> = [
    ...cosmosProtoRegistry,
    ...cosmwasmProtoRegistry,
    ...ibcProtoRegistry,
    ...lavaProtoRegistry
];

const aminoConverters = {
    ...cosmosAminoConverters,
    ...cosmwasmAminoConverters,
    ...ibcAminoConverters,
    ...lavaAminoConverters
};

const registry = new Registry(protoRegistry);
const aminoTypes = new AminoTypes(aminoConverters);

const stargateClient = await SigningStargateClient.connectWithSigner(rpcEndpoint, signer, {
    registry,
    aminoTypes
});
```

## Developing

When first cloning the repo:

```
yarn
yarn build
```

### Codegen

Contract schemas live in `./contracts`, and protos in `./proto`. Look inside of `scripts/codegen.js` and configure the settings for bundling your SDK and contracts into `lava`:

```
yarn codegen
```

### Publishing

Build the types and then publish:

```
yarn build:ts
yarn publish
```
## Credits

🛠 Built by Cosmology — if you like our tools, please consider delegating to [our validator ⚛️](https://cosmology.tech/validator)

Code built with the help of these related projects:

* [@cosmwasm/ts-codegen](https://github.com/CosmWasm/ts-codegen) for generated CosmWasm contract Typescript classes
* [@osmonauts/telescope](https://github.com/osmosis-labs/telescope) a "babel for the Cosmos", Telescope is a TypeScript Transpiler for Cosmos Protobufs.
* [cosmos-kit](https://github.com/cosmology-tech/cosmos-kit) A wallet connector for the Cosmos ⚛️

## Disclaimer

AS DESCRIBED IN THE LICENSES, THE SOFTWARE IS PROVIDED “AS IS”, AT YOUR OWN RISK, AND WITHOUT WARRANTIES OF ANY KIND.

No developer or entity involved in creating this software will be liable for any claims or damages whatsoever associated with your use, inability to use, or your interaction with other users of the code or software using the code, including any direct, indirect, incidental, special, exemplary, punitive or consequential damages, or loss of profits, cryptocurrencies, tokens, or anything else of value.
