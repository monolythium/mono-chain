package validator_test

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
	burn "github.com/monolythium/mono-chain/x/burn"
	burnkeeper "github.com/monolythium/mono-chain/x/burn/keeper"
	burntypes "github.com/monolythium/mono-chain/x/burn/types"
	validator "github.com/monolythium/mono-chain/x/validator"
	validatorante "github.com/monolythium/mono-chain/x/validator/ante"
	validatorkeeper "github.com/monolythium/mono-chain/x/validator/keeper"
	validatortypes "github.com/monolythium/mono-chain/x/validator/types"
)

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	app.SetBech32Prefixes(cfg)
	app.SetBip44CoinType(cfg)
	os.Exit(m.Run())
}

type fixture struct {
	app             *integration.App
	sdkCtx          sdk.Context
	cdc             codec.Codec
	keys            map[string]*storetypes.KVStoreKey
	accountKeeper   authkeeper.AccountKeeper
	bankKeeper      bankkeeper.Keeper
	stakingKeeper   *stakingkeeper.Keeper
	burnKeeper      burnkeeper.Keeper
	validatorKeeper validatorkeeper.Keeper
}

func initFixture(tb testing.TB) *fixture {
	tb.Helper()

	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		burntypes.StoreKey, validatortypes.StoreKey,
	)

	cdc := moduletestutil.MakeTestEncodingConfig(
		auth.AppModuleBasic{},
		staking.AppModuleBasic{},
		validator.AppModule{},
		burn.AppModule{},
	).Codec

	logger := log.NewTestLogger(tb)
	cms := integration.CreateMultiStore(keys, logger)
	newCtx := sdk.NewContext(cms, cmtprototypes.Header{}, true, logger)

	authority := authtypes.NewModuleAddress(validatortypes.GovModuleName)
	addrCodec := evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	valAddrCodec := evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	consAddrCodec := evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ConsensusAddrPrefix())

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName:     nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.ModuleName:        {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		burntypes.ModuleName:           {authtypes.Burner},
		validatortypes.ModuleName:      nil,
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
		authority.String(),
		bankKeeper,
	)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(stakingKeeper)
	burnMsgServer := burnkeeper.NewMsgServerImpl(burnKeeper)

	validatorKeeper := validatorkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[validatortypes.StoreKey]),
		cdc,
		addrCodec,
		valAddrCodec,
		authority.String(),
		burnKeeper,
		stakingMsgServer,
	)

	// Build modules for InitGenesis
	authModule := auth.NewAppModule(cdc, accountKeeper, authsims.RandomGenesisAccounts, nil)
	bankModule := bank.NewAppModule(cdc, bankKeeper, accountKeeper, nil)
	stakingModule := staking.NewAppModule(cdc, stakingKeeper, accountKeeper, bankKeeper, nil)
	burnModule := burn.NewAppModule(cdc, burnKeeper, addrCodec)
	validatorModule := validator.NewAppModule(cdc, validatorKeeper, stakingKeeper)

	integrationApp := integration.NewIntegrationApp(newCtx, logger, keys, cdc, map[string]appmodule.AppModule{
		authtypes.ModuleName:      authModule,
		banktypes.ModuleName:      bankModule,
		stakingtypes.ModuleName:   stakingModule,
		burntypes.ModuleName:      burnModule,
		validatortypes.ModuleName: validatorModule,
	})

	sdkCtx := sdk.UnwrapSDKContext(integrationApp.Context())

	// Register MsgServers on the msg service router
	router := integrationApp.MsgServiceRouter()
	stakingtypes.RegisterMsgServer(router, stakingMsgServer)
	burntypes.RegisterMsgServer(router, burnMsgServer)
	validatortypes.RegisterMsgServer(router, validatorkeeper.NewMsgServerImpl(validatorKeeper))

	// Set circuit breaker - the code under test
	router.SetCircuit(validatorante.NewStakingCircuitBreaker())

	// Default params
	if err := stakingKeeper.SetParams(sdkCtx, stakingtypes.DefaultParams()); err != nil {
		assert.NilError(tb, err)
	}
	if err := validatorKeeper.Params.Set(sdkCtx, validatortypes.NewParams(
		sdk.NewCoin(sdk.DefaultBondDenom, math.NewIntWithDecimal(100_000, 18)),
		sdk.NewCoin(sdk.DefaultBondDenom, math.NewIntWithDecimal(100_000, 18)),
	)); err != nil {
		assert.NilError(tb, err)
	}
	if err := burnKeeper.Params.Set(sdkCtx, burntypes.DefaultParams()); err != nil {
		assert.NilError(tb, err)
	}

	return &fixture{
		app:             integrationApp,
		sdkCtx:          sdkCtx,
		cdc:             cdc,
		keys:            keys,
		accountKeeper:   accountKeeper,
		bankKeeper:      bankKeeper,
		stakingKeeper:   stakingKeeper,
		burnKeeper:      burnKeeper,
		validatorKeeper: validatorKeeper,
	}
}
