package types_test

import (
	"testing"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/x/burn/types"
)

func addrCodec() address.Codec {
	return addresscodec.NewBech32Codec("mono")
}

func validAddr(t *testing.T) string {
	t.Helper()
	s, err := addrCodec().BytesToString([]byte("test_address________"))
	require.NoError(t, err)
	return s
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
			name: "valid with tracking data",
			genesis: types.GenesisState{
				Params: types.DefaultParams(),
				Count:  5,
				Total:  math.NewInt(500),
				AccountTotals: []types.AccountBurnRecord{
					{Address: validAddr(t), Total: math.NewInt(500)},
				},
			},
		},
		{
			name: "valid: zero total",
			genesis: types.GenesisState{
				Params: types.DefaultParams(),
				Total:  math.ZeroInt(),
			},
		},
		{
			name: "invalid: bad params",
			genesis: types.GenesisState{
				Params: types.NewParams(math.LegacyNewDec(-1)),
			},
			wantErr: true,
		},
		{
			name: "invalid: negative total",
			genesis: types.GenesisState{
				Params: types.DefaultParams(),
				Total:  math.NewInt(-1),
			},
			wantErr: true,
		},
		{
			name: "invalid: bad address in record",
			genesis: types.GenesisState{
				Params: types.DefaultParams(),
				AccountTotals: []types.AccountBurnRecord{
					{Address: "not_bech32", Total: math.NewInt(100)},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid: nil total in record",
			genesis: types.GenesisState{
				Params: types.DefaultParams(),
				AccountTotals: []types.AccountBurnRecord{
					{Address: validAddr(t), Total: math.Int{}},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid: negative total in record",
			genesis: types.GenesisState{
				Params: types.DefaultParams(),
				AccountTotals: []types.AccountBurnRecord{
					{Address: validAddr(t), Total: math.NewInt(-1)},
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genesis.Validate(addrCodec())
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
