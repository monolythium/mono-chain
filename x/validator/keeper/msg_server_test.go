package keeper_test

import (
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"go.uber.org/mock/gomock"

	"github.com/monolythium/mono-chain/x/validator/keeper"
	"github.com/monolythium/mono-chain/x/validator/types"
)

// makeRegisterMsg builds a valid MsgRegisterValidator for tests.
// Raw address bytes are encoded using both address codecs so sender == validator.
func (s *ValidatorKeeperTestSuite) makeRegisterMsg(rawAddr []byte, burnAmt, delegationAmt int64) *types.MsgRegisterValidator {
	senderStr, err := s.addressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)
	valStr, err := s.valAddressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)

	return &types.MsgRegisterValidator{
		Sender: senderStr,
		CreateValidator: stakingtypes.MsgCreateValidator{
			ValidatorAddress:  valStr,
			Value:             sdk.NewCoin("alyth", math.NewInt(delegationAmt)),
			MinSelfDelegation: math.NewInt(delegationAmt),
		},
		Burn: sdk.NewCoin("alyth", math.NewInt(burnAmt)),
	}
}

// setNonTrivialParams sets params that enforce real minimums.
func (s *ValidatorKeeperTestSuite) setNonTrivialParams(burnRequired, delegationRequired int64) {
	params := types.NewParams(
		sdk.NewCoin("alyth", math.NewInt(burnRequired)),
		sdk.NewCoin("alyth", math.NewInt(delegationRequired)),
	)
	err := s.validatorKeeper.Params.Set(s.ctx, params)
	s.Require().NoError(err)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_ParamsNotSet() {
	storeKey := storetypes.NewKVStoreKey("empty_register")
	storeService := runtime.NewKVStoreService(storeKey)
	testCtx := testutil.DefaultContextWithDB(s.T(), storeKey, storetypes.NewTransientStoreKey("transient_empty_register"))

	emptyKeeper := keeper.NewKeeper(
		storeService,
		s.encCfg.Codec,
		s.addressCodec,
		s.valAddressCodec,
		s.authority,
		s.burnKeeper,
		s.stakingMsgServer,
	)

	rawAddr := []byte("test_no_params______")
	msg := s.makeRegisterMsg(rawAddr, 100, 1000)

	err := emptyKeeper.RegisterValidator(testCtx.Ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrParamsRead)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_HappyPath() {
	s.setNonTrivialParams(100, 1000)
	rawAddr := []byte("test_validator______")
	msg := s.makeRegisterMsg(rawAddr, 100, 1000)
	senderAddr := sdk.AccAddress(rawAddr)

	gomock.InOrder(
		s.burnKeeper.EXPECT().BurnFromAccount(gomock.Any(), senderAddr, msg.Burn).Return(nil),
		s.stakingMsgServer.EXPECT().CreateValidator(gomock.Any(), &msg.CreateValidator).Return(&stakingtypes.MsgCreateValidatorResponse{}, nil),
	)

	resp, err := s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_SenderNotValidator() {
	s.setNonTrivialParams(100, 1000)
	senderRaw := []byte("sender_address______")
	validatorRaw := []byte("validator_address___")

	senderStr, err := s.addressCodec.BytesToString(senderRaw)
	s.Require().NoError(err)
	valStr, err := s.valAddressCodec.BytesToString(validatorRaw)
	s.Require().NoError(err)

	msg := &types.MsgRegisterValidator{
		Sender: senderStr,
		CreateValidator: stakingtypes.MsgCreateValidator{
			ValidatorAddress:  valStr,
			Value:             sdk.NewCoin("alyth", math.NewInt(1000)),
			MinSelfDelegation: math.NewInt(1000),
		},
		Burn: sdk.NewCoin("alyth", math.NewInt(100)),
	}

	_, err = s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrRegistrationAddressMismatch)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_InvalidSenderAddress() {
	msg := &types.MsgRegisterValidator{
		Sender: "bad_bech32_sender",
		CreateValidator: stakingtypes.MsgCreateValidator{
			ValidatorAddress:  "monovaloper1qqqq",
			Value:             sdk.NewCoin("alyth", math.NewInt(1000)),
			MinSelfDelegation: math.NewInt(1000),
		},
		Burn: sdk.NewCoin("alyth", math.NewInt(100)),
	}

	_, err := s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_InvalidValidatorAddress() {
	rawAddr := []byte("valid_sender________")
	senderStr, err := s.addressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)

	msg := &types.MsgRegisterValidator{
		Sender: senderStr,
		CreateValidator: stakingtypes.MsgCreateValidator{
			ValidatorAddress:  "bad_valoper_addr",
			Value:             sdk.NewCoin("alyth", math.NewInt(1000)),
			MinSelfDelegation: math.NewInt(1000),
		},
		Burn: sdk.NewCoin("alyth", math.NewInt(100)),
	}

	_, err = s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_BurnDenomMismatch() {
	s.setNonTrivialParams(100, 1000)
	rawAddr := []byte("test_burn_denom_____")
	senderStr, err := s.addressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)
	valStr, err := s.valAddressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)

	msg := &types.MsgRegisterValidator{
		Sender: senderStr,
		CreateValidator: stakingtypes.MsgCreateValidator{
			ValidatorAddress:  valStr,
			Value:             sdk.NewCoin("alyth", math.NewInt(1000)),
			MinSelfDelegation: math.NewInt(1000),
		},
		Burn: sdk.NewCoin("uatom", math.NewInt(100)), // wrong denom
	}

	_, err = s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrBurnDenomMismatch)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_BurnBelowRequired() {
	s.setNonTrivialParams(100, 1000)
	rawAddr := []byte("test_burn_below_____")
	msg := s.makeRegisterMsg(rawAddr, 50, 1000) // 50 < required 100

	_, err := s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrBurnBelowRequired)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_BurnAmountNil() {
	s.setNonTrivialParams(100, 1000)
	rawAddr := []byte("test_burn_nil_______")
	senderStr, err := s.addressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)
	valStr, err := s.valAddressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)

	msg := &types.MsgRegisterValidator{
		Sender: senderStr,
		CreateValidator: stakingtypes.MsgCreateValidator{
			ValidatorAddress:  valStr,
			Value:             sdk.NewCoin("alyth", math.NewInt(1000)),
			MinSelfDelegation: math.NewInt(1000),
		},
		Burn: sdk.Coin{Denom: "alyth"}, // Amount is nil
	}

	_, err = s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrBurnBelowRequired)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_DelegationDenomMismatch() {
	s.setNonTrivialParams(100, 1000)
	rawAddr := []byte("test_del_denom______")
	senderStr, err := s.addressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)
	valStr, err := s.valAddressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)

	msg := &types.MsgRegisterValidator{
		Sender: senderStr,
		CreateValidator: stakingtypes.MsgCreateValidator{
			ValidatorAddress:  valStr,
			Value:             sdk.NewCoin("uatom", math.NewInt(1000)), // wrong denom
			MinSelfDelegation: math.NewInt(1000),
		},
		Burn: sdk.NewCoin("alyth", math.NewInt(100)),
	}

	_, err = s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrDelegationDenomMismatch)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_MinSelfDelegationBelowRequired() {
	s.setNonTrivialParams(100, 1000)
	rawAddr := []byte("test_min_del_below__")
	msg := s.makeRegisterMsg(rawAddr, 100, 500) // 500 < required 1000

	_, err := s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrMinSelfDelegationBelowRequired)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_ValueAmountBelowRequired() {
	s.setNonTrivialParams(100, 1000)
	rawAddr := []byte("test_val_below______")
	senderStr, err := s.addressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)
	valStr, err := s.valAddressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)

	msg := &types.MsgRegisterValidator{
		Sender: senderStr,
		CreateValidator: stakingtypes.MsgCreateValidator{
			ValidatorAddress:  valStr,
			Value:             sdk.NewCoin("alyth", math.NewInt(500)), // below required
			MinSelfDelegation: math.NewInt(1000),                      // meets required
		},
		Burn: sdk.NewCoin("alyth", math.NewInt(100)),
	}

	_, err = s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrMinSelfDelegationBelowRequired)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_MinSelfDelegationFieldBelowRequired() {
	s.setNonTrivialParams(100, 1000)
	rawAddr := []byte("test_msd_field______")
	senderStr, err := s.addressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)
	valStr, err := s.valAddressCodec.BytesToString(rawAddr)
	s.Require().NoError(err)

	msg := &types.MsgRegisterValidator{
		Sender: senderStr,
		CreateValidator: stakingtypes.MsgCreateValidator{
			ValidatorAddress:  valStr,
			Value:             sdk.NewCoin("alyth", math.NewInt(1000)), // meets required
			MinSelfDelegation: math.NewInt(500),                        // below required
		},
		Burn: sdk.NewCoin("alyth", math.NewInt(100)),
	}

	_, err = s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrMinSelfDelegationBelowRequired)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_BurnFails() {
	s.setNonTrivialParams(100, 1000)
	rawAddr := []byte("test_burn_fails_____")
	msg := s.makeRegisterMsg(rawAddr, 100, 1000)
	senderAddr := sdk.AccAddress(rawAddr)

	burnErr := errors.New("burn keeper failed")
	s.burnKeeper.EXPECT().BurnFromAccount(gomock.Any(), senderAddr, msg.Burn).Return(burnErr)

	_, err := s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorIs(err, burnErr)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_CreateValidatorFails() {
	s.setNonTrivialParams(100, 1000)
	rawAddr := []byte("test_create_fails___")
	msg := s.makeRegisterMsg(rawAddr, 100, 1000)
	senderAddr := sdk.AccAddress(rawAddr)

	createErr := errors.New("create validator failed")
	gomock.InOrder(
		s.burnKeeper.EXPECT().BurnFromAccount(gomock.Any(), senderAddr, msg.Burn).Return(nil),
		s.stakingMsgServer.EXPECT().CreateValidator(gomock.Any(), &msg.CreateValidator).Return(nil, createErr),
	)

	_, err := s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().Error(err)
	s.Require().ErrorIs(err, createErr)
}

func (s *ValidatorKeeperTestSuite) TestRegisterValidator_OrderingEnforcement() {
	s.setNonTrivialParams(100, 1000)
	rawAddr := []byte("test_ordering_______")
	msg := s.makeRegisterMsg(rawAddr, 100, 1000)
	senderAddr := sdk.AccAddress(rawAddr)

	// Strict: Burn MUST come before CreateValidator
	gomock.InOrder(
		s.burnKeeper.EXPECT().BurnFromAccount(gomock.Any(), senderAddr, msg.Burn).Return(nil),
		s.stakingMsgServer.EXPECT().CreateValidator(gomock.Any(), &msg.CreateValidator).Return(&stakingtypes.MsgCreateValidatorResponse{}, nil),
	)

	resp, err := s.msgServer.RegisterValidator(s.ctx, msg)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
}

func (s *ValidatorKeeperTestSuite) TestUpdateParams_HappyPath() {
	newParams := types.NewParams(
		sdk.NewCoin("alyth", math.NewInt(200)),
		sdk.NewCoin("alyth", math.NewInt(5000)),
	)

	resp, err := s.msgServer.UpdateParams(s.ctx, &types.MsgUpdateParams{
		Authority: s.authority,
		Params:    newParams,
	})
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	got, err := s.validatorKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(newParams, got)
}

func (s *ValidatorKeeperTestSuite) TestUpdateParams_WrongAuthority() {
	_, err := s.msgServer.UpdateParams(s.ctx, &types.MsgUpdateParams{
		Authority: "mono1wrongauthority",
		Params:    types.DefaultParams(),
	})
	s.Require().Error(err)
	s.Require().ErrorIs(err, govtypes.ErrInvalidSigner)
}

func (s *ValidatorKeeperTestSuite) TestUpdateParams_InvalidParams() {
	// Registration burn with non-native denom
	_, err := s.msgServer.UpdateParams(s.ctx, &types.MsgUpdateParams{
		Authority: s.authority,
		Params: types.NewParams(
			sdk.NewCoin("uatom", math.NewInt(100)), // wrong denom
			sdk.NewCoin("alyth", math.NewInt(1000)),
		),
	})
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrInvalidRegistrationBurn)
}

func (s *ValidatorKeeperTestSuite) TestUpdateParams_ParamsSetFails() {
	sb := collections.NewSchemaBuilder(failSetStoreService{})
	s.validatorKeeper.Params = collections.NewItem(sb, types.ParamsKey, "params",
		codec.CollValue[types.Params](s.encCfg.Codec))
	_, _ = sb.Build()
	s.msgServer = keeper.NewMsgServerImpl(s.validatorKeeper)

	_, err := s.msgServer.UpdateParams(s.ctx, &types.MsgUpdateParams{
		Authority: s.authority,
		Params:    types.DefaultParams(),
	})
	s.Require().Error(err)
}
