package keeper

import (
	"bytes"
	"context"

	burnmoduletypes "github.com/monolythium/mono-chain/x/burn/types"
	"github.com/monolythium/mono-chain/x/mono/types"
)

func (k msgServer) RegisterValidator(
	ctx context.Context,
	msg *types.MsgRegisterValidator,
) (*types.MsgRegisterValidatorResponse, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, types.ErrParamsRead
	}

	if err := k.validateAddresses(msg); err != nil {
		return nil, err
	}

	if err := k.validateFunds(msg, params); err != nil {
		return nil, err
	}

	if err := k.burnFunds(ctx, msg); err != nil {
		return nil, err
	}

	// bond the validator via the unwrapped staking MsgServer
	_, err = k.stakingMsgServer.CreateValidator(ctx, &msg.CreateValidator)
	if err != nil {
		return nil, err
	}

	return &types.MsgRegisterValidatorResponse{}, nil
}

func (k msgServer) validateAddresses(msg *types.MsgRegisterValidator) error {
	senderBytes, err := k.addressCodec.StringToBytes(msg.Sender)
	if err != nil {
		return err
	}

	validatorBytes, err := k.validatorAddressCodec.StringToBytes(msg.CreateValidator.ValidatorAddress)
	if err != nil {
		return err
	}

	if !bytes.Equal(senderBytes, validatorBytes) {
		return types.ErrRegistrationAddressMismatch
	}

	return nil
}

func (k msgServer) validateFunds(msg *types.MsgRegisterValidator, params types.Params) error {
	// Validate burn denom + amount
	if msg.Burn.Denom != params.ValidatorRegistrationBurn.Denom {
		return types.ErrBurnDenomMismatch
	}
	if msg.Burn.Amount.IsNil() || msg.Burn.Amount.LT(params.ValidatorRegistrationBurn.Amount) {
		return types.ErrBurnBelowRequired
	}

	// Validate min self-delegation
	if msg.CreateValidator.Value.Denom != params.ValidatorMinSelfDelegation.Denom {
		return types.ErrDelegationDenomMismatch
	}
	if msg.CreateValidator.MinSelfDelegation.IsNil() ||
		msg.CreateValidator.MinSelfDelegation.LT(params.ValidatorMinSelfDelegation.Amount) {
		return types.ErrMinSelfDelegationBelowRequired
	}
	if msg.CreateValidator.Value.Amount.IsNil() ||
		msg.CreateValidator.Value.Amount.LT(params.ValidatorMinSelfDelegation.Amount) {
		return types.ErrMinSelfDelegationBelowRequired
	}

	return nil
}

func (k msgServer) burnFunds(ctx context.Context, msg *types.MsgRegisterValidator) error {
	_, err := k.burnMsgServer.Burn(ctx, &burnmoduletypes.MsgBurn{
		FromAddress: msg.Sender,
		Amount:      msg.Burn,
	})

	return err
}
