package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
			sdk.NewCoin(sdk.DefaultBondDenom, math.NewIntWithDecimal(100_000, 18)), // 100k LYTH
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
	require.Equal(t, genesisState.Params.ValidatorRegistrationFee, got.Params.ValidatorRegistrationFee)
}
