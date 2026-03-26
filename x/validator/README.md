---
sidebar_position: 1
---

# `x/validator`

## Abstract

This document specifies the validator module of the Monolythium blockchain.

The validator module enforces a permissioned validator registration path. Direct `MsgCreateValidator` is blocked after genesis; instead, prospective validators must submit a `MsgRegisterValidator` that burns a governance-configurable registration fee and enforces a minimum self-delegation before delegating to `x/staking` for the actual validator creation.

This creates an economic barrier to entry for the validator set, ensuring that only participants willing to permanently destroy capital can operate validators.

## Contents

* [Concepts](#concepts)
  * [Gated Validator Registration](#gated-validator-registration)
  * [Registration Requirements](#registration-requirements)
  * [MsgCreateValidator Restriction](#msgcreatevalidator-restriction)
* [State](#state)
  * [Params](#params)
* [State Transitions](#state-transitions)
  * [RegisterValidator](#registervalidator)
* [Messages](#messages)
  * [MsgRegisterValidator](#msgregistervalidator)
  * [MsgUpdateParams](#msgupdateparams)
* [Events](#events)
  * [Handlers](#handlers)
* [Parameters](#parameters)
* [Client](#client)
  * [CLI](#cli)
  * [gRPC](#grpc)
  * [REST](#rest)

## Concepts

### Gated Validator Registration

On most Cosmos SDK chains, any account can submit a `MsgCreateValidator` to join the validator set with no economic cost beyond the self-delegation. Monolythium replaces this open registration with a gated path that requires an irreversible token burn.

The sole registration path post-genesis is `MsgRegisterValidator`, which atomically:

1. Validates that the sender is the validator operator.
2. Validates the burn amount and self-delegation against governance parameters.
3. Burns the registration fee via `x/burn`.
4. Creates the validator via `x/staking`.

If any step fails, the entire transaction reverts - the burn and validator creation are atomic.

### Registration Requirements

Two governance-configurable parameters control the registration barrier:

* **Registration Burn**: A fixed amount of native tokens that must be permanently destroyed. This is a sunk cost — it cannot be recovered by unbonding.
* **Minimum Self-Delegation**: The minimum initial stake the validator must bond. Both `MinSelfDelegation` and `Value` (the initial delegation amount) on the embedded `MsgCreateValidator` must meet or exceed this threshold.

Both parameters must use the native bond denomination (`alyth`).

### MsgCreateValidator Restriction

Direct `MsgCreateValidator` is blocked at two levels to prevent bypass:

**Circuit Breaker (Msg Router)**: A `CircuitBreaker` implementation is registered on the application's msg service router. It rejects any `MsgCreateValidator` when the block height is greater than zero. This catches direct transactions, `x/authz` grant executions, and governance proposals that embed `MsgCreateValidator`.

**Restricted MsgServer (EVM Precompile)**: The staking precompile receives a wrapped `MsgServer` that overrides `CreateValidator` to return an error post-genesis. This catches `MsgCreateValidator` submitted via EVM smart contract calls to the staking precompile address.

Both layers permit `MsgCreateValidator` at block height `0` to allow genesis validator bootstrap via `gentx`.

## State

The validator module stores only governance parameters. It does not maintain validator records — those belong to `x/staking`.

### Params

* **Prefix**: `0x00` (`"params"`)
* **Key**: (none — singleton)
* **Value**: `ProtocolBuffer(Params)`

```protobuf
message Params {
  cosmos.base.v1beta1.Coin validator_registration_burn  = 1;
  cosmos.base.v1beta1.Coin validator_min_self_delegation = 2;
}
```

## State Transitions

### RegisterValidator

When a `MsgRegisterValidator` is processed, the following state transitions occur:

```go
register_validator(sender, create_validator, burn):
    params = store.Get(Params)

    // 1. Identity check
    sender_bytes  = addressCodec.Decode(sender)
    val_bytes     = valAddressCodec.Decode(create_validator.ValidatorAddress)
    require(sender_bytes == val_bytes)

    // 2. Fund validation
    require(burn.Denom == params.ValidatorRegistrationBurn.Denom)
    require(burn.Amount >= params.ValidatorRegistrationBurn.Amount)
    require(create_validator.Value.Denom == params.ValidatorMinSelfDelegation.Denom)
    require(create_validator.Value.Amount >= params.ValidatorMinSelfDelegation.Amount)
    require(create_validator.MinSelfDelegation >= params.ValidatorMinSelfDelegation.Amount)

    // 3. Burn (irreversible)
    x/burn.BurnFromAccount(sender, burn)

    // 4. Create validator (delegates to x/staking)
    x/staking.CreateValidator(create_validator)

    emit EventRegisterValidator()
```

The entire operation is atomic — if any step fails, all state changes (including the burn) revert. The burn-before-create ordering follows the Checks-Effects-Interactions pattern.

## Messages

### MsgRegisterValidator

Registers a new validator by burning the required fee and delegating to `x/staking`.

```protobuf
message MsgRegisterValidator {
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1;
  cosmos.staking.v1beta1.MsgCreateValidator create_validator = 2;
  cosmos.base.v1beta1.Coin burn = 3;
}
```

The `create_validator` field is a standard `x/staking` `MsgCreateValidator` containing the validator's public key, description, commission rates, and initial delegation.

The message fails if:

* The sender address is invalid.
* The validator address is invalid.
* The sender does not match the validator operator address.
* The burn denomination does not match the required denomination.
* The burn amount is below the required registration burn.
* The delegation denomination does not match the required denomination.
* The `MinSelfDelegation` is below the chain's required minimum.
* The `Value` (initial delegation) is below the chain's required minimum.
* The sender has insufficient funds for the burn.
* The `x/burn` module rejects the burn operation.
* The `x/staking` module rejects the validator creation.

### MsgUpdateParams

Updates the module parameters. Only the governance module account may execute this message.

```protobuf
message MsgUpdateParams {
  option (cosmos.msg.v1.signer) = "authority";
  string authority = 1;
  Params params    = 2;
}
```

The message fails if:

* The authority does not match the governance module address.
* Either parameter coin fails basic `sdk.Coin` validation.
* Either parameter uses a non-zero amount with a denomination other than the native bond denomination.

## Events

### Handlers

#### MsgRegisterValidator

| Type                 | Attribute Key | Attribute Value |
| -------------------- | ------------- | --------------- |
| `register_validator` | —             | —               |

`MsgRegisterValidator` transitively emits `x/bank` events (via `x/burn` -> `BurnCoins`) and `x/staking` events (via `CreateValidator`), in addition to the module's own `register_validator` event above.

## Parameters

The validator module contains the following parameters:

| Key                             | Type   | Default      |
| ------------------------------- | ------ | ------------ |
| `validator_registration_burn`   | `Coin` | `{alyth, 0}` |
| `validator_min_self_delegation` | `Coin` | `{alyth, 0}` |

Both parameters must use the native bond denomination. A zero amount disables the corresponding requirement.

Parameters are updatable via governance proposal using `MsgUpdateParams`.

## Client

### CLI

The validator module provides the following CLI commands via AutoCLI.

#### Query

##### params

Query the current module parameters.

```shell
monod query validator params
```

#### Transactions

##### update-params-proposal

Submit a governance proposal to update the module parameters.

```shell
monod tx validator update-params-proposal [params] --from [key]
```

Note: `MsgRegisterValidator` is not exposed via AutoCLI due to the complexity of its embedded `MsgCreateValidator` field (public key, commission rates, description). It must be submitted programmatically or via gRPC/REST.

### gRPC

The validator module exposes the following gRPC endpoints:

#### Query

| Method                                  | Description             |
| --------------------------------------- | ----------------------- |
| `monolythium.validator.v1.Query/Params` | Query module parameters |

#### Msg

| Method                                           | Description              |
| ------------------------------------------------ | ------------------------ |
| `monolythium.validator.v1.Msg/RegisterValidator` | Register a new validator |
| `monolythium.validator.v1.Msg/UpdateParams`      | Update module parameters |

### REST

| Verb | Path                                          | Description             |
| ---- | --------------------------------------------- | ----------------------- |
| GET  | `/monolythium/mono-chain/validator/v1/params` | Query module parameters |
