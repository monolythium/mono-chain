package mono

import (
	"math/rand"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	monosimulation "github.com/monolythium/mono-chain/x/mono/simulation"
	"github.com/monolythium/mono-chain/x/mono/types"
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}

	monoGenesis := types.GenesisState{
		Params: types.NewParams(
			math.LegacyNewDecWithPrec(50, 2),
			sdk.NewCoin(sdk.DefaultBondDenom, math.NewIntWithDecimal(100_000, 18)),
			sdk.NewCoin(sdk.DefaultBondDenom, math.NewIntWithDecimal(100_000, 18)),
		),
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&monoGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	const (
		opWeightValRegistration      = "op_weight_validator_registration"
		defaultWeightValRegistration = 10
	)

	var weightValRegistration int
	simState.AppParams.GetOrGenerate(opWeightValRegistration, &weightValRegistration, nil,
		func(_ *rand.Rand) {
			weightValRegistration = defaultWeightValRegistration
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightValRegistration,
		monosimulation.SimulateValidatorRegistration(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{}
}
