package types_test

import (
	"os"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/app"
	"github.com/monolythium/mono-chain/x/validator/types"
)

// TestMain sets sdk.DefaultBondDenom = "alyth" via app init().
func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	app.SetBech32Prefixes(cfg)
	app.SetBip44CoinType(cfg)
	os.Exit(m.Run())
}

func TestDefaultGenesisState(t *testing.T) {
	gs := types.DefaultGenesisState()
	require.NotNil(t, gs)
	require.Equal(t, types.DefaultParams(), gs.Params)
}

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		name    string
		genesis types.GenesisState
		wantErr bool
	}{
		{
			name:    "default is valid",
			genesis: *types.DefaultGenesisState(),
		},
		{
			name: "valid with non-zero params",
			genesis: types.GenesisState{
				Params: types.NewParams(
					sdk.NewCoin("alyth", math.NewInt(500)),
					sdk.NewCoin("alyth", math.NewInt(2000)),
				),
			},
		},
		{
			name: "invalid: bad registration burn denom",
			genesis: types.GenesisState{
				Params: types.NewParams(
					sdk.NewCoin("uatom", math.NewInt(100)),
					sdk.NewCoin("alyth", math.NewInt(1000)),
				),
			},
			wantErr: true,
		},
		{
			name: "invalid: bad min self delegation denom",
			genesis: types.GenesisState{
				Params: types.NewParams(
					sdk.NewCoin("alyth", math.NewInt(100)),
					sdk.NewCoin("uatom", math.NewInt(1000)),
				),
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genesis.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
