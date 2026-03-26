package simulation

import (
	"math/rand"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/monolythium/mono-chain/x/burn/types"
)

// RandomizedGenState generates a random GenesisState for the burn module.
func RandomizedGenState(simState *module.SimulationState) {
	var feeBurnPercent math.LegacyDec
	simState.AppParams.GetOrGenerate(
		"burn_fee_burn_percent",
		&feeBurnPercent,
		simState.Rand,
		func(r *rand.Rand) {
			feeBurnPercent = math.LegacyNewDecWithPrec(int64(r.Intn(100)), 2)
		},
	)

	burnGenesis := types.GenesisState{
		Params: types.NewParams(feeBurnPercent),
	}

	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&burnGenesis)
}
