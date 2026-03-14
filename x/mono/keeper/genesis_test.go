package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/monolythium/mono-chain/app"
	"github.com/monolythium/mono-chain/x/mono/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.EqualExportedValues(t, genesisState.Params, got.Params)
}

func TestGenesis_NonDefaultParams(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.NewParams(
			math.LegacyNewDecWithPrec(90, 2),                                       // 0.90
			sdk.NewCoin(app.DefaultBondDenom, math.NewIntWithDecimal(100_000, 18)), // 100k LYTH
			sdk.NewCoin(app.DefaultBondDenom, math.NewIntWithDecimal(100_000, 18)), // 100k LYTH
		),
	}

	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.True(t, genesisState.Params.FeeBurnPercent.Equal(got.Params.FeeBurnPercent),
		"FeeBurnPercent precision lost: want %s, got %s",
		genesisState.Params.FeeBurnPercent, got.Params.FeeBurnPercent)
	require.Equal(t, genesisState.Params.ValidatorRegistrationBurn, got.Params.ValidatorRegistrationBurn)
	require.Equal(t, genesisState.Params.ValidatorMinSelfDelegation, got.Params.ValidatorMinSelfDelegation)
}

func TestExportGenesis_ParamsNotSet(t *testing.T) {
	f := initFixture(t)

	// Remove params from the store so Get returns an error.
	err := f.keeper.Params.Remove(f.ctx)
	require.NoError(t, err)

	got, err := f.keeper.ExportGenesis(f.ctx)
	require.Error(t, err)
	require.Nil(t, got)
}
