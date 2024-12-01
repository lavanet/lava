<!--
parent:
  order: false
-->

<div align="center">
  <h1> <img src="https://user-images.githubusercontent.com/2770565/223762290-44afc792-8ad4-4dbb-b2c2-532780d6c5de.png" alt="Logo" width="30" height="25"> Lava Network  </h1>
</div>

![image](https://user-images.githubusercontent.com/2770565/203528359-dced4d06-f020-4b6a-bb5f-319124924689.png)

### What is Lava?

The Lava Protocol aims to provide decentralized and scalable access to blockchain data through the use of a network of providers and consumers. It utilizes a proof-of-stake consensus mechanism and incentivizes participants through the use of its native LAVA token. The protocol includes features such as a stake-weighted pseudorandom pairing function, backfilling, and a lazy settlement process to improve scalability and efficiency. The roadmap for the Lava Protocol includes further development of governance, conflict resolution, privacy, and quality of service, as well as support for additional API specifications. It is designed to be a public good that enables decentralized access to the Web3 ecosystem.

Read more about Lava in the [whitepaper](http://lavanet.xyz/whitepaper) and visit the [Docs](https://docs.lavanet.xyz?utm_source=github.com&utm_medium=github&utm_campaign=readme)

## Lava blockchain

Lava is built using the [Cosmos SDK](https://github.com/cosmos/cosmos-sdk/) which runs on top of [Tendermint Core](https://github.com/tendermint/tendermint) consensus engine.

[Documentation](x/README.md)

**Note**: Requires [Go 1.20.5](https://golang.org/dl/)

## Quick Start
 
The best way to start working with lava is to use docker, for additional reading go to:
[Running via compose](docker/README.md)

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for details on how to contribute. If you want to follow the updates or learn more about the latest design then join our [Discord](https://discord.com/invite/Tbk5NxTCdA).

## Developing 

### Installing development dependencies

before running the scripts make sure you have go installed and added to $PATH, you can validate by running `which go`
init_install will install all necessary dependencies to develop on lava.

```bash
./scripts/init_install.sh
```

### Building the binaries

LAVA_BINARY=all will build all lava binaries (lavad, lavap, lavavisor) and place them in the go bin path on your environment.

```bash
LAVA_BINARY=all make install
```

#### Building the binaries locally

You can also build the binaries locally (path will be ./build/...) by running

```bash
LAVA_BINARY=all make build
```

#### Building only a specific binary

it is possible to build only one binary: lavad/lavap/lavavisor

```bash
LAVA_BINARY=lavad make install
```

Or check out the latest [release](https://github.com/lavanet/lava/releases).

### Add `lavad`/`lavap` autocomplete

You can add a useful autocomplete feature to `lavad` & `lavap` with a simple bash [script](https://github.com/lavanet/lava/blob/main/scripts/automation_scripts/lava_auto_completion_install.sh).

## Join Lava

Join Lava's testnet, [read instructions here](https://docs.lavanet.xyz/testnet?utm_source=github.com&utm_medium=github&utm_campaign=readme)

⚠️ THERE'S NO MAINNET LAVA TOKEN, BE AWARE OF SCAMS.

## Community

- [Github Discussions](https://github.com/lavanet/lava/discussions)
- [Discord](https://discord.com/invite/Tbk5NxTCdA)
- [X (formerly Twitter)](https://x.com/lavanetxyz)
