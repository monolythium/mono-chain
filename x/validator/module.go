package validator

import (
	"encoding/json"

	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil/simsx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	evmaddress "github.com/cosmos/evm/encoding/address"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"

	"github.com/monolythium/mono-chain/x/validator/keeper"
	"github.com/monolythium/mono-chain/x/validator/simulation"
	"github.com/monolythium/mono-chain/x/validator/types"
)

const ConsensusVersion = 1

var (
	_ module.AppModuleBasic      = AppModule{}
	_ module.HasGenesis          = AppModule{}
	_ module.AppModuleSimulation = AppModule{}

	_ appmodule.AppModule   = AppModule{}
	_ appmodule.HasServices = AppModule{}
)

type AppModule struct {
	cdc           codec.Codec
	keeper        keeper.Keeper
	stakingKeeper simulation.StakingQuerier
}

func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
	sk simulation.StakingQuerier,
) AppModule {
	return AppModule{
		cdc:           cdc,
		keeper:        keeper,
		stakingKeeper: sk,
	}
}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// Name returns the name of the module as a string.
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec is a no-op.
// Amino signing is not supported on EVM-native chains.
func (AppModule) RegisterLegacyAminoCodec(*codec.LegacyAmino) {}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(
		clientCtx.CmdContext,
		mux,
		types.NewQueryClient(clientCtx),
	); err != nil {
		panic(err)
	}
}

// RegisterInterfaces registers a module's interface types and their concrete implementations as proto.Message.
func (AppModule) RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registrar)
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries.
func (am AppModule) RegisterServices(registrar grpc.ServiceRegistrar) error {
	types.RegisterMsgServer(registrar, keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(registrar, keeper.NewQueryServerImpl(am.keeper))

	return nil
}

// DefaultGenesis returns a default GenesisState for the module, marshalled to json.RawMessage.
func (am AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis validates the GenesisState, given in its json.RawMessage form.
func (am AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return err
	}

	return genState.Validate()
}

// InitGenesis performs the module's genesis initialization.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(gs, &genState); err != nil {
		panic(err)
	}

	if err := am.keeper.InitGenesis(ctx, genState); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState, err := am.keeper.ExportGenesis(ctx)
	if err != nil {
		panic(err)
	}

	bz, err := cdc.MarshalJSON(genState)
	if err != nil {
		panic(err)
	}

	return bz
}

// ConsensusVersion is a sequence number for state-breaking change of the module.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

/// Simulation (AppModuleSimulation gate) ///

// GenerateGenesisState creates a randomized GenState of the validator module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// RegisterStoreDecoder registers a decoder for the validator module's types.
func (am AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
	sdr[types.StoreKey] = simtypes.NewStoreDecoderFuncFromCollectionsSchema(am.keeper.Schema)
}

// WeightedOperations satisfies the AppModuleSimulation interface (gate stub).
// Actual operations are registered via WeightedOperationsX.
func (AppModule) WeightedOperations(module.SimulationState) []simtypes.WeightedOperation {
	return nil
}

/// Simulation (simsx behavior) ///

// WeightedOperationsX registers weighted validator module operations for simulation.
func (am AppModule) WeightedOperationsX(weights simsx.WeightSource, reg simsx.Registry) {
	valAddrCodec := evmaddress.NewEvmCodec(
		sdk.GetConfig().GetBech32ValidatorAddrPrefix(),
	)
	reg.Add(
		weights.Get("msg_register_validator", 100),
		simulation.MsgRegisterValidatorFactory(
			am.keeper,
			am.stakingKeeper,
			valAddrCodec,
		),
	)
}

// ProposalMsgsX registers governance proposal messages for simulation.
func (AppModule) ProposalMsgsX(weights simsx.WeightSource, reg simsx.Registry) {
	reg.Add(weights.Get("msg_update_validator_params", 100), simulation.MsgUpdateParamsFactory())
}
