package keeper_test

import (
	"bytes"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/app"
	"github.com/monolythium/mono-chain/x/mono/keeper"
	"github.com/monolythium/mono-chain/x/mono/types"
)

func TestMsgUpdateParams(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	// default params
	testCases := []struct {
		name      string
		input     *types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "wrong authority",
			input: func() *types.MsgUpdateParams {
				wrongAddr, _ := f.addressCodec.BytesToString(bytes.Repeat([]byte{0xFF}, 20))
				return &types.MsgUpdateParams{
					Authority: wrongAddr,
					Params:    params,
				}
			}(),
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "nil params rejected",
			input: &types.MsgUpdateParams{
				Authority: authorityStr,
				Params:    types.Params{},
			},
			expErr:    true,
			expErrMsg: "invalid fee burn percent",
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{
				Authority: authorityStr,
				Params:    params,
			},
			expErr: false,
		},
		{
			name: "reject registration burn wrong denom",
			input: &types.MsgUpdateParams{
				Authority: authorityStr,
				Params:    types.NewParams(math.LegacyZeroDec(), sdk.NewCoin("uatom", math.NewInt(1000)), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt())),
			},
			expErr:    true,
			expErrMsg: "invalid validator registration fee",
		},
		{
			name: "reject min self delegation wrong denom",
			input: &types.MsgUpdateParams{
				Authority: authorityStr,
				Params:    types.NewParams(math.LegacyZeroDec(), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()), sdk.NewCoin("uatom", math.NewInt(1000))),
			},
			expErr:    true,
			expErrMsg: "invalid validator min self delegation",
		},
		{
			name: "reject negative registration burn",
			input: &types.MsgUpdateParams{
				Authority: authorityStr,
				Params:    types.NewParams(math.LegacyZeroDec(), sdk.Coin{Denom: app.DefaultBondDenom, Amount: math.NewInt(-1)}, sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt())),
			},
			expErr:    true,
			expErrMsg: "invalid validator registration fee",
		},
		{
			name: "reject negative min self delegation",
			input: &types.MsgUpdateParams{
				Authority: authorityStr,
				Params:    types.NewParams(math.LegacyZeroDec(), sdk.NewCoin(app.DefaultBondDenom, math.ZeroInt()), sdk.Coin{Denom: app.DefaultBondDenom, Amount: math.NewInt(-1)}),
			},
			expErr:    true,
			expErrMsg: "invalid validator min self delegation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateParams(f.ctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgUpdateParams_NonDefaultValues(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	authorityStr, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	require.NoError(t, err)

	newParams := types.NewParams(
		math.LegacyNewDecWithPrec(90, 2),
		sdk.NewCoin(app.DefaultBondDenom, math.NewIntWithDecimal(100_000, 18)),
		sdk.NewCoin(app.DefaultBondDenom, math.NewIntWithDecimal(50_000, 18)),
	)

	_, err = ms.UpdateParams(f.ctx, &types.MsgUpdateParams{
		Authority: authorityStr,
		Params:    newParams,
	})
	require.NoError(t, err)

	got, err := f.keeper.Params.Get(f.ctx)
	require.NoError(t, err)
	require.True(t, newParams.FeeBurnPercent.Equal(got.FeeBurnPercent))
	require.Equal(t, newParams.ValidatorRegistrationBurn, got.ValidatorRegistrationBurn)
	require.Equal(t, newParams.ValidatorMinSelfDelegation, got.ValidatorMinSelfDelegation)
}
