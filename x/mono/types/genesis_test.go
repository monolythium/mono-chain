package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/monolythium/mono-chain/x/mono/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc:     "empty genesis state is invalid (nil params)",
			genState: &types.GenesisState{},
			valid:    false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestParams_Validate(t *testing.T) {
	tests := []struct {
		name      string
		params    types.Params
		valid     bool
		errSubstr string
	}{
		{
			name:   "valid: zero burn percent",
			params: types.NewParams(math.LegacyZeroDec(), sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())),
			valid:  true,
		},
		{
			name:   "valid: 100% burn",
			params: types.NewParams(math.LegacyOneDec(), sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())),
			valid:  true,
		},
		{
			name:      "invalid: negative burn percent",
			params:    types.NewParams(math.LegacyNewDec(-1), sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())),
			valid:     false,
			errSubstr: "must not be negative",
		},
		{
			name:      "invalid: burn percent exceeds 1.0",
			params:    types.NewParams(math.LegacyNewDecWithPrec(101, 2), sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())),
			valid:     false,
			errSubstr: "must not exceed 1.0",
		},
		{
			name:      "invalid: registration fee wrong denom",
			params:    types.NewParams(math.LegacyZeroDec(), sdk.NewCoin("uatom", math.NewInt(1000))),
			valid:     false,
			errSubstr: "denom must be",
		},
		{
			name:   "valid: non-zero registration fee correct denom",
			params: types.NewParams(math.LegacyZeroDec(), sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000))),
			valid:  true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errSubstr)
			}
		})
	}
}
