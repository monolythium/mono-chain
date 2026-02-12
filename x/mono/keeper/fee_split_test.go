package keeper_test

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/core/address"
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

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/monolythium/mono-chain/app"
	"github.com/monolythium/mono-chain/x/mono/keeper"
	module "github.com/monolythium/mono-chain/x/mono/module"
	"github.com/monolythium/mono-chain/x/mono/types"
)

type modToModCall struct {
	SenderModule, RecipientModule string
	Amt                           sdk.Coins
}

type modToAccCall struct {
	SenderModule  string
	RecipientAddr sdk.AccAddress
	Amt           sdk.Coins
}

type burnCall struct {
	ModuleName string
	Amt        sdk.Coins
}

type feeSplitMockBank struct {
	balances map[string]sdk.Coins

	sendModToModCalls []modToModCall
	sendModToAccCalls []modToAccCall
	burnCoinsCalls    []burnCall

	sendModToModCallCount  int
	failSendModToModOnCall int // 0 = never, N = fail on Nth call

	burnCoinsCallCount  int
	failBurnCoinsOnCall int

	failSendModToAcc bool
}

func newFeeSplitMockBank() *feeSplitMockBank {
	return &feeSplitMockBank{
		balances: make(map[string]sdk.Coins),
	}
}

func (m *feeSplitMockBank) SpendableCoins(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	return m.balances[addr.String()]
}

func (m *feeSplitMockBank) GetAllBalances(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	return m.balances[addr.String()]
}

func (m *feeSplitMockBank) SendCoinsFromModuleToModule(_ context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	m.sendModToModCallCount++
	m.sendModToModCalls = append(m.sendModToModCalls, modToModCall{senderModule, recipientModule, amt})
	if m.failSendModToModOnCall > 0 && m.sendModToModCallCount == m.failSendModToModOnCall {
		return errors.New("mock: SendCoinsFromModuleToModule failed")
	}
	return nil
}

func (m *feeSplitMockBank) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	m.sendModToAccCalls = append(m.sendModToAccCalls, modToAccCall{senderModule, recipientAddr, amt})
	if m.failSendModToAcc {
		return errors.New("mock: SendCoinsFromModuleToAccount failed")
	}
	return nil
}

func (m *feeSplitMockBank) BurnCoins(_ context.Context, moduleName string, amt sdk.Coins) error {
	m.burnCoinsCallCount++
	m.burnCoinsCalls = append(m.burnCoinsCalls, burnCall{moduleName, amt})
	if m.failBurnCoinsOnCall > 0 && m.burnCoinsCallCount == m.failBurnCoinsOnCall {
		return errors.New("mock: BurnCoins failed")
	}
	return nil
}

type feeSplitMockStaking struct {
	validators map[string]stakingtypes.Validator // consAddr string → validator
	failLookup bool
	returnNil  bool
	addrCodec  address.Codec
}

func newFeeSplitMockStaking() *feeSplitMockStaking {
	return &feeSplitMockStaking{
		validators: make(map[string]stakingtypes.Validator),
		addrCodec:  addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

func (m *feeSplitMockStaking) ConsensusAddressCodec() address.Codec {
	return m.addrCodec
}

func (m *feeSplitMockStaking) ValidatorByConsAddr(_ context.Context, addr sdk.ConsAddress) (stakingtypes.ValidatorI, error) {
	if m.failLookup {
		return nil, errors.New("mock: validator lookup failed")
	}
	if m.returnNil {
		return nil, nil
	}
	if val, ok := m.validators[addr.String()]; ok {
		return val, nil
	}
	return nil, errors.New("mock: validator not found")
}

type feeSplitFixture struct {
	ctx         sdk.Context
	keeper      keeper.Keeper
	mockBank    *feeSplitMockBank
	mockStaking *feeSplitMockStaking
	addrCodec   address.Codec
}

func initFeeSplitFixture(t *testing.T, mockBank *feeSplitMockBank, mockStaking *feeSplitMockStaking) *feeSplitFixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	require.Equal(t, "mono", sdk.GetConfig().GetBech32AccountAddrPrefix())
	ac := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		ac,
		authority,
		mockBank,
		mockStaking,
	)

	return &feeSplitFixture{
		ctx:         ctx,
		keeper:      k,
		mockBank:    mockBank,
		mockStaking: mockStaking,
		addrCodec:   ac,
	}
}

// setParams sets params on the keeper and fatals on error
func (f *feeSplitFixture) setParams(t *testing.T, feeBurnPercent math.LegacyDec, regFee sdk.Coin) {
	t.Helper()
	err := f.keeper.Params.Set(f.ctx, types.NewParams(feeBurnPercent, regFee))
	require.NoError(t, err)
}

// setFeeCollectorBalance sets the mock balance for the fee_collector address
func (f *feeSplitFixture) setFeeCollectorBalance(coins sdk.Coins) {
	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	f.mockBank.balances[feeCollectorAddr.String()] = coins
}

// withProposer returns a context with the given proposer consensus address in the header
func (f *feeSplitFixture) withProposer(consAddr sdk.ConsAddress) sdk.Context {
	return f.ctx.WithBlockHeader(cmtproto.Header{
		ProposerAddress: consAddr,
	})
}

// proposerAccAddr derives the expected account address from a validator address
func proposerAccAddr(valAddr sdk.ValAddress) sdk.AccAddress {
	return sdk.AccAddress(valAddr)
}

func TestProcessFeeSplit_EmptyFeeCollector(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	// Fee collector has no balance
	err := f.keeper.ProcessFeeSplit(f.ctx)

	require.NoError(t, err)
	require.Empty(t, mockBank.sendModToModCalls, "no transfers should occur")
	require.Empty(t, mockBank.burnCoinsCalls, "no burns should occur")
	require.Empty(t, mockBank.sendModToAccCalls, "no sends should occur")
}

func TestProcessFeeSplit_NormalSplit(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	// Setup fee collector with 1000 alyth
	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))))

	// Setup proposer
	consAddr := sdk.ConsAddress("proposer_cons_addr_1")
	valAddr := sdk.ValAddress("proposer_val_addr__1")
	mockStaking.validators[consAddr.String()] = stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.NoError(t, err)

	// Verify burn: 1000 * 0.90 = 900
	require.Len(t, mockBank.burnCoinsCalls, 1)
	require.Equal(t, types.ModuleName, mockBank.burnCoinsCalls[0].ModuleName)
	require.Equal(t, math.NewInt(900), mockBank.burnCoinsCalls[0].Amt.AmountOf(app.DefaultBondDenom))

	// Verify burn transfer went through mono module
	require.Len(t, mockBank.sendModToModCalls, 1)
	require.Equal(t, authtypes.FeeCollectorName, mockBank.sendModToModCalls[0].SenderModule)
	require.Equal(t, types.ModuleName, mockBank.sendModToModCalls[0].RecipientModule)

	// Verify proposer got remainder: 1000 - 900 = 100
	require.Len(t, mockBank.sendModToAccCalls, 1)
	require.Equal(t, authtypes.FeeCollectorName, mockBank.sendModToAccCalls[0].SenderModule)
	require.Equal(t, proposerAccAddr(valAddr), mockBank.sendModToAccCalls[0].RecipientAddr)
	require.Equal(t, math.NewInt(100), mockBank.sendModToAccCalls[0].Amt.AmountOf(app.DefaultBondDenom))
}

func TestProcessFeeSplit_FullBurn(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyOneDec(), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt())) // 100% burn

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(500))))

	consAddr := sdk.ConsAddress("proposer_cons_full_1")
	valAddr := sdk.ValAddress("proposer_val_full__1")
	mockStaking.validators[consAddr.String()] = stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.NoError(t, err)

	// All 500 burned
	require.Len(t, mockBank.burnCoinsCalls, 1)
	require.Equal(t, math.NewInt(500), mockBank.burnCoinsCalls[0].Amt.AmountOf(app.DefaultBondDenom))

	// Nothing sent to proposer
	require.Empty(t, mockBank.sendModToAccCalls, "proposer should receive nothing at 100% burn")
}

func TestProcessFeeSplit_ZeroBurn(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyZeroDec(), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt())) // 0% burn

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(500))))

	consAddr := sdk.ConsAddress("proposer_cons_zero_1")
	valAddr := sdk.ValAddress("proposer_val_zero__1")
	mockStaking.validators[consAddr.String()] = stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.NoError(t, err)

	// Nothing burned
	require.Empty(t, mockBank.burnCoinsCalls, "nothing should be burned at 0%")
	require.Empty(t, mockBank.sendModToModCalls, "no module-to-module transfer needed")

	// All 500 to proposer
	require.Len(t, mockBank.sendModToAccCalls, 1)
	require.Equal(t, math.NewInt(500), mockBank.sendModToAccCalls[0].Amt.AmountOf(app.DefaultBondDenom))
}

func TestProcessFeeSplit_NoProposerAddress(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))))

	// Empty proposer address (genesis block) — use default context with no proposer
	err := f.keeper.ProcessFeeSplit(f.ctx)

	require.NoError(t, err)

	// Main burn: 900
	// Fallback burn of proposer share: 100 (since no proposer to pay)
	// Total BurnCoins calls: 2 (main burn + fallback burn)
	require.Len(t, mockBank.burnCoinsCalls, 2)
	require.Equal(t, math.NewInt(900), mockBank.burnCoinsCalls[0].Amt.AmountOf(app.DefaultBondDenom))
	require.Equal(t, math.NewInt(100), mockBank.burnCoinsCalls[1].Amt.AmountOf(app.DefaultBondDenom))

	// Two SendModToMod calls: one for main burn, one for fallback burn
	require.Len(t, mockBank.sendModToModCalls, 2)

	// No direct send to any account
	require.Empty(t, mockBank.sendModToAccCalls)
}

func TestProcessFeeSplit_ProposerLookupFails(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	mockStaking.failLookup = true
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))))

	consAddr := sdk.ConsAddress("proposer_cons_fail_1")
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.NoError(t, err, "should not error — graceful fallback to burn")

	// Main burn (900) + fallback burn of proposer share (100)
	require.Len(t, mockBank.burnCoinsCalls, 2)
	require.Equal(t, math.NewInt(900), mockBank.burnCoinsCalls[0].Amt.AmountOf(app.DefaultBondDenom))
	require.Equal(t, math.NewInt(100), mockBank.burnCoinsCalls[1].Amt.AmountOf(app.DefaultBondDenom))

	require.Empty(t, mockBank.sendModToAccCalls, "no funds should reach an account")
}

func TestProcessFeeSplit_NilValidator(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	mockStaking.returnNil = true
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(200))))

	consAddr := sdk.ConsAddress("proposer_cons_nil__1")
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.NoError(t, err, "nil validator should trigger fallback burn")

	// Main burn: 180, fallback burn: 20
	require.Len(t, mockBank.burnCoinsCalls, 2)
	require.Equal(t, math.NewInt(180), mockBank.burnCoinsCalls[0].Amt.AmountOf(app.DefaultBondDenom))
	require.Equal(t, math.NewInt(20), mockBank.burnCoinsCalls[1].Amt.AmountOf(app.DefaultBondDenom))
}

func TestProcessFeeSplit_BurnTransferFails(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockBank.failSendModToModOnCall = 1 // fail on first SendModToMod (main burn transfer)
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))))

	consAddr := sdk.ConsAddress("proposer_cons_bxf_1")
	valAddr := sdk.ValAddress("proposer_val_bxf__1")
	mockStaking.validators[consAddr.String()] = stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrFeeSplitFailed)
	require.Contains(t, err.Error(), "transfer to burn intermediary failed")

	// No burns should have occurred
	require.Empty(t, mockBank.burnCoinsCalls)
	// No proposer payment
	require.Empty(t, mockBank.sendModToAccCalls)
}

func TestProcessFeeSplit_BurnCoinsFails(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockBank.failBurnCoinsOnCall = 1 // fail on first BurnCoins (after successful transfer)
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))))

	consAddr := sdk.ConsAddress("proposer_cons_bcf_1")
	valAddr := sdk.ValAddress("proposer_val_bcf__1")
	mockStaking.validators[consAddr.String()] = stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrFeeSplitFailed)
	require.Contains(t, err.Error(), "burn failed after transfer")

	// Transfer happened but burn failed
	require.Len(t, mockBank.sendModToModCalls, 1)
	require.Len(t, mockBank.burnCoinsCalls, 1)
	// Proposer never reached
	require.Empty(t, mockBank.sendModToAccCalls)
}

func TestProcessFeeSplit_ProposerSendFails(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockBank.failSendModToAcc = true
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))))

	consAddr := sdk.ConsAddress("proposer_cons_psf_1")
	valAddr := sdk.ValAddress("proposer_val_psf__1")
	mockStaking.validators[consAddr.String()] = stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrFeeSplitFailed)
	require.Contains(t, err.Error(), "send to proposer failed")

	// Burn succeeded
	require.Len(t, mockBank.burnCoinsCalls, 1)
	require.Equal(t, math.NewInt(900), mockBank.burnCoinsCalls[0].Amt.AmountOf(app.DefaultBondDenom))
}

func TestProcessFeeSplit_MultiDenom(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	// Fee collector has two denoms
	f.setFeeCollectorBalance(sdk.NewCoins(
		sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000)),
		sdk.NewCoin("other", math.NewInt(200)),
	))

	consAddr := sdk.ConsAddress("proposer_cons_multi")
	valAddr := sdk.ValAddress("proposer_val_multi_")
	mockStaking.validators[consAddr.String()] = stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.NoError(t, err)

	// Burn: 900 alyth + 180 other
	burnAmt := mockBank.burnCoinsCalls[0].Amt
	require.Equal(t, math.NewInt(900), burnAmt.AmountOf(app.DefaultBondDenom))
	require.Equal(t, math.NewInt(180), burnAmt.AmountOf("other"))

	// Proposer: 100 alyth + 20 other
	proposerAmt := mockBank.sendModToAccCalls[0].Amt
	require.Equal(t, math.NewInt(100), proposerAmt.AmountOf(app.DefaultBondDenom))
	require.Equal(t, math.NewInt(20), proposerAmt.AmountOf("other"))
}

func TestProcessFeeSplit_RoundingDust(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	// 99 * 0.90 = 89.1 → truncated to 89
	// proposer gets 99 - 89 = 10 (not 9.9)
	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(99))))

	consAddr := sdk.ConsAddress("proposer_cons_rnd__1")
	valAddr := sdk.ValAddress("proposer_val_rnd___1")
	mockStaking.validators[consAddr.String()] = stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.NoError(t, err)

	burnAmt := mockBank.burnCoinsCalls[0].Amt.AmountOf(app.DefaultBondDenom)
	proposerAmt := mockBank.sendModToAccCalls[0].Amt.AmountOf(app.DefaultBondDenom)

	require.Equal(t, math.NewInt(89), burnAmt, "burn should be truncated")
	require.Equal(t, math.NewInt(10), proposerAmt, "proposer gets remainder")

	// Critical invariant: burn + proposer == total (no dust lost)
	require.Equal(t, math.NewInt(99), burnAmt.Add(proposerAmt), "total must be conserved")
}

func TestProcessFeeSplit_SingleUnit(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	// 1 * 0.90 = 0.9 → truncated to 0
	// proposer gets 1 - 0 = 1 (all of it)
	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1))))

	consAddr := sdk.ConsAddress("proposer_cons_sgl__1")
	valAddr := sdk.ValAddress("proposer_val_sgl___1")
	mockStaking.validators[consAddr.String()] = stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.NoError(t, err)

	// Burn is 0 — no burn should occur
	require.Empty(t, mockBank.burnCoinsCalls, "zero burn should be skipped")
	require.Empty(t, mockBank.sendModToModCalls, "no module transfer for zero burn")

	// Proposer gets the single unit
	require.Len(t, mockBank.sendModToAccCalls, 1)
	require.Equal(t, math.NewInt(1), mockBank.sendModToAccCalls[0].Amt.AmountOf(app.DefaultBondDenom))
}

func TestProcessFeeSplit_FallbackBurnTransferFails(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	// First SendModToMod succeeds (main burn), second fails (fallback burn)
	mockBank.failSendModToModOnCall = 2
	mockStaking := newFeeSplitMockStaking()
	mockStaking.failLookup = true // force proposer resolution failure → fallback path
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))))

	consAddr := sdk.ConsAddress("proposer_cons_fbt_1")
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrFeeSplitFailed)
	require.Contains(t, err.Error(), "fallback burn transfer failed")

	// Main burn succeeded
	require.Len(t, mockBank.burnCoinsCalls, 1)
	require.Equal(t, math.NewInt(900), mockBank.burnCoinsCalls[0].Amt.AmountOf(app.DefaultBondDenom))

	// Fallback transfer failed — second burn never reached
	require.Len(t, mockBank.sendModToModCalls, 2) // both attempted
}

func TestProcessFeeSplit_FallbackBurnCoinsFails(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	// First BurnCoins succeeds (main burn), second fails (fallback burn)
	mockBank.failBurnCoinsOnCall = 2
	mockStaking := newFeeSplitMockStaking()
	mockStaking.failLookup = true // force fallback path
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	f.setParams(t, math.LegacyNewDecWithPrec(90, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))))

	consAddr := sdk.ConsAddress("proposer_cons_fbc_1")
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrFeeSplitFailed)
	require.Contains(t, err.Error(), "fallback burn failed")

	// Main burn succeeded, fallback burn failed
	require.Len(t, mockBank.burnCoinsCalls, 2)
	require.Equal(t, math.NewInt(900), mockBank.burnCoinsCalls[0].Amt.AmountOf(app.DefaultBondDenom))
}

func TestProcessFeeSplit_ParamsNotSet(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)
	// Deliberately do NOT set params

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))))

	err := f.keeper.ProcessFeeSplit(f.ctx)

	require.Error(t, err, "should fail when params not initialized")
	require.Contains(t, err.Error(), "failed to get mono params")
}

func TestProcessFeeSplit_HighPrecisionPercent(t *testing.T) {
	mockBank := newFeeSplitMockBank()
	mockStaking := newFeeSplitMockStaking()
	f := initFeeSplitFixture(t, mockBank, mockStaking)

	// 33.333...% burn — tests non-trivial decimal precision
	f.setParams(t, math.LegacyNewDecWithPrec(33, 2), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()))

	f.setFeeCollectorBalance(sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100))))

	consAddr := sdk.ConsAddress("proposer_cons_hp___1")
	valAddr := sdk.ValAddress("proposer_val_hp____1")
	mockStaking.validators[consAddr.String()] = stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}
	ctx := f.withProposer(consAddr)

	err := f.keeper.ProcessFeeSplit(ctx)

	require.NoError(t, err)

	// 100 * 0.33 = 33.0 → 33 burned
	// Proposer gets 100 - 33 = 67
	burnAmt := mockBank.burnCoinsCalls[0].Amt.AmountOf(app.DefaultBondDenom)
	proposerAmt := mockBank.sendModToAccCalls[0].Amt.AmountOf(app.DefaultBondDenom)

	require.Equal(t, math.NewInt(33), burnAmt)
	require.Equal(t, math.NewInt(67), proposerAmt)
	require.Equal(t, math.NewInt(100), burnAmt.Add(proposerAmt), "total must be conserved")
}
