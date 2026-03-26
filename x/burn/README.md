---
sidebar_position: 1
---

# `x/burn`

## Abstract

This document specifies the burn module of the Monolythium blockchain.

The burn module implements a dual-mechanism deflationary token system. It automatically
burns a governance-configurable percentage of native transaction fees each block, and
provides a voluntary burn path for any account holder. All burns are tracked globally
and per-account, creating an on-chain audit trail of supply reduction.

The module is designed to run in `BeginBlock` before `x/mint` and `x/distribution`,
ensuring fees are partially removed from circulation before inflation is applied and
rewards are distributed.

## Contents

* [Concepts](#concepts)
  * [Fee Burn](#fee-burn)
  * [Voluntary Burn](#voluntary-burn)
    * [Burn Tracking](#burn-tracking)
* [State](#state)
  * [Params](#params)
  * [GlobalBurnCount](#globalburncount)
  * [GlobalBurnTotal](#globalburntotal)
  * [AccountBurnTotal](#accountburntotal)
* [Begin-Block](#begin-block)
  * [Fee Burn Processing](#fee-burn-processing)
* [Messages](#messages)
  * [MsgBurn](#msgburn)
  * [MsgUpdateParams](#msgupdateparams)
* [Events](#events)
  * [BeginBlocker](#beginblocker)
* [Parameters](#parameters)
* [Client](#client)
  * [CLI](#cli)
  * [gRPC](#grpc)
  * [REST](#rest)

## Concepts

### Fee Burn

At the beginning of each block, the burn module inspects the `fee_collector` module
account for accumulated native transaction fees. A percentage of these fees, determined
by the `fee_burn_percent` parameter, is transferred to the burn module account and
permanently destroyed.

The remainder stays in `fee_collector` for normal processing by `x/distribution`,
which allocates it to validators and delegators. This creates a deflationary pressure
that counterbalances the inflationary pressure from `x/mint`.

```go
fees_in_collector = bank.GetBalance(fee_collector, "alyth")
burn_amount       = floor(fee_burn_percent * fees_in_collector)
remainder         = fees_in_collector - burn_amount
```

The `floor()` truncation ensures fractional-atto amounts always round down, keeping
the burn conservative and the remainder slightly larger.

### Voluntary Burn

Any account may permanently destroy native tokens by submitting a `MsgBurn`
transaction. The tokens are transferred from the sender to the burn module account,
then destroyed via the bank module's `BurnCoins` method.

Only the native bond denomination (`alyth`) can be burned. Attempts to burn other
denominations are rejected.

### Burn Tracking

Every burn operation — whether from fee processing or voluntary action — updates three
on-chain trackers:

1. **GlobalBurnCount**: A monotonically increasing sequence counting total burn operations.
2. **GlobalBurnTotal**: The cumulative amount of native tokens burned across all operations.
3. **AccountBurnTotal**: A per-account map tracking how much each address has burned.

For fee burns, the tracked address is the `fee_collector` module account address.

## State

The burn module uses `cosmossdk.io/collections` for type-safe state management.

### Params

Module parameters are stored under a single key.

* **Prefix**: `0x00`
* **Key**: (none — singleton)
* **Value**: `ProtocolBuffer(Params)`

```protobuf
message Params {
  string fee_burn_percent = 1; // cosmos.Dec
}
```

### GlobalBurnCount

A global monotonic counter incremented on every burn operation.

* **Prefix**: `0x01`
* **Key**: (none — singleton)
* **Value**: `uint64` (sequence)

### GlobalBurnTotal

The cumulative total of all native tokens burned.

* **Prefix**: `0x02`
* **Key**: (none — singleton)
* **Value**: `math.Int`

### AccountBurnTotal

Per-account cumulative burn totals.

* **Prefix**: `0x03`
* **Key**: `AccAddress`
* **Value**: `math.Int`

## Begin-Block

### Fee Burn Processing

At the beginning of each block, the module executes `ProcessFeeBurn`:

```go
begin_block():
    params = store.Get(Params)
    fee_balance = bank.GetBalance(fee_collector, bond_denom)
    burn_amount = truncate(params.fee_burn_percent * fee_balance)

    if burn_amount <= 0:
        return  // nothing to burn

    bank.SendCoinsFromModuleToModule(fee_collector, burn, burn_amount)
    bank.BurnCoins(burn, burn_amount)

    // update trackers
    GlobalBurnCount.Next()
    GlobalBurnTotal += burn_amount
    AccountBurnTotal[fee_collector_address] += burn_amount

    emit EventFeeBurn(burn_percent, burn_amount)
```

This **must** run before `x/mint` and `x/distribution` in the `BeginBlock` ordering.
If it ran after `x/mint`, newly minted inflation would also be subject to the fee burn
percentage, defeating the inflation mechanism.

## Messages

### MsgBurn

A token holder burns native tokens from their own account.

```protobuf
message MsgBurn {
  option (cosmos.msg.v1.signer) = "from_address";
  string from_address                    = 1;
  cosmos.base.v1beta1.Coin amount        = 2;
}
```

The message handler:

1. Decodes `from_address` using the EVM-aware address codec.
2. Transfers the specified `amount` from the sender to the burn module account.
3. If the amount is not positive, returns as a no-op.
4. Validates that the denomination is the native bond denomination.
5. Burns the coins via `bank.BurnCoins`.
6. Increments `GlobalBurnCount`, adds to `GlobalBurnTotal`, and adds to `AccountBurnTotal[sender]`.

The message fails if:

* The sender address is invalid.
* The denomination is not the native bond denomination (`alyth`).
* The sender has insufficient funds.

### MsgUpdateParams

Updates the module parameters. Only the governance module account may execute this message.

```protobuf
message MsgUpdateParams {
  option (cosmos.msg.v1.signer) = "authority";
  string authority               = 1;
  Params params                  = 2;
}
```

The message fails if:

* The authority does not match the governance module address.
* The `fee_burn_percent` is nil, negative, or greater than `1.0`.

## Events

### BeginBlocker

| Type       | Attribute Key  | Attribute Value             |
| ---------- | -------------- | --------------------------- |
| `fee_burn` | `burn_percent` | `{params.fee_burn_percent}` |
| `fee_burn` | `amount`       | `{burn_amount}{bond_denom}` |

The `fee_burn` event is only emitted when the calculated burn amount is positive.

## Parameters

The burn module contains the following parameter:

| Key                | Type  | Default | Valid Range  |
| ------------------ | ----- | ------- | ------------ |
| `fee_burn_percent` | `Dec` | `0`     | `[0.0, 1.0]` |

A value of `0` disables fee burning entirely. A value of `1.0` burns 100% of collected
fees, leaving nothing for validator/delegator rewards via `x/distribution`.

Parameters are updatable via governance proposal using `MsgUpdateParams`.

## Client

### CLI

The burn module provides the following CLI commands via AutoCLI.

#### Query

##### params

Query the current module parameters.

```shell
monod query burn params
```

##### burn-stats

Query global burn statistics (total operations and total amount burned).

```shell
monod query burn burn-stats
```

##### account-burns

Query the cumulative burn total for a specific account.

```shell
monod query burn account-burns [address]
```

#### Transactions

##### burn

Burn native tokens from the signing account.

```shell
monod tx burn burn [amount] --from [key]
```

Example:

```shell
monod tx burn burn 1000000000000000000alyth --from alice
```

##### update-params-proposal

Submit a governance proposal to update the module parameters.

```shell
monod tx burn update-params-proposal [params] --from [key]
```

### gRPC

The burn module exposes the following gRPC endpoints:

#### Query

| Method                                   | Description                           |
| ---------------------------------------- | ------------------------------------- |
| `monolythium.burn.v1.Query/Params`       | Query module parameters               |
| `monolythium.burn.v1.Query/BurnStats`    | Query global burn statistics          |
| `monolythium.burn.v1.Query/AccountBurns` | Query burn total for a single account |

#### Msg

| Method                                   | Description              |
| ---------------------------------------- | ------------------------ |
| `monolythium.burn.v1.Msg/Burn`           | Burn native tokens       |
| `monolythium.burn.v1.Msg/UpdateParams`   | Update module parameters |

### REST

| Verb | Path                                                       | Description                           |
| ---- | ---------------------------------------------------------- | ------------------------------------- |
| GET  | `/monolythium/mono-chain/burn/v1/params`                   | Query module parameters               |
| GET  | `/monolythium/mono-chain/burn/v1/stats`                    | Query global burn statistics          |
| GET  | `/monolythium/mono-chain/burn/v1/accounts/{address}/burns` | Query burn total for a single account |
