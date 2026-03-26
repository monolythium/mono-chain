package validator_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"cosmossdk.io/collections"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/monolythium/mono-chain/app"
	validator "github.com/monolythium/mono-chain/x/validator"
	"github.com/monolythium/mono-chain/x/validator/keeper"
	"github.com/monolythium/mono-chain/x/validator/types"
)

type mockServiceRegistrar struct{}

func (mockServiceRegistrar) RegisterService(*grpc.ServiceDesc, any) {}

type failSetStore struct{}

func (failSetStore) Get([]byte) ([]byte, error)                                 { return nil, nil }
func (failSetStore) Has([]byte) (bool, error)                                   { return false, nil }
func (failSetStore) Set([]byte, []byte) error                                   { return errors.New("store write failed") }
func (failSetStore) Delete([]byte) error                                        { return nil }
func (failSetStore) Iterator([]byte, []byte) (corestore.Iterator, error)        { return nil, nil }
func (failSetStore) ReverseIterator([]byte, []byte) (corestore.Iterator, error) { return nil, nil }

type failSetStoreService struct{}

func (failSetStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return failSetStore{}
}

type failMarshalCodec struct{ codec.Codec }

func (failMarshalCodec) MarshalJSON(proto.Message) ([]byte, error) {
	return nil, errors.New("marshal failed")
}

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	app.SetBech32Prefixes(cfg)
	app.SetBip44CoinType(cfg)
	os.Exit(m.Run())
}

type moduleFixture struct {
	am     validator.AppModule
	ctx    sdk.Context
	keeper keeper.Keeper
	cdc    codec.Codec
}

func setupModule(t *testing.T) moduleFixture {
	t.Helper()
	encCfg := moduletestutil.MakeTestEncodingConfig()
	addrCodec := addresscodec.NewBech32Codec("mono")
	valAddrCodec := addresscodec.NewBech32Codec("monovaloper")
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority, err := addrCodec.BytesToString(authtypes.NewModuleAddress(types.GovModuleName))
	require.NoError(t, err)

	k := keeper.NewKeeper(storeService, encCfg.Codec, addrCodec, valAddrCodec, authority, nil, nil)
	err = k.Params.Set(ctx, types.DefaultParams())
	require.NoError(t, err)

	return moduleFixture{
		am:     validator.NewAppModule(encCfg.Codec, k, nil),
		ctx:    ctx,
		keeper: k,
		cdc:    encCfg.Codec,
	}
}

func TestAppModule_Name(t *testing.T) {
	f := setupModule(t)
	require.Equal(t, "validator", f.am.Name())
}

func TestAppModule_ConsensusVersion(t *testing.T) {
	f := setupModule(t)
	require.Equal(t, uint64(1), f.am.ConsensusVersion())
}

func TestAppModule_IsAppModule(t *testing.T) {
	f := setupModule(t)
	f.am.IsAppModule()
	f.am.IsOnePerModuleType()
}

func TestAppModule_RegisterLegacyAminoCodec(t *testing.T) {
	f := setupModule(t)
	f.am.RegisterLegacyAminoCodec(nil)
}

func TestAppModule_RegisterServices(t *testing.T) {
	f := setupModule(t)
	err := f.am.RegisterServices(mockServiceRegistrar{})
	require.NoError(t, err)
}

func TestAppModule_DefaultGenesis(t *testing.T) {
	f := setupModule(t)

	bz := f.am.DefaultGenesis(f.cdc)
	require.NotNil(t, bz)

	var genState types.GenesisState
	err := f.cdc.UnmarshalJSON(bz, &genState)
	require.NoError(t, err)
	require.Equal(t, types.DefaultParams(), genState.Params)
}

func TestAppModule_ValidateGenesis_Valid(t *testing.T) {
	f := setupModule(t)

	bz := f.am.DefaultGenesis(f.cdc)
	err := f.am.ValidateGenesis(f.cdc, nil, bz)
	require.NoError(t, err)
}

func TestAppModule_ValidateGenesis_InvalidJSON(t *testing.T) {
	f := setupModule(t)
	err := f.am.ValidateGenesis(f.cdc, nil, []byte("not json"))
	require.Error(t, err)
}

func TestAppModule_ValidateGenesis_InvalidParams(t *testing.T) {
	f := setupModule(t)

	genState := types.GenesisState{
		Params: types.NewParams(
			sdk.Coin{Denom: "alyth", Amount: math.NewInt(-1)},
			sdk.NewCoin("alyth", math.NewInt(1000)),
		),
	}
	bz, err := f.cdc.MarshalJSON(&genState)
	require.NoError(t, err)

	err = f.am.ValidateGenesis(f.cdc, nil, bz)
	require.Error(t, err)
}

func TestAppModule_InitExportGenesis_RoundTrip(t *testing.T) {
	f := setupModule(t)

	bz := f.am.DefaultGenesis(f.cdc)
	require.NotPanics(t, func() {
		f.am.InitGenesis(f.ctx, f.cdc, bz)
	})

	exported := f.am.ExportGenesis(f.ctx, f.cdc)
	require.NotNil(t, exported)

	err := f.am.ValidateGenesis(f.cdc, nil, exported)
	require.NoError(t, err)
}

func TestAppModule_InitGenesis_BadJSON(t *testing.T) {
	f := setupModule(t)
	require.Panics(t, func() {
		f.am.InitGenesis(f.ctx, f.cdc, []byte("bad json"))
	})
}

func TestAppModule_ExportGenesis_KeeperError(t *testing.T) {
	f := setupModule(t)

	err := f.keeper.Params.Remove(f.ctx)
	require.NoError(t, err)

	require.Panics(t, func() {
		f.am.ExportGenesis(f.ctx, f.cdc)
	})
}

func TestAppModule_ExportGenesis_MarshalFails(t *testing.T) {
	f := setupModule(t)

	require.Panics(t, func() {
		f.am.ExportGenesis(f.ctx, failMarshalCodec{Codec: f.cdc})
	})
}

func TestAppModule_InitGenesis_KeeperError(t *testing.T) {
	f := setupModule(t)

	sb := collections.NewSchemaBuilder(failSetStoreService{})
	f.keeper.Params = collections.NewItem(sb, types.ParamsKey, "params",
		codec.CollValue[types.Params](f.cdc))
	_, _ = sb.Build()

	am := validator.NewAppModule(f.cdc, f.keeper, nil)
	bz := f.cdc.MustMarshalJSON(types.DefaultGenesisState())

	require.Panics(t, func() {
		am.InitGenesis(f.ctx, f.cdc, bz)
	})
}

func TestAppModule_RegisterGRPCGatewayRoutes(t *testing.T) {
	f := setupModule(t)
	mux := gwruntime.NewServeMux()
	clientCtx := client.Context{CmdContext: context.Background()}

	require.NotPanics(t, func() {
		f.am.RegisterGRPCGatewayRoutes(clientCtx, mux)
	})
}

func TestAppModule_RegisterInterfaces(t *testing.T) {
	f := setupModule(t)
	encCfg := moduletestutil.MakeTestEncodingConfig()

	f.am.RegisterInterfaces(encCfg.InterfaceRegistry)

	resolved, err := encCfg.InterfaceRegistry.Resolve(sdk.MsgTypeURL(&types.MsgRegisterValidator{}))
	require.NoError(t, err)
	require.NotNil(t, resolved)
}
