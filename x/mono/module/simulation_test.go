package mono_test

import (
	"encoding/json"
	"os"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	evmaddress "github.com/cosmos/evm/encoding/address"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/app"
	"github.com/monolythium/mono-chain/x/mono/keeper"
	mono "github.com/monolythium/mono-chain/x/mono/module"
	"github.com/monolythium/mono-chain/x/mono/types"
)

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	app.SetBech32Prefixes(cfg)
	app.SetBip44CoinType(cfg)
	os.Exit(m.Run())
}

type simFixture struct {
	ctx sdk.Context
	am  mono.AppModule
	cdc moduletestutil.TestEncodingConfig
}

func initSimFixture(t *testing.T) *simFixture {
	t.Helper()
	encCfg := moduletestutil.MakeTestEncodingConfig(mono.AppModule{})
	addressCodec := evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx
	authority := authtypes.NewModuleAddress(types.GovModuleName)

	k := keeper.NewKeeper(storeService, encCfg.Codec, addressCodec, authority, nil, nil, nil, nil, nil, nil)
	am := mono.NewAppModule(encCfg.Codec, k, nil, nil)

	return &simFixture{ctx: ctx, am: am, cdc: encCfg}
}

func TestGenerateGenesisState_RoundTrip(t *testing.T) {
	f := initSimFixture(t)

	simState := module.SimulationState{
		Cdc:      f.cdc.Codec,
		GenState: make(map[string]json.RawMessage),
	}
	f.am.GenerateGenesisState(&simState)

	raw, ok := simState.GenState[types.ModuleName]
	require.True(t, ok, "GenerateGenesisState must populate module key")

	err := f.am.ValidateGenesis(f.cdc.Codec, nil, raw)
	require.NoError(t, err, "sim-generated genesis must pass validation")

	f.am.InitGenesis(f.ctx, f.cdc.Codec, raw)
	exported := f.am.ExportGenesis(f.ctx, f.cdc.Codec)

	var original, roundTripped types.GenesisState
	require.NoError(t, f.cdc.Codec.UnmarshalJSON(raw, &original))
	require.NoError(t, f.cdc.Codec.UnmarshalJSON(exported, &roundTripped))

	require.True(t, original.Params.FeeBurnPercent.Equal(roundTripped.Params.FeeBurnPercent),
		"FeeBurnPercent lost precision: want %s, got %s",
		original.Params.FeeBurnPercent, roundTripped.Params.FeeBurnPercent)
	require.Equal(t, original.Params.ValidatorRegistrationBurn, roundTripped.Params.ValidatorRegistrationBurn)
	require.Equal(t, original.Params.ValidatorMinSelfDelegation, roundTripped.Params.ValidatorMinSelfDelegation)
}

func TestWeightedOperations(t *testing.T) {
	f := initSimFixture(t)

	simState := module.SimulationState{
		AppParams: make(simtypes.AppParams),
	}
	ops := f.am.WeightedOperations(simState)

	require.Len(t, ops, 1, "must register exactly one weighted operation")
	require.Equal(t, 10, ops[0].Weight(), "default weight must be 10")
}
