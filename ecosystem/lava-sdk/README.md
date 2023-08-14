<a name="readme-top"></a>

<!-- PROJECT LOGO -->
<br />
<div align="center">
  <img src="https://user-images.githubusercontent.com/2770565/223762290-44afc792-8ad4-4dbb-b2c2-532780d6c5de.png" alt="Logo" width="80" height="80">
  <h3 align="center">Lava SDK - <i>ALPHA</i></h3>
  </p>
</div>

<b>Access Web3 APIs, the Lava way 🌋</b>
    
JavaScript/TypeScript SDK reference implementation designed for developers looking for access through the Lava Network. It can be added to your app/dapp and run in browsers to provide multi-chain peer-to-peer access to blockchain APIs.

# Official Documentation
https://docs.lavanet.xyz/access-sdk

<!-- Roadmap -->
# Roadmap
The SDK is currently in the Alpha stage and is not production-ready for all usecases. 

Roadmap highlights:

1. ~Send Relays per Lava Pairing~ ✅
2. ~Find seed providers for the initial connection~ ✅
3. ~EtherJS Integration~ ✅
4. Ability to run in the browser without compromising keys
5. High throughput via session management
6. More libraries integrations (Cosmjs, web3.js...)
6. Other Lava consensus implementations (e.g. QoS, data reliability, ...)

<!-- Prerequisites -->

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- Installation -->

# Installation

### Important Version Control
 * Please check the current Lava node version before installing the lava-sdk. 
  
 * Make sure you are using the `"Latest"` tag. You can check the latest releases here: https://github.com/lavanet/lava/releases 

 * lava-sdk releases can be found here: https://github.com/lavanet/lava-sdk/releases or in the npm official site: https://www.npmjs.com/package/@lavanet/lava-sdk

### For Example
If lava latest release version is `v0.8.0` or any minor version such as v0.8.1 ➡️ sdk version will be `v0.8.0`

---

### Prerequisites (Alpha version)
_SDK setup requires additional steps at the moment, but we're working on minimizing prerequisites as we progress through the roadmap._

1. Create a wallet on the Lava Testnet, have LAVA tokens
1. Stake in the chain you want to access
2. Stake in Lava chain

Need help? We've got you covered 😻 Head over to our [Discord](https://discord.gg/5VcqgwMmkA) channel `#developers` and we'll provide testnet tokens and further support

### Yarn

```bash
yarn add @lavanet/lava-sdk
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- USAGE EXAMPLES -->
# Usage

A single instance of the SDK establishes a connection with a specific blockchain network using a single RPC interface. _Need multiple chains or use multiple RPC interfaces? Create multiple instances._ 

To use the SDK, you will first need to initialize it.

```typescript
const lavaSDK = await new LavaSDK({
  privateKey: privKey, // string (Only if Badge is not provided)
  badge: badgeOptions, // BadgeOptions (Only if privateKey is not provided)
  chainID: chainID, // string (Required)
  rpcInterface: rpcInterface, // string (Optional)
  pairingListConfig: localConfigPath, // string (Optional)
  network: network, // string (Optional)
  geolocation: geolocation, // string (Optional)
  secure: secureFlag, // boolean (Optional)
  allowInsecureTransport: insecureTransportFlag, // boolean (Optional)
  debug: debugFlag, // boolean (Optional)
});

const badgeOptions: BadgeOptions = {
    badgeServerAddress: serverAddress, // string (Required)
    projectId: projectIdValue, // string (Required)
    authentication: authValue, // string (Optional)
};
```
Lava SDK options: 

- `badge` parameter specifies the public URL of the badge server and the ID of the project you want to connect to. If you enable the badge, you should remove the `privateKey`

- `privateKey` parameter is required and should be the private key of the staked Lava client for the specified `chainID`

- `chainID` parameter is required and should be the ID of the chain you want to query. You can find all supported chains with their IDs [supportedChains](https://github.com/lavanet/lava-sdk/blob/main/supportedChains.json)

- `rpcInterface` is an optional parameter representing the interface that will be used for sending relays. For cosmos chains it can be `tendermintRPC` or `rest`. For evm compatible chains `jsonRPC` or `rest`. You can find the list of all default rpc interfaces [supportedChains](https://github.com/lavanet/lava-sdk/blob/main/supportedChains.json)

- `pairingListConfig` is an optional parameter that specifies the lava pairing list config used for communicating with lava network. Lava SDK does not rely on one centralized rpc for querying lava network. It uses a list of rpc providers to fetch list of the providers for specified `chainID` and `rpcInterface` from lava network. If not pairingListConfig set, the default list will be used [default lava pairing list](https://github.com/lavanet/lava-providers/blob/main/pairingList.json)

- `network` is an optional parameter that specifies the network from pairingListConfig which will be used. Default value is `testnet`

- `geolocation` is an optional parameter that specifies the geolocation which will be used. Default value is `1` which represents North America providers. Besides North America providers, lava supports EU providers on geolocation `2`

- `lavaChainId` is an optional parameter that specifies the chain id of the lava network. Default value is `lava-testnet-2` which represents Lava testnet

- `secure` is an optional parameter which indicates whether the SDK should communicate through HTTPS. This is a temporary flag that will be disabled once the chain uses HTTPS by default

- `allowInsecureTransport` is an optional parameter which indicates whether to use an insecure transport when connecting to the provider. This option is intended for testing purposes only and allows for the use of self-signed certificates

- `debug` is an optional parameter used for debugging the LavaSDK. When enabled, it mostly prints logs to speed up development

Badge options:

- `badgeServerAddress` is an optional parameter that specifies the public URL of the badge server

- `projectId` is an optional parameter that represents the ID of the project you want to connect to

- `authentication` is an optional parameter that specifies any additional authentication requirements


### Private keys vs Badge server

In the LavaSDK, users have the option to authenticate using either a private key or a badge server. Both methods have their advantages and specific use cases. Let's dive into the details of each:

#### Private Key:

The private key is the quickest way to start using the LavaSDK. All you need is the private key of an account that has staked both for the Lava Network and the chain you wish to query. With this, you can immediately begin utilizing the functionalities of the LavaSDK.

However, it's crucial to note that using a private key directly, especially in a browser environment, comes with security risks. If a user exposes their private key in a browser-based application, malicious actors can easily discover it and exploit it for their purposes. As such, direct usage of the private key should be limited to server-side operations or browser testing environments. It's not recommended for production-level applications, especially those that run client-side.

#### Badge Server:

The Badge server is a dedicated Go service that clients can initiate. It acts as an intermediary between the LavaSDK and the Lava Network, ensuring that no secrets or private keys are stored directly within the SDK. Instead, all sensitive information is securely held on the Badge server, with the LavaSDK only communicating with this server.

The primary advantage of using the Badge server is the enhanced security it offers. Additionally, the Badge server pre-updates some information like pairing list for current epoch, allowing the LavaSDK to fetch data more rapidly by merely pinging the server. This results in faster and more efficient operations. 

However, there are some challenges to consider. Users need to bootstrap and maintain the Badge server, which might require additional resources and expertise. For those interested in exploring the Badge server without setting up their own, we have deployed a Lava Test Badge server. You can access and experiment with it through our gateway application at [Lava Gateway](https://accounts.lavanet.xyz/)

For detailed instructions on how to start the Badge server, refer to our [documentation](https://github.com/lavanet/lava/blob/main/protocol/badgegenerator/Readme.md)

---

### TendermintRPC / JSON-RPC interface:
```typescript
  const blockResponse = await lavaSDK.sendRelay({
    method: "block",
    params: ["5"],
  });
```
Here, `method` is the RPC method and `params` is an array of strings representing parameters for the method.

You can find more examples for tendermintRPC sendRelay calls [TendermintRPC examples](https://github.com/lavanet/lava-sdk/blob/main/examples/tendermintRPC.ts)

### Rest API interface:
```typescript
const data = await lavaSDK.sendRelay({
  method: "GET",
  url: "/cosmos/bank/v1beta1/denoms_metadata",
  data: {
    "pagination.count_total": true,
    "pagination.reverse": "true",
  },
});
```
In this case, `method` is the HTTP method (either GET or POST), `url` is the REST endpoint, and `data` is the query data.

You can find more examples for rest sendRelay calls [Rest examples](https://github.com/lavanet/lava-sdk/blob/main/examples/restAPI.ts)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- Troubleshooting -->
# Troubleshooting

### <b> Webpack >= 5 </b>
If you are using `create-react-app` version 5 or higher, or `Angular` version 11 or higher, you may encounter build issues. This is because these versions use `webpack version 5`, which does not include Node.js polyfills.

#### <b> Create-react-app solution </b>

1. Install react-app-rewired and the missing modules:
```bash
yarn add --dev react-app-rewired crypto-browserify stream-browserify browserify-zlib assert stream-http https-browserify os-browserify url buffer process net tls bufferutil utf-8-validate path-browserify
```
2. Create `config-overrides.js` in the root of your project folder, and append the following lines:
```javascript
const webpack = require("webpack");

module.exports = function override(config) {
  const fallback = config.resolve.fallback || {};
  Object.assign(fallback, {
    crypto: require.resolve("crypto-browserify"),
    stream: require.resolve("stream-browserify"),
    assert: require.resolve("assert"),
    http: require.resolve("stream-http"),
    https: require.resolve("https-browserify"),
    os: require.resolve("os-browserify"),
    url: require.resolve("url"),
    zlib: require.resolve("browserify-zlib"),
    fs: false,
    bufferutil: require.resolve("bufferutil"),
    "utf-8-validate": require.resolve("utf-8-validate"),
    path: require.resolve("path-browserify"),
  });
  config.resolve.fallback = fallback;
  config.plugins = (config.plugins || []).concat([
    new webpack.ProvidePlugin({
      process: "process/browser",
      Buffer: ["buffer", "Buffer"],
    }),
  ]);
  config.ignoreWarnings = [/Failed to parse source map/];
  config.module.rules.push({
    test: /\.(js|mjs|jsx)$/,
    enforce: "pre",
    loader: require.resolve("source-map-loader"),
    resolve: {
      fullySpecified: false,
    },
  });
  return config;
};
```

3. In the `package.json` change the script for start, test, build and eject:
```JSON
"scripts": {
    "start": "react-app-rewired start",
    "build": "react-app-rewired build",
    "test": "react-app-rewired test",
    "eject": "react-scripts eject"
},
```
#### <b> Angular solution (TBD)</b>

<p align="right">(<a href="#readme-top">back to top</a>)</p>


[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![Apache 2 License][license-shield]]([license-url])
[![LinkedIn][linkedin-shield]][linkedin-url]


<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[contributors-shield]: https://img.shields.io/github/contributors/lavanet/lava-sdk.svg?style=for-the-badge
[contributors-url]: https://github.com/lavanet/lava-sdk/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/lavanet/lava-sdk.svg?style=for-the-badge
[forks-url]: https://github.com/lavanet/lava-sdk/network/members
[stars-shield]: https://img.shields.io/github/stars/lavanet/lava-sdk.svg?style=for-the-badge
[stars-url]: https://github.com/lavanet/lava-sdk/stargazers
[issues-shield]: https://img.shields.io/github/issues/lavanet/lava-sdk.svg?style=for-the-badge
[issues-url]: https://github.com/lavanet/lava-sdk/issues
[license-shield]: https://img.shields.io/github/license/lavanet/lava-sdk.svg?style=for-the-badge
[license-url]: https://github.com/lavanet/lava-sdk/blob/main/LICENSE
[linkedin-shield]: https://img.shields.io/badge/-LinkedIn-black.svg?style=for-the-badge&logo=linkedin&colorB=555
[linkedin-url]: https://www.linkedin.com/company/lava-network/


# Contribution FAQ

Q: how to compile the protobufs?
A: 
```bash
sudo apt install -y protobuf-compiler
```
cd go/.../lava/ecosystem/lava-sdk

If you've made changes to the protobufs directory run: 
```bash
./scripts/protoc.sh 
```
If you've made changes to relay.proto specifically run:
```bash
./scripts/protoc_grpc_relay.sh 
```