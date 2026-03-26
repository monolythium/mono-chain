package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/x/burn/types"
)

func TestParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  types.Params
		wantErr bool
	}{
		{
			name:   "valid: zero percent",
			params: types.NewParams(math.LegacyZeroDec()),
		},
		{
			name:   "valid: 50%",
			params: types.NewParams(math.LegacyNewDecWithPrec(5, 1)),
		},
		{
			name:   "valid: 100%",
			params: types.NewParams(math.LegacyOneDec()),
		},
		{
			name:    "invalid: nil percent",
			params:  types.NewParams(math.LegacyDec{}),
			wantErr: true,
		},
		{
			name:    "invalid: negative percent",
			params:  types.NewParams(math.LegacyNewDec(-1)),
			wantErr: true,
		},
		{
			name:    "invalid: exceeds 1.0",
			params:  types.NewParams(math.LegacyNewDecWithPrec(101, 2)),
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, types.ErrInvalidFeeBurnPercent)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()
	require.NoError(t, params.Validate())
	require.True(t, params.FeeBurnPercent.Equal(math.LegacyZeroDec()))
}
