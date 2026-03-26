package simulation

import (
	"math/rand"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/monolythium/mono-chain/x/validator/types"
)

// RandomizedGenState generates a random GenesisState for the validator module.
func RandomizedGenState(simState *module.SimulationState) {
	var registrationBurnAmount int64
	simState.AppParams.GetOrGenerate(
		"validator_registration_burn",
		&registrationBurnAmount,
		simState.Rand,
		func(r *rand.Rand) {
			registrationBurnAmount = int64(r.Intn(1000))
		},
	)

	var minSelfDelegationAmount int64
	simState.AppParams.GetOrGenerate(
		"validator_min_self_delegation",
		&minSelfDelegationAmount,
		simState.Rand,
		func(r *rand.Rand) {
			minSelfDelegationAmount = int64(r.Intn(1000))
		},
	)

	validatorGenesis := types.GenesisState{
		Params: types.NewParams(
			sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(registrationBurnAmount)),
			sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(minSelfDelegationAmount)),
		),
	}

	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&validatorGenesis)
}
