package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/monolythium/mono-chain/x/mono/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	testCases := []struct {
		desc     string
		genState *types.GenesisState
		wantErr  bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			wantErr:  false,
		},
		{
			desc:     "empty genesis state is invalid (nil params)",
			genState: &types.GenesisState{},
			wantErr:  true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_Validate(t *testing.T) {
	testCases := []struct {
		name      string
		params    types.Params
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid: zero burn percent",
			params: types.NewParams(
				math.LegacyZeroDec(),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
			),
			wantErr: false,
		},
		{
			name: "valid: 100% burn",
			params: types.NewParams(
				math.LegacyOneDec(),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
			),
			wantErr: false,
		},
		{
			name: "invalid: negative burn percent",
			params: types.NewParams(
				math.LegacyNewDec(-1),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
			),
			wantErr:   true,
			errSubstr: "invalid fee burn percent",
		},
		{
			name: "invalid: burn percent exceeds 1.0",
			params: types.NewParams(
				math.LegacyNewDecWithPrec(101, 2),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
			),
			wantErr:   true,
			errSubstr: "invalid fee burn percent",
		},
		{
			name: "invalid: registration fee wrong denom",
			params: types.NewParams(
				math.LegacyZeroDec(),
				sdk.NewCoin("uatom", math.NewInt(1000)),
				sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000)),
			),
			wantErr:   true,
			errSubstr: "invalid validator registration fee",
		},
		{
			name: "valid: non-zero registration fee correct denom",
			params: types.NewParams(
				math.LegacyZeroDec(),
				sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000)),
				sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000)),
			),
			wantErr: false,
		},
		{
			name: "invalid: min self delegation wrong denom",
			params: types.NewParams(
				math.LegacyZeroDec(),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
				sdk.NewCoin("uatom", math.NewInt(1000)),
			),
			wantErr:   true,
			errSubstr: "invalid validator min self delegation",
		},
		{
			name: "invalid: registration burn negative amount",
			params: types.NewParams(
				math.LegacyZeroDec(),
				sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: math.NewInt(-1)},
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
			),
			wantErr:   true,
			errSubstr: "invalid validator registration fee",
		},
		{
			name: "invalid: min self delegation negative amount",
			params: types.NewParams(
				math.LegacyZeroDec(),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
				sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: math.NewInt(-1)},
			),
			wantErr:   true,
			errSubstr: "invalid validator min self delegation",
		},
		{
			name: "invalid: nil fee burn percent",
			params: types.NewParams(
				math.LegacyDec{},
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
			),
			wantErr:   true,
			errSubstr: "invalid fee burn percent",
		},
		{
			name: "valid: non-zero min self delegation correct denom",
			params: types.NewParams(
				math.LegacyZeroDec(),
				sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
				sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000)),
			),
			wantErr: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
