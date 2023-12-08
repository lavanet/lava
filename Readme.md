<!--
parent:
  order: false
-->

[![Website](https://img.shields.io/badge/WEBSITE-https%3A%2F%2Fwww.lavanet.xyz-green?style=for-the-badge)](https://www.lavanet.xyz) &emsp;  [![Discord](https://img.shields.io/discord/963778337904427018?color=green&logo=discord&logoColor=white&style=for-the-badge)](https://discord.gg/EKzbc6bx) &emsp; [![Docs](https://img.shields.io/badge/DOCS-https%3A%2F%2Fdocs.lavanet.xyz-green?style=for-the-badge)](https://docs.lavanet.xyz) 

<div align="center">
  <h1> <img src="https://user-images.githubusercontent.com/2770565/223762290-44afc792-8ad4-4dbb-b2c2-532780d6c5de.png" alt="Logo" width="30" height="25"> Lava Network  </h1>
</div>

![LavaCubicBanner](https://github.com/lavanet/lava/assets/82295340/b902152e-0351-46d4-a82f-dd9ea40e3ecf)

Lava Protocol is multichain RPC and decentralized web3 APIs that **just freakin' works**! Lava Protocol implements a proof-of-stake (PoS) blockchain based upon [Cosmos SDK](https://github.com/cosmos/cosmos-sdk/) with a native LAVA token. Lava is an open network of 30+ blockchains and ecosystems, with a modular and interoperable system for supporting new web3 APIs.
You can learn more about Lava in the [litepaper](https://litepaper.lavanet.xyz?utm_source=github.com&utm_medium=github&utm_campaign=readme) or by visiting the [Docs](https://docs.lavanet.xyz?utm_source=github.com&utm_medium=github&utm_campaign=readme).


## 🧰 Repository Contents

This repository (**@lavanet/lava**) contains the source code for all official lava binaries ([`lavad`](https://github.com/lavanet/lava/tree/main/cmd/lavad), [`lavap`](https://github.com/lavanet/lava/tree/main/cmd/lavap), [`lavavisor`](https://github.com/lavanet/lava/tree/main/cmd/lavavisor)), as well as [LavaSDK](https://github.com/lavanet/lava/tree/main/ecosystem/lava-sdk), and [LavaJS](https://github.com/lavanet/lava/tree/main/ecosystem/lavajs). 

Lava uses a monorepository structure to unify most of its products in the same location. The source code for Lava's technical documentation website is available in a separate repository ([@lavanet/docs](https://github.com/lavanet/lava)).

## 💻 Developing on Lava

### Installing development dependencies
**Note**: Requires [Go 1.20.5](https://golang.org/dl/)
Before running the scripts, make sure you have go installed and added to $PATH, you can validate by running `which go`
`init_install` will install all necessary dependencies to develop on lava.
```bash
./scripts/init_install.sh
```

### Building the binaries 
`install-all` will build all lava binaries (lavad, lavap, lavavisor) and place them in the go bin path on your environment.
```bash
make install-all
```

### Building the binaries locally
You can also build the binaries locally (path will be ./build/...) by running 
```bash
make build-all
```

### Building only a specific binary
it is possible to also build only one binary, for example lavad only. 
```bash
LAVA_BINARY=lavad make install
```

Learn more about installing the binaries on our [docs](https://docs.lavanet.xyz/install-lava)
Or check out the latest [release](https://github.com/lavanet/lava/releases).

### Add `lavad` autocomplete

You can add a useful autocomplete feature to `lavad` with a simple bash [script](https://github.com/lavanet/lava/blob/main/scripts/lavad_auto_completion_install.sh).

## 🔥 Quick Start

Join Lava's testnet, [read instructions here](https://docs.lavanet.xyz/testnet?utm_source=github.com&utm_medium=github&utm_campaign=readme)

⚠️ THERE'S NO MAINNET LAVA TOKEN, BE AWARE OF SCAMS.

## Community

- [Discord](https://discord.gg/lavanetxyz)
- [Twitter](https://twitter.com/lavanetxyz)
- [Discourse](https://community.lavanet.xyz/)
- [Upvoty](https://lavanet.upvoty.com/)
