package burn_test

import (
	"testing"

	"cosmossdk.io/math"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	burntypes "github.com/monolythium/mono-chain/x/burn/types"
)

func TestBeginBlock_ProcessFeeBurn(t *testing.T) {
	f := initFixture(t)

	// Override burn percent (initFixture sets it to zero)
	burnPercent := math.LegacyNewDecWithPrec(30, 2) // 30%
	require.NoError(t, f.burnKeeper.Params.Set(f.sdkCtx, burntypes.NewParams(burnPercent)))

	// Fund fee_collector via mint module (has Minter permission)
	feeAmount := math.NewIntWithDecimal(1, 18)
	feeCoins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, feeAmount))
	require.NoError(t, f.bankKeeper.MintCoins(f.sdkCtx, minttypes.ModuleName, feeCoins))
	require.NoError(t, f.bankKeeper.SendCoinsFromModuleToModule(
		f.sdkCtx, minttypes.ModuleName, authtypes.FeeCollectorName, feeCoins,
	))

	// FinalizeBlock triggers BeginBlock -> mono.ProcessFeeBurn
	height := f.app.LastBlockHeight() + 1
	_, err := f.app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height})
	require.NoError(t, err)

	// fee_collector should retain only the unburned remainder
	expectedBurn := burnPercent.MulInt(feeAmount).TruncateInt()
	expectedRemainder := feeAmount.Sub(expectedBurn)

	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	got := f.bankKeeper.GetAllBalances(f.sdkCtx, feeCollectorAddr).AmountOf(sdk.DefaultBondDenom)
	require.Equal(t, expectedRemainder, got)
}
