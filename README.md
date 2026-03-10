![Mono Chain](https://raw.githubusercontent.com/mono-labs-org/.github/prod/media/github-banners/monolythium/mono-chain.png)

# Monolythium

**Monolythium** is a blockchain built using Cosmos SDK and Tendermint.

## Networks

| Network   | EVM Chain ID (HEX)  | Cosmos Chain ID |
| :-------: | :-----------------: | :-------------: |
| mainnet   | `6941` (`0x1b1d`)   | `mono_6941-1`   |
| testnet   | `6940` (`0x1b1c`)   | `mono_6940-1`   |
| sprintnet | `262146` (`0x4002`) | `mono-sprint-1` |

## Get started

```shell
ignite chain serve
```

`serve` command installs dependencies, builds, initializes, and starts your blockchain in development.

### Configure

Your blockchain in development can be configured with `config.yml`. To learn more, see the [Ignite CLI docs](https://docs.ignite.com).

### Web Frontend

Additionally, Ignite CLI offers a frontend scaffolding feature (based on Vue) to help you quickly build a web frontend for your blockchain:

Use: `ignite scaffold vue`
This command can be run within your scaffolded blockchain project.

For more information see the [monorepo for Ignite front-end development](https://github.com/ignite/web).

## Release

To release a new version of your blockchain, create and push a new tag with `v` prefix. A new draft release with the configured targets will be created.

```shell
git tag v0.1
git push origin v0.1
```

After a draft release is created, make your final changes from the release page and publish it.

### Install

To install the latest version of your blockchain node's binary, execute the following command on your machine:

```shell
curl https://get.ignite.com/monolythium/mono@latest! | sudo bash
```

`monolythium/mono` should match the `username` and `repo_name` of the Github repository to which the source code was pushed. Learn more about [the install process](https://github.com/ignite/installer).

## Learn more

- [Ignite CLI](https://ignite.com/cli)
- [Tutorials](https://docs.ignite.com/guide)
- [Ignite CLI docs](https://docs.ignite.com)
- [Cosmos SDK docs](https://docs.cosmos.network)
- [Developer Chat](https://discord.com/invite/ignitecli)
