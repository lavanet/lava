<!--
parent:
  order: false
-->

[![Website](https://img.shields.io/badge/WEBSITE-https%3A%2F%2Fwww.lavanet.xyz-green?style=for-the-badge)](https://www.lavanet.xyz) &emsp;  [![Discord](https://img.shields.io/discord/963778337904427018?color=green&logo=discord&logoColor=white&style=for-the-badge)](https://discord.gg/EKzbc6bx) &emsp; [![Docs](https://img.shields.io/badge/DOCS-https%3A%2F%2Fdocs.lavanet.xyz-green?style=for-the-badge)](https://docs.lavanet.xyz) 

<div align="center">
  <h1> <img src="https://user-images.githubusercontent.com/2770565/223762290-44afc792-8ad4-4dbb-b2c2-532780d6c5de.png" alt="Logo" width="30" height="25"> Lava Network  </h1>
</div>

![image](https://user-images.githubusercontent.com/2770565/203528359-dced4d06-f020-4b6a-bb5f-319124924689.png)

### What is Lava?
The Lava Protocol aims to provide decentralized and scalable access to blockchain data through the use of a network of providers and consumers. It utilizes a proof-of-stake consensus mechanism and incentivizes participants through the use of its native LAVA token. The protocol includes features such as a stake-weighted pseudorandom pairing function, backfilling, and a lazy settlement process to improve scalability and efficiency. The roadmap for the Lava Protocol includes further development of governance, conflict resolution, privacy, and quality of service, as well as support for additional API specifications. It is designed to be a public good that enables decentralized access to the Web3 ecosystem.

Read more about Lava in the [litepaper](https://litepaper.lavanet.xyz?utm_source=github.com&utm_medium=github&utm_campaign=readme) and visit the [Docs](https://docs.lavanet.xyz?utm_source=github.com&utm_medium=github&utm_campaign=readme)

## Lava blockchain

Lava is built using the [Cosmos SDK](https://github.com/cosmos/cosmos-sdk/) which runs on top of [Tendermint Core](https://github.com/tendermint/tendermint) consensus engine.

**Note**: Requires [Go 1.20.5](https://golang.org/dl/)

## Repository Contents
This repository (**@lavanet/lava**) contains the source code for all [lava binaries (`lavad`, `lavap`, `lavavisor`)](https://github.com/lavanet/lava/tree/main/cmd), [LavaSDK](https://github.com/lavanet/lava/tree/main/ecosystem/lava-sdk), [LavaJS](https://github.com/lavanet/lava/tree/main/ecosystem/lavajs).

Lava uses a monorepository structure to unify most of its products in the same location. The source code for Lava's technical documentation website is available in a separate repository.

### Installing development dependencies
before running the scripts make sure you have go installed and added to $PATH, you can validate by running `which go`
init_install will install all necessary dependencies to develop on lava.
```bash
./scripts/init_install.sh
```

## Building the binaries
install-all will build all lava binaries (lavad, lavap, lavavisor) and place them in the go bin path on your environment.
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


Or check out the latest [release](https://github.com/lavanet/lava/releases).

### Add `lavad` autocomplete

You can add a useful autocomplete feature to `lavad` with a simple bash [script](https://github.com/lavanet/lava/blob/update-readme-autocomplete/scripts/lavad_auto_completion_install.sh).

### Quick Start

Join Lava's testnet, [read instructions here](https://docs.lavanet.xyz/testnet?utm_source=github.com&utm_medium=github&utm_campaign=readme)

⚠️ THERE'S NO MAINNET LAVA TOKEN, BE AWARE OF SCAMS.

## Community

- [Discord](https://discord.gg/lavanetxyz)
- [Twitter](https://twitter.com/lavanetxyz)
