package keeper

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/monolythium/mono-chain/x/validator/types"
)

type Keeper struct {
	storeService          corestore.KVStoreService
	cdc                   codec.Codec
	addressCodec          address.Codec
	validatorAddressCodec address.Codec

	authority string

	burnKeeper       types.BurnKeeper
	stakingMsgServer types.StakingMsgServer

	Schema collections.Schema
	Params collections.Item[types.Params]
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	validatorAddressCodec address.Codec,

	authority string,

	burnKeeper types.BurnKeeper,
	stakingMsgServer types.StakingMsgServer,
) Keeper {
	if _, err := addressCodec.StringToBytes(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", err))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		storeService:          storeService,
		cdc:                   cdc,
		addressCodec:          addressCodec,
		validatorAddressCodec: validatorAddressCodec,

		authority: authority,

		burnKeeper:       burnKeeper,
		stakingMsgServer: stakingMsgServer,

		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) RegisterValidator(ctx context.Context, msg *types.MsgRegisterValidator) error {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return types.ErrParamsRead
	}

	sender, err := k.resolveSender(msg)
	if err != nil {
		return err
	}

	if err := k.validateRegistrationFunds(msg, params); err != nil {
		return err
	}

	if err := k.burnKeeper.BurnFromAccount(ctx, sdk.AccAddress(sender), msg.Burn); err != nil {
		return err
	}

	if _, err := k.stakingMsgServer.CreateValidator(ctx, &msg.CreateValidator); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(types.EventTypeRegisterValidator),
	)

	return nil
}

func (k Keeper) resolveSender(msg *types.MsgRegisterValidator) ([]byte, error) {
	sender, err := k.addressCodec.StringToBytes(msg.Sender)
	if err != nil {
		return nil, err
	}

	validator, err := k.validatorAddressCodec.StringToBytes(msg.CreateValidator.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(sender, validator) {
		return nil, types.ErrRegistrationAddressMismatch
	}

	return sender, nil
}

func (k Keeper) validateRegistrationFunds(msg *types.MsgRegisterValidator, params types.Params) error {
	if msg.Burn.Denom != params.ValidatorRegistrationBurn.Denom {
		return types.ErrBurnDenomMismatch
	}

	if msg.Burn.Amount.IsNil() || msg.Burn.Amount.LT(params.ValidatorRegistrationBurn.Amount) {
		return types.ErrBurnBelowRequired
	}

	if msg.CreateValidator.Value.Denom != params.ValidatorMinSelfDelegation.Denom {
		return types.ErrDelegationDenomMismatch
	}

	if msg.CreateValidator.Value.Amount.IsNil() ||
		msg.CreateValidator.Value.Amount.LT(params.ValidatorMinSelfDelegation.Amount) {
		return types.ErrMinSelfDelegationBelowRequired
	}

	if msg.CreateValidator.MinSelfDelegation.IsNil() ||
		msg.CreateValidator.MinSelfDelegation.LT(params.ValidatorMinSelfDelegation.Amount) {
		return types.ErrMinSelfDelegationBelowRequired
	}

	return nil
}
