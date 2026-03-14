package keeper_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/app"
	burnmoduletypes "github.com/monolythium/mono-chain/x/burn/types"
	"github.com/monolythium/mono-chain/x/mono/keeper"
	"github.com/monolythium/mono-chain/x/mono/types"
)

var (
	regAddr1          = bytes.Repeat([]byte{0x01}, 20)
	regAddr2          = bytes.Repeat([]byte{0x02}, 20)
	testRegBurnAmt    = math.NewIntWithDecimal(100_000, 18) // 100k LYTH in alyth
	testMinSelfDelAmt = math.NewIntWithDecimal(100_000, 18)
)

func validRegMsg(addrBytes []byte) *types.MsgRegisterValidator {
	return &types.MsgRegisterValidator{
		Sender: sdk.AccAddress(addrBytes).String(),
		CreateValidator: stakingtypes.MsgCreateValidator{
			ValidatorAddress:  sdk.ValAddress(addrBytes).String(),
			MinSelfDelegation: testMinSelfDelAmt,
			Value:             sdk.NewCoin(app.DefaultBondDenom, testMinSelfDelAmt),
			Description:       stakingtypes.Description{Moniker: "test-validator"},
			Commission: stakingtypes.CommissionRates{
				Rate:          math.LegacyNewDecWithPrec(10, 2),
				MaxRate:       math.LegacyNewDecWithPrec(20, 2),
				MaxChangeRate: math.LegacyNewDecWithPrec(1, 2),
			},
		},
		Burn: sdk.NewCoin(app.DefaultBondDenom, testRegBurnAmt),
	}
}

func setRegParams(t *testing.T, f *fixture) {
	t.Helper()
	params := types.NewParams(
		math.LegacyNewDecWithPrec(90, 2),
		sdk.NewCoin(app.DefaultBondDenom, testRegBurnAmt),
		sdk.NewCoin(app.DefaultBondDenom, testMinSelfDelAmt),
	)
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))
}

func TestRegisterValidator(t *testing.T) {
	testCases := []struct {
		name    string
		setup   func(t *testing.T, f *fixture)
		msgFn   func() *types.MsgRegisterValidator // nil = validRegMsg(regAddr1)
		expErr  bool
		errType error
	}{
		// Functional paths
		{
			name: "happy path with production-scale amounts",
		},
		{
			name: "excess burn passes",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.Burn.Amount = testRegBurnAmt.Mul(math.NewInt(2))
				return msg
			},
		},
		{
			name: "sender != validator operator",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.Sender = sdk.AccAddress(regAddr2).String()
				return msg
			},
			expErr:  true,
			errType: types.ErrRegistrationAddressMismatch,
		},
		{
			name: "burn denom mismatch",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.Burn.Denom = "uatom"
				return msg
			},
			expErr:  true,
			errType: types.ErrBurnDenomMismatch,
		},
		{
			name: "burn below required",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.Burn.Amount = testRegBurnAmt.Sub(math.OneInt())
				return msg
			},
			expErr:  true,
			errType: types.ErrBurnBelowRequired,
		},
		{
			name: "delegation denom mismatch",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.CreateValidator.Value.Denom = "uatom"
				return msg
			},
			expErr:  true,
			errType: types.ErrDelegationDenomMismatch,
		},
		{
			name: "min self delegation below required",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.CreateValidator.MinSelfDelegation = testMinSelfDelAmt.Sub(math.OneInt())
				return msg
			},
			expErr:  true,
			errType: types.ErrMinSelfDelegationBelowRequired,
		},
		{
			name: "value amount below required",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.CreateValidator.Value.Amount = testMinSelfDelAmt.Sub(math.OneInt())
				return msg
			},
			expErr:  true,
			errType: types.ErrMinSelfDelegationBelowRequired,
		},
		{
			name: "params not set",
			setup: func(t *testing.T, f *fixture) {
				require.NoError(t, f.keeper.Params.Remove(f.ctx))
			},
			expErr:  true,
			errType: types.ErrParamsRead,
		},
		{
			name: "burn module returns error",
			setup: func(_ *testing.T, f *fixture) {
				f.mockBurnServer.burnFn = func(
					_ context.Context,
					_ *burnmoduletypes.MsgBurn,
				) (*burnmoduletypes.MsgBurnResponse, error) {
					return nil, errors.New("mock: insufficient funds")
				}
			},
			expErr: true,
		},
		{
			name: "staking CreateValidator returns error",
			setup: func(_ *testing.T, f *fixture) {
				f.mockStakingServer.createValidatorFn = func(
					_ context.Context,
					_ *stakingtypes.MsgCreateValidator,
				) (*stakingtypes.MsgCreateValidatorResponse, error) {
					return nil, errors.New("mock: staking error")
				}
			},
			expErr: true,
		},
		{
			name: "already-registered validator (duplicate)",
			setup: func(_ *testing.T, f *fixture) {
				f.mockStakingServer.createValidatorFn = func(
					_ context.Context,
					_ *stakingtypes.MsgCreateValidator,
				) (*stakingtypes.MsgCreateValidatorResponse, error) {
					return nil, stakingtypes.ErrValidatorOwnerExists
				}
			},
			expErr:  true,
			errType: stakingtypes.ErrValidatorOwnerExists,
		},
		// Nil / zero / empty inputs
		{
			name: "zero-value msg errors without panic",
			msgFn: func() *types.MsgRegisterValidator {
				return &types.MsgRegisterValidator{}
			},
			expErr: true,
		},
		{
			name: "empty sender",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.Sender = ""
				return msg
			},
			expErr:  true,
			errType: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty validator address",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.CreateValidator.ValidatorAddress = ""
				return msg
			},
			expErr:  true,
			errType: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "nil burn amount",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.Burn = sdk.Coin{Denom: app.DefaultBondDenom}
				return msg
			},
			expErr:  true,
			errType: types.ErrBurnBelowRequired,
		},
		{
			name: "nil min self delegation",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.CreateValidator.MinSelfDelegation = math.Int{}
				return msg
			},
			expErr:  true,
			errType: types.ErrMinSelfDelegationBelowRequired,
		},
		{
			name: "nil value amount",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.CreateValidator.Value = sdk.Coin{Denom: app.DefaultBondDenom}
				return msg
			},
			expErr:  true,
			errType: types.ErrMinSelfDelegationBelowRequired,
		},
		{
			name: "empty burn denom",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.Burn.Denom = ""
				return msg
			},
			expErr:  true,
			errType: types.ErrBurnDenomMismatch,
		},
		{
			name: "empty value denom",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.CreateValidator.Value.Denom = ""
				return msg
			},
			expErr:  true,
			errType: types.ErrDelegationDenomMismatch,
		},
		// Negative / adversarial values
		{
			name: "negative burn amount",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.Burn.Amount = math.NewInt(-1)
				return msg
			},
			expErr:  true,
			errType: types.ErrBurnBelowRequired,
		},
		{
			name: "negative min self delegation",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.CreateValidator.MinSelfDelegation = math.NewInt(-1)
				return msg
			},
			expErr:  true,
			errType: types.ErrMinSelfDelegationBelowRequired,
		},
		{
			name: "negative value amount",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.CreateValidator.Value.Amount = math.NewInt(-1)
				return msg
			},
			expErr:  true,
			errType: types.ErrMinSelfDelegationBelowRequired,
		},
		{
			name: "malformed sender address",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.Sender = "not_a_valid_address"
				return msg
			},
			expErr: true,
		},
		{
			name: "malformed validator address",
			msgFn: func() *types.MsgRegisterValidator {
				msg := validRegMsg(regAddr1)
				msg.CreateValidator.ValidatorAddress = "not_a_valid_address"
				return msg
			},
			expErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := initFixture(t)
			setRegParams(t, f)
			if tc.setup != nil {
				tc.setup(t, f)
			}

			var msg *types.MsgRegisterValidator
			if tc.msgFn != nil {
				msg = tc.msgFn()
			} else {
				msg = validRegMsg(regAddr1)
			}

			ms := keeper.NewMsgServerImpl(f.keeper)
			_, err := ms.RegisterValidator(f.ctx, msg)

			if tc.expErr {
				require.Error(t, err)
				if tc.errType != nil {
					require.ErrorIs(t, err, tc.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRegisterValidator_Atomicity(t *testing.T) {
	f := initFixture(t)
	setRegParams(t, f)

	f.mockStakingServer.createValidatorFn = func(
		_ context.Context,
		_ *stakingtypes.MsgCreateValidator,
	) (*stakingtypes.MsgCreateValidatorResponse, error) {
		return nil, errors.New("mock: staking failure after burn")
	}

	msg := validRegMsg(regAddr1)
	ms := keeper.NewMsgServerImpl(f.keeper)
	_, err := ms.RegisterValidator(f.ctx, msg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "staking failure after burn")
	require.Contains(t, f.mockBurnServer.calls, "Burn", "burn must have executed before staking error")
	require.Contains(t, f.mockStakingServer.calls, "CreateValidator", "CreateValidator must have been attempted")
}

func TestRegisterValidator_ExecutionOrder(t *testing.T) {
	f := initFixture(t)
	setRegParams(t, f)

	var order []string
	f.mockBurnServer.burnFn = func(
		_ context.Context,
		_ *burnmoduletypes.MsgBurn,
	) (*burnmoduletypes.MsgBurnResponse, error) {
		order = append(order, "Burn")
		return &burnmoduletypes.MsgBurnResponse{}, nil
	}
	f.mockStakingServer.createValidatorFn = func(
		_ context.Context,
		_ *stakingtypes.MsgCreateValidator,
	) (*stakingtypes.MsgCreateValidatorResponse, error) {
		order = append(order, "CreateValidator")
		return &stakingtypes.MsgCreateValidatorResponse{}, nil
	}

	msg := validRegMsg(regAddr1)
	ms := keeper.NewMsgServerImpl(f.keeper)
	_, err := ms.RegisterValidator(f.ctx, msg)

	require.NoError(t, err)
	require.Equal(t, []string{"Burn", "CreateValidator"}, order,
		"burn must execute before CreateValidator for CacheMultiStore atomicity")
}

func TestRegisterValidator_TrustBoundary(t *testing.T) {
	t.Run("burn uses msg.Sender as FromAddress", func(t *testing.T) {
		f := initFixture(t)
		setRegParams(t, f)

		msg := validRegMsg(regAddr1)
		ms := keeper.NewMsgServerImpl(f.keeper)
		_, err := ms.RegisterValidator(f.ctx, msg)

		require.NoError(t, err)
		require.NotNil(t, f.mockBurnServer.lastMsg)
		require.Equal(t, msg.Sender, f.mockBurnServer.lastMsg.FromAddress,
			"burnFunds must pass msg.Sender as MsgBurn.FromAddress")
	})

	t.Run("burn does not use validator operator address", func(t *testing.T) {
		f := initFixture(t)
		setRegParams(t, f)

		msg := validRegMsg(regAddr1)
		ms := keeper.NewMsgServerImpl(f.keeper)
		_, err := ms.RegisterValidator(f.ctx, msg)

		require.NoError(t, err)
		require.NotNil(t, f.mockBurnServer.lastMsg)
		// Same underlying bytes, different bech32 encoding (mono1... vs monovaloper1...)
		require.NotEqual(t, msg.CreateValidator.ValidatorAddress, f.mockBurnServer.lastMsg.FromAddress,
			"burnFunds must NOT use ValidatorAddress as burn source")
	})
}
