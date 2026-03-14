package mono_test

import (
	"os"
	"testing"

	cmtprototypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"gotest.tools/v3/assert"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/integration"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	evmaddress "github.com/cosmos/evm/encoding/address"

	"github.com/monolythium/mono-chain/app"
	burnkeeper "github.com/monolythium/mono-chain/x/burn/keeper"
	burn "github.com/monolythium/mono-chain/x/burn/module"
	burntypes "github.com/monolythium/mono-chain/x/burn/types"
	monoante "github.com/monolythium/mono-chain/x/mono/ante"
	monokeeper "github.com/monolythium/mono-chain/x/mono/keeper"
	mono "github.com/monolythium/mono-chain/x/mono/module"
	monotypes "github.com/monolythium/mono-chain/x/mono/types"
)

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	app.SetBech32Prefixes(cfg)
	app.SetBip44CoinType(cfg)
	os.Exit(m.Run())
}

type fixture struct {
	app           *integration.App
	sdkCtx        sdk.Context
	cdc           codec.Codec
	keys          map[string]*storetypes.KVStoreKey
	accountKeeper authkeeper.AccountKeeper
	bankKeeper    bankkeeper.Keeper
	stakingKeeper *stakingkeeper.Keeper
	burnKeeper    burnkeeper.Keeper
	monoKeeper    monokeeper.Keeper
}

func initFixture(tb testing.TB) *fixture {
	tb.Helper()

	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		burntypes.StoreKey, monotypes.StoreKey,
	)

	cdc := moduletestutil.MakeTestEncodingConfig(
		auth.AppModuleBasic{},
		staking.AppModuleBasic{},
		mono.AppModule{},
		burn.AppModule{},
	).Codec

	logger := log.NewTestLogger(tb)
	cms := integration.CreateMultiStore(keys, logger)
	newCtx := sdk.NewContext(cms, cmtprototypes.Header{}, true, logger)

	authority := authtypes.NewModuleAddress(monotypes.GovModuleName)
	addrCodec := evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	valAddrCodec := evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	consAddrCodec := evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ConsensusAddrPrefix())

	maccPerms := map[string][]string{
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.ModuleName:        {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		burntypes.ModuleName:           {authtypes.Burner},
		monotypes.ModuleName:           {authtypes.Burner},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		addrCodec,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authority.String(),
	)

	blockedAddresses := map[string]bool{
		accountKeeper.GetAuthority(): false,
	}
	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		accountKeeper,
		blockedAddresses,
		authority.String(),
		log.NewNopLogger(),
	)

	stakingKeeper := stakingkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		authority.String(),
		valAddrCodec,
		consAddrCodec,
	)

	burnKeeper := burnkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[burntypes.StoreKey]),
		cdc,
		addrCodec,
		authority,
		bankKeeper,
	)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(stakingKeeper)
	burnMsgServer := burnkeeper.NewMsgServerImpl(burnKeeper)

	monoKeeper := monokeeper.NewKeeper(
		runtime.NewKVStoreService(keys[monotypes.StoreKey]),
		cdc,
		addrCodec,
		authority,
		nil, nil, nil, // bankKeeper, stakingKeeper, distrKeeper -- not needed for msg handler
		stakingMsgServer,
		burnMsgServer,
		valAddrCodec,
	)

	// Build modules for InitGenesis
	authModule := auth.NewAppModule(cdc, accountKeeper, authsims.RandomGenesisAccounts, nil)
	bankModule := bank.NewAppModule(cdc, bankKeeper, accountKeeper, nil)
	stakingModule := staking.NewAppModule(cdc, stakingKeeper, accountKeeper, bankKeeper, nil)
	burnModule := burn.NewAppModule(cdc, burnKeeper, nil, nil)
	monoModule := mono.NewAppModule(cdc, monoKeeper, nil, nil)

	integrationApp := integration.NewIntegrationApp(newCtx, logger, keys, cdc, map[string]appmodule.AppModule{
		authtypes.ModuleName:    authModule,
		banktypes.ModuleName:    bankModule,
		stakingtypes.ModuleName: stakingModule,
		burntypes.ModuleName:    burnModule,
		monotypes.ModuleName:    monoModule,
	})

	sdkCtx := sdk.UnwrapSDKContext(integrationApp.Context())

	// Register MsgServers on the msg service router
	router := integrationApp.MsgServiceRouter()
	stakingtypes.RegisterMsgServer(router, stakingMsgServer)
	burntypes.RegisterMsgServer(router, burnMsgServer)
	monotypes.RegisterMsgServer(router, monokeeper.NewMsgServerImpl(monoKeeper))

	// Set circuit breaker -- the code under test
	router.SetCircuit(monoante.NewStakingCircuitBreaker())

	// Default params
	if err := stakingKeeper.SetParams(sdkCtx, stakingtypes.DefaultParams()); err != nil {
		assert.NilError(tb, err)
	}
	if err := monoKeeper.Params.Set(sdkCtx, monotypes.NewParams(
		math.LegacyZeroDec(),
		sdk.NewCoin(sdk.DefaultBondDenom, math.NewIntWithDecimal(100_000, 18)),
		sdk.NewCoin(sdk.DefaultBondDenom, math.NewIntWithDecimal(100_000, 18)),
	)); err != nil {
		assert.NilError(tb, err)
	}
	if err := burnKeeper.Params.Set(sdkCtx, burntypes.DefaultParams()); err != nil {
		assert.NilError(tb, err)
	}

	return &fixture{
		app:           integrationApp,
		sdkCtx:        sdkCtx,
		cdc:           cdc,
		keys:          keys,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
		stakingKeeper: stakingKeeper,
		burnKeeper:    burnKeeper,
		monoKeeper:    monoKeeper,
	}
}
