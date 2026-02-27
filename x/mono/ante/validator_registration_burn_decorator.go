package ante

import (
	"bytes"
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	stakingprecompile "github.com/cosmos/evm/precompiles/staking"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"

	burnkeeper "github.com/monolythium/mono-chain/x/burn/keeper"
	burnmoduletypes "github.com/monolythium/mono-chain/x/burn/types"
	"github.com/monolythium/mono-chain/x/mono/keeper"
	"github.com/monolythium/mono-chain/x/mono/types"
)

// ValidatorRegistrationBurnDecorator enforces that any transaction containing
// MsgCreateValidator must also include a MsgBurn of at least
// validator_registration_fee from the same key as the validator operator.
type ValidatorRegistrationBurnDecorator struct {
	monoKeeper    keeper.Keeper
	burnKeeper    burnkeeper.Keeper
	stakingKeeper interface {
		GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	}
	accountAddressCodec   address.Codec
	validatorAddressCodec address.Codec
}

func NewValidatorRegistrationBurnDecorator(
	mk keeper.Keeper,
	bk burnkeeper.Keeper,
	sk interface {
		GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	},
	accCodec, valCodec address.Codec,
) ValidatorRegistrationBurnDecorator {
	return ValidatorRegistrationBurnDecorator{
		monoKeeper:            mk,
		burnKeeper:            bk,
		stakingKeeper:         sk,
		accountAddressCodec:   accCodec,
		validatorAddressCodec: valCodec,
	}
}

func (vbd ValidatorRegistrationBurnDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	// Skip during genesis
	// Initial validator set doesn't require burn
	if !ctx.IsCheckTx() && (ctx.BlockHeight() == 0) {
		return next(ctx, tx, simulate)
	}

	params, err := vbd.monoKeeper.Params.Get(ctx)
	if err != nil {
		return ctx, types.ErrParamsRead
	}

	if params.ValidatorRegistrationFee.IsZero() {
		return next(ctx, tx, simulate)
	}

	var createMsg *stakingtypes.MsgCreateValidator
	var burnMsg *burnmoduletypes.MsgBurn
	for _, msgs := range tx.GetMsgs() {
		switch msg := msgs.(type) {
		case *stakingtypes.MsgCreateValidator:
			if createMsg != nil {
				return ctx, types.ErrDuplicateRegistrationInfo
			}
			createMsg = msg
		case *burnmoduletypes.MsgBurn:
			if burnMsg != nil {
				return ctx, types.ErrDuplicateRegistrationInfo
			}
			burnMsg = msg
		case *evmtypes.MsgEthereumTx:
			if err := vbd.handleEVMCreateValidator(ctx, msg, params); err != nil {
				return ctx, err
			}
		}
	}

	if createMsg == nil {
		return next(ctx, tx, simulate)
	}

	if burnMsg == nil {
		return ctx, types.ErrMissingBurnInfo
	}

	valAddrBytes, err := vbd.validatorAddressCodec.StringToBytes(createMsg.ValidatorAddress)
	if err != nil {
		return ctx, errorsmod.Wrapf(types.ErrInvalidValidatorAddress, "failed to decode validator address: %s", err)
	}

	burnAddrBytes, err := vbd.accountAddressCodec.StringToBytes(burnMsg.FromAddress)
	if err != nil {
		return ctx, errorsmod.Wrapf(types.ErrInvalidBurnAddress, "failed to decode burn from address: %s", err)
	}

	if !bytes.Equal(valAddrBytes, burnAddrBytes) {
		return ctx, types.ErrBurnSenderMismatch
	}

	if burnMsg.Amount.Denom != params.ValidatorRegistrationFee.Denom {
		return ctx, types.ErrBurnDenomMismatch
	}

	if burnMsg.Amount.Amount.LT(params.ValidatorRegistrationFee.Amount) {
		return ctx, errorsmod.Wrapf(
			types.ErrInsufficientBurnAmount,
			"Validator registration requires a burn of: %s %s",
			params.ValidatorRegistrationFee.Amount,
			params.ValidatorRegistrationFee.Denom,
		)
	}

	if createMsg.MinSelfDelegation.LT(params.ValidatorRegistrationFee.Amount) {
		return ctx, errorsmod.Wrapf(
			types.ErrInsufficientMinSelfDelegation,
			"minimum self-delegation must be at least %s %s",
			params.ValidatorRegistrationFee.Amount,
			params.ValidatorRegistrationFee.Denom,
		)
	}

	return next(ctx, tx, simulate)
}

// handleEVMCreateValidator checks if an EVM tx is calling the staking
// precompile's createValidator method and enforces burn requirements.
func (vbd ValidatorRegistrationBurnDecorator) handleEVMCreateValidator(
	ctx sdk.Context,
	msg *evmtypes.MsgEthereumTx,
	params types.Params,
) error {
	_, ethTx, err := evmtypes.UnpackEthMsg(msg)
	if err != nil {
		return nil // Not a valid eth msg, let it fail later
	}

	to := ethTx.To()
	if to == nil || *to != common.HexToAddress(evmtypes.StakingPrecompileAddress) {
		return nil
	}

	data := ethTx.Data()
	if len(data) < 4 {
		return nil
	}

	createValID := stakingprecompile.ABI.Methods[stakingprecompile.CreateValidatorMethod].ID
	if !bytes.Equal(data[:4], createValID) {
		return nil
	}

	// This is a createValidator precompile call - check burn history
	sender := sdk.AccAddress(msg.GetFrom())

	// Check if already a validator
	_, err = vbd.stakingKeeper.GetValidator(ctx, sdk.ValAddress(sender))
	if err == nil {
		return types.ErrAlreadyValidator
	}

	// Check burn history
	burned, err := vbd.burnKeeper.BurnAccountTotal.Get(ctx, sender)
	if err != nil {
		if err == collections.ErrNotFound {
			return errorsmod.Wrapf(
				types.ErrInsufficientBurnAmount,
				"no burns found for account; validator registration requires a burn of: %s",
				params.ValidatorRegistrationFee,
			)
		}
		return err
	}

	if burned.Amount.LT(params.ValidatorRegistrationFee.Amount) {
		return errorsmod.Wrapf(
			types.ErrInsufficientBurnAmount,
			"account burned %s but validator registration requires: %s",
			burned, params.ValidatorRegistrationFee,
		)
	}

	return nil
}
