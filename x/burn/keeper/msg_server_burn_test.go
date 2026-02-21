package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/app"
	"github.com/monolythium/mono-chain/x/burn/keeper"
	module "github.com/monolythium/mono-chain/x/burn/module"
	"github.com/monolythium/mono-chain/x/burn/types"
)

// mockBankKeeper implements types.BankKeeper for unit testing
type mockBankKeeper struct {
	balances  map[string]sdk.Coin
	supplies  map[string]sdk.Coin
	spendable map[string]sdk.Coins

	failSendCoins bool
	failBurnCoins bool

	sendCoinsCalled bool
	burnCoinsCalled bool

	// For simulating post-burn verification failures
	getBalanceCalls     int
	getSupplyCalls      int
	balanceOnSecondCall sdk.Coin // If set, return this on 2nd GetBalance call
	supplyOnSecondCall  sdk.Coin // If set, return this on 2nd GetSupply call
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances:  make(map[string]sdk.Coin),
		supplies:  make(map[string]sdk.Coin),
		spendable: make(map[string]sdk.Coins),
	}
}

func (m *mockBankKeeper) SpendableCoins(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	if coins, ok := m.spendable[addr.String()]; ok {
		return coins
	}
	return sdk.Coins{}
}

func (m *mockBankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	m.getBalanceCalls++
	// Simulate failure: return wrong value on second call if override is set
	if m.getBalanceCalls == 2 && !m.balanceOnSecondCall.IsNil() && m.balanceOnSecondCall.Denom == denom {
		return m.balanceOnSecondCall
	}
	if coin, ok := m.balances[addr.String()]; ok && coin.Denom == denom {
		return coin
	}
	return sdk.NewCoin(denom, math.ZeroInt())
}

func (m *mockBankKeeper) GetSupply(_ context.Context, denom string) sdk.Coin {
	m.getSupplyCalls++
	// Simulate failure: return wrong value on second call if override is set
	if m.getSupplyCalls == 2 && !m.supplyOnSecondCall.IsNil() && m.supplyOnSecondCall.Denom == denom {
		return m.supplyOnSecondCall
	}
	if supply, ok := m.supplies[denom]; ok {
		return supply
	}
	return sdk.NewCoin(denom, math.ZeroInt())
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, _ string, amt sdk.Coins) error {
	m.sendCoinsCalled = true
	if m.failSendCoins {
		return sdkerrors.ErrInsufficientFunds.Wrap("mock failure")
	}
	coin := amt[0]
	current := m.balances[senderAddr.String()]
	m.balances[senderAddr.String()] = sdk.NewCoin(current.Denom, current.Amount.Sub(coin.Amount))
	return nil
}

func (m *mockBankKeeper) BurnCoins(_ context.Context, _ string, amt sdk.Coins) error {
	m.burnCoinsCalled = true
	if m.failBurnCoins {
		return sdkerrors.ErrUnauthorized.Wrap("mock failure")
	}
	coin := amt[0]
	current := m.supplies[coin.Denom]
	m.supplies[coin.Denom] = sdk.NewCoin(coin.Denom, current.Amount.Sub(coin.Amount))
	return nil
}

type burnFixture struct {
	ctx          context.Context
	keeper       keeper.Keeper
	msgServer    types.MsgServer
	addressCodec address.Codec
	mockBank     *mockBankKeeper
}

func initBurnFixture(t *testing.T, mockBank *mockBankKeeper) *burnFixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	require.Equal(t, "mono", sdk.GetConfig().GetBech32AccountAddrPrefix())
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		mockBank,
	)

	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &burnFixture{
		ctx:          ctx,
		keeper:       k,
		msgServer:    keeper.NewMsgServerImpl(k),
		addressCodec: addressCodec,
		mockBank:     mockBank,
	}
}

func TestMsgServerBurn_ValidBurn(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	addr := sdk.AccAddress("test_address_valid_1")
	burnAmount := sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100))

	mockBank.balances[addr.String()] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))
	mockBank.spendable[addr.String()] = sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000)))
	mockBank.supplies[app.DefaultBondDenom] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(10000))

	msg := &types.MsgBurn{
		FromAddress: addr.String(),
		Amount:      burnAmount,
	}

	resp, err := f.msgServer.Burn(f.ctx, msg)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, mockBank.sendCoinsCalled)
	require.True(t, mockBank.burnCoinsCalled)
	require.Equal(t, math.NewInt(900), mockBank.balances[addr.String()].Amount)
	require.Equal(t, math.NewInt(9900), mockBank.supplies[app.DefaultBondDenom].Amount)
}

func TestMsgServerBurn_InsufficientSpendable(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	addr := sdk.AccAddress("test_address_insuf_1")

	// Has balance but not spendable (vesting/locked)
	mockBank.balances[addr.String()] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))
	mockBank.spendable[addr.String()] = sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(50)))
	mockBank.supplies[app.DefaultBondDenom] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(10000))

	msg := &types.MsgBurn{
		FromAddress: addr.String(),
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)),
	}

	_, err := f.msgServer.Burn(f.ctx, msg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient")
	require.False(t, mockBank.sendCoinsCalled)
}

func TestMsgServerBurn_StateCorruption_BalanceExceedsSupply(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	addr := sdk.AccAddress("test_address_corrupt1")

	// Corrupted: balance > supply
	mockBank.balances[addr.String()] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(2000))
	mockBank.spendable[addr.String()] = sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(2000)))
	mockBank.supplies[app.DefaultBondDenom] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))

	msg := &types.MsgBurn{
		FromAddress: addr.String(),
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)),
	}

	_, err := f.msgServer.Burn(f.ctx, msg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrStateCorruption)
}

func TestMsgServerBurn_SupplyUnderflow(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	addr := sdk.AccAddress("test_address_under_1")

	// User has 100, total supply is 150 (valid state)
	// But trying to burn 200 (more than supply)
	// This tests the supply underflow check at line 48
	mockBank.balances[addr.String()] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100))
	mockBank.spendable[addr.String()] = sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)))
	mockBank.supplies[app.DefaultBondDenom] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(150))

	msg := &types.MsgBurn{
		FromAddress: addr.String(),
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(200)),
	}

	_, err := f.msgServer.Burn(f.ctx, msg)

	// Should fail on spendable check first since user only has 100
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient")
	require.False(t, mockBank.sendCoinsCalled)
}

// Test when supply is less than burn amount but user has enough
func TestMsgServerBurn_SupplyLessThanBurn(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	addr1 := sdk.AccAddress("user1")

	// Setup: addr1 has 100, addr2 has 100, total supply is 200 (valid)
	// addr1 will try to burn 150 (has enough via spendable, but exceeds their balance)
	// This is tricky - we need a scenario where spendable > balance (vesting account)
	// Actually, let's simulate: total supply is somehow less than what user wants to burn

	// Simpler: Multiple users, total supply 100, one user has 100
	// User tries to burn 150 - has it spendable but supply is only 100
	mockBank.balances[addr1.String()] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100))
	mockBank.spendable[addr1.String()] = sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(150))) // Spendable > balance (edge case)
	mockBank.supplies[app.DefaultBondDenom] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100))

	msg := &types.MsgBurn{
		FromAddress: addr1.String(),
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(150)),
	}

	_, err := f.msgServer.Burn(f.ctx, msg)

	// Should fail on supply check
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrSupplyUnderflow)
	require.False(t, mockBank.sendCoinsCalled)
}

func TestMsgServerBurn_BurnCoinsFailure_ShouldError(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	addr := sdk.AccAddress("test_address_fail_1")

	mockBank.balances[addr.String()] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))
	mockBank.spendable[addr.String()] = sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000)))
	mockBank.supplies[app.DefaultBondDenom] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(10000))
	mockBank.failBurnCoins = true

	msg := &types.MsgBurn{
		FromAddress: addr.String(),
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)),
	}

	_, err := f.msgServer.Burn(f.ctx, msg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrBurnFailed)
	require.True(t, mockBank.sendCoinsCalled)
}

func TestMsgServerBurn_InvalidDenom(t *testing.T) {
	msg := &types.MsgBurn{
		FromAddress: sdk.AccAddress("test").String(),
		Amount:      sdk.NewCoin("uatom", math.NewInt(100)),
	}

	err := msg.ValidateBasic()
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidBurnDenom)
}

func TestMsgServerBurn_ZeroAmount(t *testing.T) {
	msg := &types.MsgBurn{
		FromAddress: sdk.AccAddress("test").String(),
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(0)),
	}

	err := msg.ValidateBasic()
	require.Error(t, err)
}

func TestMsgServerBurn_BurnEntireBalance(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	addr := sdk.AccAddress("test_address_entire1")
	balance := sdk.NewCoin(app.DefaultBondDenom, math.NewInt(500))

	mockBank.balances[addr.String()] = balance
	mockBank.spendable[addr.String()] = sdk.NewCoins(balance)
	mockBank.supplies[app.DefaultBondDenom] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(10000))

	msg := &types.MsgBurn{
		FromAddress: addr.String(),
		Amount:      balance,
	}

	resp, err := f.msgServer.Burn(f.ctx, msg)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, mockBank.balances[addr.String()].IsZero())
	require.Equal(t, math.NewInt(9500), mockBank.supplies[app.DefaultBondDenom].Amount)
}

func TestMsgServerBurn_InvalidAddress(t *testing.T) {
	msg := &types.MsgBurn{
		FromAddress: "invalid_address",
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)),
	}

	err := msg.ValidateBasic()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid")
}

func TestMsgServerBurn_NegativeAmount(t *testing.T) {
	// Cosmos SDK Coin doesn't allow negative amounts in construction
	// but we can test the validation path
	msg := &types.MsgBurn{
		FromAddress: sdk.AccAddress("test").String(),
		Amount:      sdk.Coin{Denom: app.DefaultBondDenom, Amount: math.NewInt(-100)},
	}

	err := msg.ValidateBasic()
	require.Error(t, err)
}

// Test NewMsgBurn constructor
func TestNewMsgBurn_Constructor(t *testing.T) {
	addr := sdk.AccAddress("test").String()
	coin := sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100))

	msg := types.NewMsgBurn(addr, coin)

	require.Equal(t, addr, msg.FromAddress)
	require.Equal(t, coin, msg.Amount)
}

// Test GetSigners returns correct address
func TestMsgBurn_GetSigners_Valid(t *testing.T) {
	addr := sdk.AccAddress("signer")
	msg := types.NewMsgBurn(addr.String(), sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)))

	signers := msg.GetSigners()

	require.Len(t, signers, 1)
	require.Equal(t, addr, signers[0])
}

// Test GetSigners panics on invalid address
func TestMsgBurn_GetSigners_InvalidPanics(t *testing.T) {
	msg := &types.MsgBurn{
		FromAddress: "invalid",
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)),
	}

	require.Panics(t, func() {
		msg.GetSigners()
	})
}

func TestMsgServerBurn_InvalidFromAddress(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	msg := &types.MsgBurn{
		FromAddress: "not_valid_bech32",
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)),
	}

	_, err := f.msgServer.Burn(f.ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid from address")
}

func TestMsgServerBurn_SendCoinsFailure(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	addr := sdk.AccAddress("test_address_sendf_1")

	mockBank.balances[addr.String()] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))
	mockBank.spendable[addr.String()] = sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000)))
	mockBank.supplies[app.DefaultBondDenom] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(10000))
	mockBank.failSendCoins = true

	msg := &types.MsgBurn{
		FromAddress: addr.String(),
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)),
	}

	_, err := f.msgServer.Burn(f.ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient")
	require.True(t, mockBank.sendCoinsCalled)
	require.False(t, mockBank.burnCoinsCalled, "burn must not execute if transfer failed")
}

func TestMsgServerBurn_EntireSupplyToZero(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	addr := sdk.AccAddress("test_address_tozero")

	// User holds entire supply
	mockBank.balances[addr.String()] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100))
	mockBank.spendable[addr.String()] = sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)))
	mockBank.supplies[app.DefaultBondDenom] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100))

	msg := &types.MsgBurn{
		FromAddress: addr.String(),
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)),
	}

	resp, err := f.msgServer.Burn(f.ctx, msg)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, mockBank.balances[addr.String()].IsZero())
	require.True(t, mockBank.supplies[app.DefaultBondDenom].IsZero(), "supply must reach zero cleanly")
}

// Test post-burn balance verification catches inconsistent state
func TestMsgServerBurn_PostBurnBalanceInconsistent(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	addr := sdk.AccAddress("test_balance_check")

	// Setup initial state
	mockBank.balances[addr.String()] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))
	mockBank.spendable[addr.String()] = sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000)))
	mockBank.supplies[app.DefaultBondDenom] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(10000))

	// Simulate race condition: balance changes incorrectly after burn
	// Should be 900 after burning 100, but returns 950
	mockBank.balanceOnSecondCall = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(950))

	msg := &types.MsgBurn{
		FromAddress: addr.String(),
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)),
	}

	// Should return error when post-burn verification detects inconsistency
	_, err := f.msgServer.Burn(f.ctx, msg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrPostBurnValidation)

	// Verify burn was attempted
	require.True(t, mockBank.sendCoinsCalled)
	require.True(t, mockBank.burnCoinsCalled)
}

// Test post-burn supply verification catches inconsistent state
func TestMsgServerBurn_PostBurnSupplyInconsistent(t *testing.T) {
	mockBank := newMockBankKeeper()
	f := initBurnFixture(t, mockBank)

	addr := sdk.AccAddress("test_supply_check")

	// Setup initial state
	mockBank.balances[addr.String()] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000))
	mockBank.spendable[addr.String()] = sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewInt(1000)))
	mockBank.supplies[app.DefaultBondDenom] = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(10000))

	// Simulate consensus failure: supply changes incorrectly after burn
	// Should be 9900 after burning 100, but returns 9950
	mockBank.supplyOnSecondCall = sdk.NewCoin(app.DefaultBondDenom, math.NewInt(9950))

	msg := &types.MsgBurn{
		FromAddress: addr.String(),
		Amount:      sdk.NewCoin(app.DefaultBondDenom, math.NewInt(100)),
	}

	// Should return error when post-burn verification detects inconsistency
	_, err := f.msgServer.Burn(f.ctx, msg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrPostBurnValidation)

	// Verify burn was attempted
	require.True(t, mockBank.sendCoinsCalled)
	require.True(t, mockBank.burnCoinsCalled)
}
