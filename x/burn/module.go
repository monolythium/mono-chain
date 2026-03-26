package burn

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil/simsx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"

	"github.com/monolythium/mono-chain/x/burn/keeper"
	"github.com/monolythium/mono-chain/x/burn/simulation"
	"github.com/monolythium/mono-chain/x/burn/types"
)

const ConsensusVersion = 1

var (
	_ module.AppModuleBasic      = AppModule{}
	_ module.HasGenesis          = AppModule{}
	_ module.AppModuleSimulation = AppModule{}

	_ appmodule.AppModule       = AppModule{}
	_ appmodule.HasServices     = AppModule{}
	_ appmodule.HasBeginBlocker = AppModule{}
)

// AppModule implements the interface that defines the inter-dependent methods needed by modules.
type AppModule struct {
	cdc          codec.Codec
	keeper       keeper.Keeper
	addressCodec address.Codec
}

func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
	addressCodec address.Codec,
) AppModule {
	return AppModule{
		cdc:          cdc,
		keeper:       keeper,
		addressCodec: addressCodec,
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
	if err := types.RegisterQueryHandlerClient(clientCtx.CmdContext, mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterInterfaces registers a module's interface types and their concrete implementations as proto.Message.
func (AppModule) RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registrar)
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(registrar grpc.ServiceRegistrar) error {
	types.RegisterMsgServer(registrar, keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(registrar, keeper.NewQueryServerImpl(am.keeper))

	return nil
}

// DefaultGenesis returns a default GenesisState for the module, marshalled to json.RawMessage.
// The default GenesisState need to be defined by the module developer and is primarily used for testing.
func (am AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis used to validate the GenesisState, given in its json.RawMessage form.
func (am AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return err
	}

	return genState.Validate(am.addressCodec)
}

// InitGenesis performs the module's genesis initialization. It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) {
	var genState types.GenesisState
	// Initialize global index to index in genesis state
	if err := cdc.UnmarshalJSON(gs, &genState); err != nil {
		panic(err)
	}

	if err := am.keeper.InitGenesis(ctx, genState); err != nil {
		panic(fmt.Errorf("failed to initialize %s genesis state: %w", types.ModuleName, err))
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
// It should be incremented on each consensus-breaking change introduced by the module.
// To avoid wrong/empty versions, the initial version should be set to 1.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

/// Simulation (AppModuleSimulation gate) ///

// GenerateGenesisState creates a randomized GenState of the burn module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// RegisterStoreDecoder registers a decoder for the burn module's types.
func (am AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
	sdr[types.StoreKey] = simtypes.NewStoreDecoderFuncFromCollectionsSchema(am.keeper.Schema)
}

// WeightedOperations satisfies the AppModuleSimulation interface (gate stub).
// Actual operations are registered via WeightedOperationsX.
func (AppModule) WeightedOperations(module.SimulationState) []simtypes.WeightedOperation {
	return nil
}

/// Simulation (simsx behavior) ///

// WeightedOperationsX registers weighted burn module operations for simulation.
func (AppModule) WeightedOperationsX(weights simsx.WeightSource, reg simsx.Registry) {
	reg.Add(weights.Get("msg_burn", 100), simulation.MsgBurnFactory())
}

// ProposalMsgsX registers governance proposal messages for simulation.
func (AppModule) ProposalMsgsX(weights simsx.WeightSource, reg simsx.Registry) {
	reg.Add(weights.Get("msg_update_burn_params", 100), simulation.MsgUpdateParamsFactory())
}
