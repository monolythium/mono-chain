package burn_test

import (
	"os"
	"testing"

	cmtprototypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"gotest.tools/v3/assert"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
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
	evmaddress "github.com/cosmos/evm/encoding/address"

	"github.com/monolythium/mono-chain/app"
	burn "github.com/monolythium/mono-chain/x/burn"
	burnkeeper "github.com/monolythium/mono-chain/x/burn/keeper"
	burntypes "github.com/monolythium/mono-chain/x/burn/types"
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
	burnKeeper    burnkeeper.Keeper
}

func initFixture(tb testing.TB) *fixture {
	tb.Helper()

	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, burntypes.StoreKey,
	)

	cdc := moduletestutil.MakeTestEncodingConfig(
		auth.AppModuleBasic{},
		burn.AppModule{},
	).Codec

	logger := log.NewTestLogger(tb)
	cms := integration.CreateMultiStore(keys, logger)
	newCtx := sdk.NewContext(cms, cmtprototypes.Header{}, true, logger)

	authority := authtypes.NewModuleAddress(burntypes.GovModuleName)
	addrCodec := evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName: nil,
		minttypes.ModuleName:       {authtypes.Minter},
		burntypes.ModuleName:       {authtypes.Burner},
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

	burnKeeper := burnkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[burntypes.StoreKey]),
		cdc,
		addrCodec,
		authority.String(),
		bankKeeper,
	)

	authModule := auth.NewAppModule(cdc, accountKeeper, authsims.RandomGenesisAccounts, nil)
	bankModule := bank.NewAppModule(cdc, bankKeeper, accountKeeper, nil)
	burnModule := burn.NewAppModule(cdc, burnKeeper, addrCodec)

	integrationApp := integration.NewIntegrationApp(newCtx, logger, keys, cdc, map[string]appmodule.AppModule{
		authtypes.ModuleName: authModule,
		banktypes.ModuleName: bankModule,
		burntypes.ModuleName: burnModule,
	})

	sdkCtx := sdk.UnwrapSDKContext(integrationApp.Context())

	burnMsgServer := burnkeeper.NewMsgServerImpl(burnKeeper)
	burntypes.RegisterMsgServer(integrationApp.MsgServiceRouter(), burnMsgServer)

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
		burnKeeper:    burnKeeper,
	}
}
