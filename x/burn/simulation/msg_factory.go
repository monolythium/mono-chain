package simulation

import (
	"context"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/testutil/simsx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/monolythium/mono-chain/x/burn/types"
)

// MsgBurnFactory creates a simulation factory for MsgBurn.
func MsgBurnFactory() simsx.SimMsgFactoryFn[*types.MsgBurn] {
	return func(
		_ context.Context,
		testData *simsx.ChainDataSource,
		reporter simsx.SimulationReporter,
	) ([]simsx.SimAccount, *types.MsgBurn) {
		from := testData.AnyAccount(reporter, simsx.WithDenomBalance(sdk.DefaultBondDenom))
		if reporter.IsSkipped() {
			return nil, nil
		}

		coin := from.LiquidBalance().RandSubsetCoin(reporter, sdk.DefaultBondDenom)
		if reporter.IsSkipped() {
			return nil, nil
		}

		msg := types.NewMsgBurn(from.AddressBech32, coin)
		return []simsx.SimAccount{from}, msg
	}
}

// MsgUpdateParamsFactory creates a simulation factory for burn MsgUpdateParams governance proposals.
func MsgUpdateParamsFactory() simsx.SimMsgFactoryFn[*types.MsgUpdateParams] {
	return func(
		_ context.Context,
		testData *simsx.ChainDataSource,
		reporter simsx.SimulationReporter,
	) ([]simsx.SimAccount, *types.MsgUpdateParams) {
		r := testData.Rand()

		feeBurnPercent := r.DecN(math.LegacyOneDec())

		return nil, &types.MsgUpdateParams{
			Authority: testData.ModuleAccountAddress(reporter, types.GovModuleName),
			Params:    types.NewParams(feeBurnPercent),
		}
	}
}
