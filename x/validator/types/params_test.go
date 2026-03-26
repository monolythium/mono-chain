package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/x/validator/types"
)

func TestParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  types.Params
		wantErr error
	}{
		{
			name:   "default is valid",
			params: types.DefaultParams(),
		},
		{
			name: "valid: non-zero native denom",
			params: types.NewParams(
				sdk.NewCoin("alyth", math.NewInt(500)),
				sdk.NewCoin("alyth", math.NewInt(2000)),
			),
		},
		{
			name: "invalid: registration burn negative amount",
			params: types.NewParams(
				sdk.Coin{Denom: "alyth", Amount: math.NewInt(-1)},
				sdk.NewCoin("alyth", math.NewInt(1000)),
			),
			wantErr: types.ErrInvalidRegistrationBurn,
		},
		{
			name: "invalid: registration burn wrong denom",
			params: types.NewParams(
				sdk.NewCoin("uatom", math.NewInt(100)),
				sdk.NewCoin("alyth", math.NewInt(1000)),
			),
			wantErr: types.ErrInvalidRegistrationBurn,
		},
		{
			name: "invalid: min self delegation negative amount",
			params: types.NewParams(
				sdk.NewCoin("alyth", math.NewInt(100)),
				sdk.Coin{Denom: "alyth", Amount: math.NewInt(-1)},
			),
			wantErr: types.ErrInvalidMinSelfDelegation,
		},
		{
			name: "invalid: min self delegation wrong denom",
			params: types.NewParams(
				sdk.NewCoin("alyth", math.NewInt(100)),
				sdk.NewCoin("uatom", math.NewInt(1000)),
			),
			wantErr: types.ErrInvalidMinSelfDelegation,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
