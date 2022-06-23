# lava
**lava** is a blockchain built using Cosmos SDK and Tendermint and created with [Ignite (Starport)](https://ignite.com/cli).

## Get started

* install starport on your computer from the starport repository: https://docs.starport.com/guide/install.html
* install Go Version 1.18 , follow the go installation instructions: https://go.dev/doc/install
* make sure you set your go environment properly: https://go.dev/doc/gopath_code#GOPATH
* checkout the repository, you might need a personal token: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token

## build your chain
```
ignite chain serve
```
you might want to provide arguments like -v for verbosity or -r for restart, use the help for more

`serve` command installs dependencies, builds, initializes, and starts your blockchain in development.

### Configure

Your blockchain in development can be configured with `config.yml`. To learn more, see the [Starport docs](https://docs.starport.com).

### Web Frontend

Starport has scaffolded a Vue.js-based web app in the `vue` directory. Run the following commands to install dependencies and start the app:

```
cd vue
npm install
npm run serve
```

The frontend app is built using the `@starport/vue` and `@starport/vuex` packages. For details, see the [monorepo for Starport front-end development](https://github.com/tendermint/vue).

## Release
To release a new version of your blockchain, create and push a new tag with `v` prefix. A new draft release with the configured targets will be created.

```
git tag v0.1
git push origin v0.1
```

After a draft release is created, make your final changes from the release page and publish it.
## Learn more

- [Starport](https://starport.com)
- [Tutorials](https://docs.starport.com/guide)
- [Starport docs](https://docs.starport.com)
- [Cosmos SDK docs](https://docs.cosmos.network)
- [Developer Chat](https://discord.gg/H6wGTY8sxw)

# running the chain
* during dev your chain is running locally with a single validator
* when compiling a binary called: "lavad", is created by starport on your $GOPATH (and if you configured go environment properly,
    it should be ~/go/bin/lavad)
* you can interact with the binary to send transactions and queries
* ./scripts/init_chain_commands.sh has examples on how to set up an initial chain with 5 servicers and 1 user,
    for basic testing of pairing on ethereum mainnet, you can run it or have a look at the commands to learn more


# Lava e2e Test & Mock Proxy 
* check out ./testutils/e2e/Readme.md
