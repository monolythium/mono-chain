package ante_test

import (
	"bytes"
	"os"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	"github.com/monolythium/mono-chain/app"
	burnmoduletypes "github.com/monolythium/mono-chain/x/burn/types"
	monoante "github.com/monolythium/mono-chain/x/mono/ante"
	"github.com/monolythium/mono-chain/x/mono/keeper"
	module "github.com/monolythium/mono-chain/x/mono/module"
	"github.com/monolythium/mono-chain/x/mono/types"
)

type mockTx struct {
	msgs []sdk.Msg
}

func (m mockTx) GetMsgs() []sdk.Msg                    { return m.msgs }
func (m mockTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

func noopAnteHandler(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
	return ctx, nil
}

var VALIDATOR_LYTH_REQUIREMENT = math.NewIntWithDecimal(100_000, 18)

type depositFixture struct {
	ctx       sdk.Context
	keeper    keeper.Keeper
	decorator monoante.ValidatorRegistrationBurnDecorator
}

func initDepositFixture(t *testing.T, registrationFee sdk.Coin) *depositFixture {
	t.Helper()

	require.Equal(t, "mono", sdk.GetConfig().GetBech32AccountAddrPrefix())
	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx.WithBlockHeight(2)

	authority := authtypes.NewModuleAddress(types.GovModuleName)

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		nil,
		nil,
		nil,
	)

	params := types.NewParams(
		math.LegacyNewDecWithPrec(90, 2),
		registrationFee,
	)
	require.NoError(t, k.Params.Set(ctx, params))

	return &depositFixture{
		ctx:    ctx,
		keeper: k,
		decorator: monoante.NewValidatorRegistrationBurnDecorator(
			k,
			addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
			addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		),
	}
}

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	app.SetBech32Prefixes(cfg)
	app.SetBip44CoinType(cfg)
	os.Exit(m.Run())
}

// Helper: build a MsgCreateValidator from raw address bytes
func makeCreateValidatorMsg(addrBytes []byte) *stakingtypes.MsgCreateValidator {
	valAddr := sdk.ValAddress(addrBytes)
	return &stakingtypes.MsgCreateValidator{
		ValidatorAddress:  valAddr.String(),
		MinSelfDelegation: VALIDATOR_LYTH_REQUIREMENT,
		Value:             sdk.NewCoin(sdk.DefaultBondDenom, VALIDATOR_LYTH_REQUIREMENT),
	}
}

// Helper: build a MsgBurn from raw address bytes
func makeBurnMsg(addrBytes []byte, amount math.Int) *burnmoduletypes.MsgBurn {
	accAddr := sdk.AccAddress(addrBytes)
	return &burnmoduletypes.MsgBurn{
		FromAddress: accAddr.String(),
		Amount:      sdk.NewCoin(sdk.DefaultBondDenom, amount),
	}
}

var (
	addr1  = bytes.Repeat([]byte{0x01}, 20)
	addr2  = bytes.Repeat([]byte{0x02}, 20)
	regFee = sdk.NewCoin(sdk.DefaultBondDenom, VALIDATOR_LYTH_REQUIREMENT)
)

func TestValidatorRegistrationBurn_NoCreateValidator(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT)}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.NoError(t, err)
}

func TestValidatorRegistrationBurn_ValidPair(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{
		makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT),
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.NoError(t, err)
}

func TestValidatorRegistrationBurn_ExcessBurnAmount(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{
		makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT.Mul(math.NewInt(5))),
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.NoError(t, err)
}

func TestValidatorRegistrationBurn_ExactAmount(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{
		makeCreateValidatorMsg(addr1),
		makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.NoError(t, err)
}

func TestValidatorRegistrationBurn_MissingBurn(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrMissingBurnInfo)
}

func TestValidatorRegistrationBurn_GenesisHeightBypass(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{
		makeCreateValidatorMsg(addr1),
	}}

	_, err := f.decorator.AnteHandle(f.ctx.WithBlockHeight(0), tx, false, noopAnteHandler)
	require.NoError(t, err)
}

func TestValidatorRegistrationBurn_HeightOneIsEnforced(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{
		makeCreateValidatorMsg(addr1),
	}}

	_, err := f.decorator.AnteHandle(f.ctx.WithBlockHeight(1), tx, false, noopAnteHandler)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrMissingBurnInfo)
}

func TestValidatorRegistrationBurn_InsufficientBurn(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{
		makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT.Sub(math.OneInt())),
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInsufficientBurnAmount)
}

func TestValidatorRegistrationBurn_WrongDenom(t *testing.T) {
	f := initDepositFixture(t, regFee)

	accAddr := sdk.AccAddress(addr1)
	wrongDenomBurn := &burnmoduletypes.MsgBurn{
		FromAddress: accAddr.String(),
		Amount:      sdk.NewCoin("uatom", VALIDATOR_LYTH_REQUIREMENT),
	}

	tx := mockTx{msgs: []sdk.Msg{
		wrongDenomBurn,
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrBurnDenomMismatch)
}

func TestValidatorRegistrationBurn_ZeroFeeDisabled(t *testing.T) {
	zeroFee := sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())
	f := initDepositFixture(t, zeroFee)

	// MsgCreateValidator with no burn should pass when fee is zero
	tx := mockTx{msgs: []sdk.Msg{
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.NoError(t, err)
}

func TestValidatorRegistrationBurn_BurnFromAddr2_ValidatorAddr1(t *testing.T) {
	f := initDepositFixture(t, regFee)

	// addr2 burns enough, but addr1 creates validator
	// should fail with sender mismatch
	tx := mockTx{msgs: []sdk.Msg{
		makeBurnMsg(addr2, VALIDATOR_LYTH_REQUIREMENT.Mul(math.NewInt(2))),
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrBurnSenderMismatch)
}

func TestValidatorRegistrationBurn_SimulationEnforced_ValidPair(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{
		makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT),
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, true, noopAnteHandler)

	require.NoError(t, err)
}

func TestValidatorRegistrationBurn_SimulationEnforced_MissingBurn(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, true, noopAnteHandler)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrMissingBurnInfo)
}

func TestValidatorRegistrationBurn_EmptyValidatorAddress(t *testing.T) {
	f := initDepositFixture(t, regFee)

	badMsg := &stakingtypes.MsgCreateValidator{
		ValidatorAddress: "",
	}
	tx := mockTx{msgs: []sdk.Msg{
		makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT),
		badMsg,
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidValidatorAddress)
}

func TestValidatorRegistrationBurn_MalformedBurnAddress(t *testing.T) {
	f := initDepositFixture(t, regFee)

	badBurn := &burnmoduletypes.MsgBurn{
		FromAddress: "not_a_valid_bech32",
		Amount:      sdk.NewCoin(sdk.DefaultBondDenom, VALIDATOR_LYTH_REQUIREMENT),
	}
	tx := mockTx{msgs: []sdk.Msg{
		badBurn,
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidBurnAddress)
}

func TestValidatorRegistrationBurn_DuplicateCreateValidator(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{
		makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT),
		makeCreateValidatorMsg(addr1),
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrDuplicateRegistrationInfo)
}

func TestValidatorRegistrationBurn_DuplicateBurn(t *testing.T) {
	f := initDepositFixture(t, regFee)

	tx := mockTx{msgs: []sdk.Msg{
		makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT),
		makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT),
		makeCreateValidatorMsg(addr1),
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrDuplicateRegistrationInfo)
}

func TestValidatorRegistrationBurn_MinSelfDelegationBelowFee(t *testing.T) {
	f := initDepositFixture(t, regFee)

	lowSelfDelegation := &stakingtypes.MsgCreateValidator{
		ValidatorAddress:  sdk.ValAddress(addr1).String(),
		MinSelfDelegation: VALIDATOR_LYTH_REQUIREMENT.Sub(math.OneInt()),
		Value:             sdk.NewCoin(sdk.DefaultBondDenom, VALIDATOR_LYTH_REQUIREMENT),
	}
	tx := mockTx{msgs: []sdk.Msg{
		makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT),
		lowSelfDelegation,
	}}
	_, err := f.decorator.AnteHandle(f.ctx, tx, false, noopAnteHandler)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInsufficientMinSelfDelegation)
}

func TestValidatorRegistrationBurn_ParamsReadFailure(t *testing.T) {
	// Create a keeper with a store but do NOT set params.
	require.Equal(t, "mono", sdk.GetConfig().GetBech32AccountAddrPrefix())
	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx.WithBlockHeight(2)

	authority := authtypes.NewModuleAddress(types.GovModuleName)
	k := keeper.NewKeeper(storeService, encCfg.Codec, addressCodec, authority, nil, nil, nil)

	decorator := monoante.NewValidatorRegistrationBurnDecorator(
		k,
		addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
	)

	tx := mockTx{msgs: []sdk.Msg{
		makeBurnMsg(addr1, VALIDATOR_LYTH_REQUIREMENT),
		makeCreateValidatorMsg(addr1),
	}}
	_, err := decorator.AnteHandle(ctx, tx, false, noopAnteHandler)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrParamsRead)
}
