package burn_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	burntypes "github.com/monolythium/mono-chain/x/burn/types"
)

func TestMsgBurn_Integration(t *testing.T) {
	f := initFixture(t)

	// Create and fund a user account
	userAddr := sdk.AccAddress([]byte("burn_integration_usr"))
	userAcc := f.accountKeeper.NewAccountWithAddress(f.sdkCtx, userAddr)
	f.accountKeeper.SetAccount(f.sdkCtx, userAcc)

	fundAmount := math.NewInt(10_000)
	fundCoins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, fundAmount))
	require.NoError(t, f.bankKeeper.MintCoins(f.sdkCtx, minttypes.ModuleName, fundCoins))
	require.NoError(t, f.bankKeeper.SendCoinsFromModuleToAccount(f.sdkCtx, minttypes.ModuleName, userAddr, fundCoins))

	// Verify starting state
	bal := f.bankKeeper.GetBalance(f.sdkCtx, userAddr, sdk.DefaultBondDenom)
	require.Equal(t, fundAmount, bal.Amount)

	// Execute MsgBurn through the app router
	burnAmount := math.NewInt(3_000)
	msg := &burntypes.MsgBurn{
		FromAddress: userAddr.String(),
		Amount:      sdk.NewCoin(sdk.DefaultBondDenom, burnAmount),
	}
	_, err := f.app.RunMsg(msg)
	require.NoError(t, err)

	// Verify balance decreased
	bal = f.bankKeeper.GetBalance(f.sdkCtx, userAddr, sdk.DefaultBondDenom)
	require.Equal(t, fundAmount.Sub(burnAmount), bal.Amount)

	// Verify burn tracking
	globalCount, err := f.burnKeeper.GlobalBurnCount.Peek(f.sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), globalCount)

	globalBurnTotal, err := f.burnKeeper.GetGlobalBurnTotal(f.sdkCtx)
	require.Equal(t, burnAmount, globalBurnTotal)
	require.NoError(t, err)

	accountBurnTotal, err := f.burnKeeper.GetAccountBurnTotal(f.sdkCtx, userAddr)
	require.Equal(t, burnAmount, accountBurnTotal)
	require.NoError(t, err)
}

func TestMsgBurn_Integration_WrongDenom(t *testing.T) {
	f := initFixture(t)

	userAddr := sdk.AccAddress([]byte("burn_wrong_denom_usr"))
	userAcc := f.accountKeeper.NewAccountWithAddress(f.sdkCtx, userAddr)
	f.accountKeeper.SetAccount(f.sdkCtx, userAcc)

	msg := &burntypes.MsgBurn{
		FromAddress: userAddr.String(),
		Amount:      sdk.NewCoin("uatom", math.NewInt(100)),
	}
	_, err := f.app.RunMsg(msg)
	require.Error(t, err)
}

func TestMsgBurn_Integration_InsufficientFunds(t *testing.T) {
	f := initFixture(t)

	// Create account with small balance
	userAddr := sdk.AccAddress([]byte("burn_insuf_fund_usr_"))
	userAcc := f.accountKeeper.NewAccountWithAddress(f.sdkCtx, userAddr)
	f.accountKeeper.SetAccount(f.sdkCtx, userAcc)

	fundCoins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100)))
	require.NoError(t, f.bankKeeper.MintCoins(f.sdkCtx, minttypes.ModuleName, fundCoins))
	require.NoError(t, f.bankKeeper.SendCoinsFromModuleToAccount(f.sdkCtx, minttypes.ModuleName, userAddr, fundCoins))

	// Try to burn more than balance
	msg := &burntypes.MsgBurn{
		FromAddress: userAddr.String(),
		Amount:      sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(500)),
	}
	_, err := f.app.RunMsg(msg)
	require.Error(t, err)

	// Balance unchanged
	bal := f.bankKeeper.GetBalance(f.sdkCtx, userAddr, sdk.DefaultBondDenom)
	require.Equal(t, math.NewInt(100), bal.Amount)
}

func TestMsgBurn_Integration_EntireBalance(t *testing.T) {
	f := initFixture(t)

	userAddr := sdk.AccAddress([]byte("burn_entire_bal_usr_"))
	userAcc := f.accountKeeper.NewAccountWithAddress(f.sdkCtx, userAddr)
	f.accountKeeper.SetAccount(f.sdkCtx, userAcc)

	fundAmount := math.NewInt(500)
	fundCoins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, fundAmount))
	require.NoError(t, f.bankKeeper.MintCoins(f.sdkCtx, minttypes.ModuleName, fundCoins))
	require.NoError(t, f.bankKeeper.SendCoinsFromModuleToAccount(f.sdkCtx, minttypes.ModuleName, userAddr, fundCoins))

	// Burn entire balance
	msg := &burntypes.MsgBurn{
		FromAddress: userAddr.String(),
		Amount:      sdk.NewCoin(sdk.DefaultBondDenom, fundAmount),
	}
	_, err := f.app.RunMsg(msg)
	require.NoError(t, err)

	bal := f.bankKeeper.GetBalance(f.sdkCtx, userAddr, sdk.DefaultBondDenom)
	require.True(t, bal.IsZero())
}

func TestMsgBurn_Integration_MultipleBurns(t *testing.T) {
	f := initFixture(t)

	userAddr := sdk.AccAddress([]byte("burn_multi_burn_usr_"))
	userAcc := f.accountKeeper.NewAccountWithAddress(f.sdkCtx, userAddr)
	f.accountKeeper.SetAccount(f.sdkCtx, userAcc)

	fundCoins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000)))
	require.NoError(t, f.bankKeeper.MintCoins(f.sdkCtx, minttypes.ModuleName, fundCoins))
	require.NoError(t, f.bankKeeper.SendCoinsFromModuleToAccount(f.sdkCtx, minttypes.ModuleName, userAddr, fundCoins))

	// Burn 1: 300
	_, err := f.app.RunMsg(&burntypes.MsgBurn{
		FromAddress: userAddr.String(),
		Amount:      sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(300)),
	})
	require.NoError(t, err)

	// Burn 2: 200
	_, err = f.app.RunMsg(&burntypes.MsgBurn{
		FromAddress: userAddr.String(),
		Amount:      sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(200)),
	})
	require.NoError(t, err)

	// Balance: 1000 - 300 - 200 = 500
	bal := f.bankKeeper.GetBalance(f.sdkCtx, userAddr, sdk.DefaultBondDenom)
	require.Equal(t, math.NewInt(500), bal.Amount)

	// Tracking: count=2, total=500, accountTotal=500
	globalCount, err := f.burnKeeper.GlobalBurnCount.Peek(f.sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), globalCount)

	globalBurnTotal, err := f.burnKeeper.GetGlobalBurnTotal(f.sdkCtx)
	require.Equal(t, math.NewInt(500), globalBurnTotal)
	require.NoError(t, err)

	accountBurnTotal, err := f.burnKeeper.GetAccountBurnTotal(f.sdkCtx, userAddr)
	require.Equal(t, math.NewInt(500), accountBurnTotal)
	require.NoError(t, err)
}

func TestMsgUpdateParams_Integration(t *testing.T) {
	f := initFixture(t)

	authority := authtypes.NewModuleAddress(burntypes.GovModuleName)
	newParams := burntypes.NewParams(math.LegacyNewDecWithPrec(25, 2)) // 25%

	msg := &burntypes.MsgUpdateParams{
		Authority: authority.String(),
		Params:    newParams,
	}
	_, err := f.app.RunMsg(msg)
	require.NoError(t, err)

	got, err := f.burnKeeper.Params.Get(f.sdkCtx)
	require.NoError(t, err)
	require.Equal(t, newParams, got)
}
