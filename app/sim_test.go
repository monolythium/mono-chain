//go:build sims

package app

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"io"
	"math/rand"
	"strings"
	"sync"
	"testing"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/feegrant"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sims "github.com/cosmos/cosmos-sdk/testutil/simsx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/evm/x/vm"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const SimAppChainID = "mono-simapp"

var FlagEnableStreamingValue bool

func init() {
	simcli.GetSimulatorFlags()
	flag.BoolVar(&FlagEnableStreamingValue, "EnableStreaming", false, "Enable streaming service")
}

// evmProcessInit is a process-level sync.Once shared across all app instances.
// cosmos/evm's AppModule creates a per-instance sync.Once that guards
// SetGlobalConfigVariables — a function that sets a process-level global
// (evmCoinInfo). When the sim framework creates multiple apps per process
// (one per seed, running in parallel), the second app panics:
// "EVM coin info already set". This process-level Once ensures the global
// is set exactly once regardless of how many apps are created.
var evmProcessInit sync.Once

// evmSimModule wraps vm.AppModule to replace its per-instance sync.Once
// with the process-level evmProcessInit. All other methods (BeginBlock,
// EndBlock, ExportGenesis, RegisterServices, etc.) delegate to the
// embedded original via Go method promotion.
type evmSimModule struct {
	vm.AppModule
	keeper *evmkeeper.Keeper
	ak     evmtypes.AccountKeeper
	bk     evmtypes.BankKeeper
}

func (m evmSimModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState evmtypes.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	vm.InitGenesis(ctx, m.keeper, m.ak, m.bk, genesisState, &evmProcessInit)
	return []abci.ValidatorUpdate{}
}

func (m evmSimModule) PreBlock(goCtx context.Context) (appmodule.ResponsePreBlock, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	coinInfo := m.keeper.GetEvmCoinInfo(ctx)
	evmProcessInit.Do(func() {
		vm.SetGlobalConfigVariables(coinInfo)
	})
	return &sdk.ResponsePreBlock{ConsensusParamsChanged: false}, nil
}

// simNew wraps the standard app constructor for simulation tests:
//  1. Disables the EVM mempool — sim framework delivers txs directly via
//     FinalizeBlock, bypassing mempool.Insert/Remove. Without this, subsequent
//     apps see GetChainConfig()!=nil and activate ExperimentalEVMMempool,
//     causing "tx not found in mempool" log spam on every Remove() call.
//  2. Replaces the EVM module with evmSimModule so parallel sim seeds share
//     one sync.Once for the EVM global initializer.
func simNew(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *App {
	wrapped := appOptionsFn(func(key string) any {
		if key == server.FlagMempoolMaxTxs {
			return -1
		}
		return appOpts.Get(key)
	})
	app := New(logger, db, traceStore, loadLatest, wrapped, baseAppOptions...)
	app.ModuleManager.Modules[evmtypes.ModuleName] = evmSimModule{
		AppModule: app.ModuleManager.Modules[evmtypes.ModuleName].(vm.AppModule),
		keeper:    app.EVMKeeper,
		ak:        app.AccountKeeper,
		bk:        app.BankKeeper,
	}
	return app
}

// interBlockCacheOpt returns a BaseApp option function that sets the persistent
// inter-block write-through cache.
func interBlockCacheOpt() func(*baseapp.BaseApp) {
	return baseapp.SetInterBlockCache(store.NewCommitKVStoreCacheManager())
}

func setupStateFactory(app *App) sims.SimStateFactory {
	return sims.SimStateFactory{
		Codec:         app.AppCodec(),
		AppStateFn:    SimDefaultGenesis(app),
		BlockedAddr:   BlockedAddresses(),
		AccountSource: app.AccountKeeper,
		BalanceSource: app.BankKeeper,
	}
}

func TestFullAppSimulation(t *testing.T) {
	sims.Run(t, simNew, setupStateFactory)
}

var (
	exportAllModules       = []string{}
	exportWithValidatorSet = []string{}
)

func TestAppImportExport(t *testing.T) {
	sims.Run(t, simNew, setupStateFactory, func(tb testing.TB, ti sims.TestInstance[*App], accs []simtypes.Account) {
		tb.Helper()
		app := ti.App
		tb.Log("exporting genesis...")
		exported, err := app.ExportAppStateAndValidators(false, exportWithValidatorSet, exportAllModules)
		require.NoError(tb, err)

		tb.Log("importing genesis...")
		newTestInstance := sims.NewSimulationAppInstance(tb, ti.Cfg, simNew)
		newApp := newTestInstance.App
		var genesisState GenesisState
		require.NoError(tb, json.Unmarshal(exported.AppState, &genesisState))
		ctxB := newApp.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
		_, err = newApp.ModuleManager.InitGenesis(ctxB, newApp.AppCodec(), genesisState)
		if isEmptyValidatorSetErr(err) {
			tb.Skip("Skipping simulation as all validators have been unbonded")
			return
		}
		require.NoError(tb, err)
		err = newApp.StoreConsensusParams(ctxB, exported.ConsensusParams)
		require.NoError(tb, err)

		tb.Log("comparing stores...")
		skipPrefixes := map[string][][]byte{
			upgradetypes.StoreKey: {
				[]byte{upgradetypes.VersionMapByte},
			},
			stakingtypes.StoreKey: {
				stakingtypes.UnbondingQueueKey, stakingtypes.RedelegationQueueKey, stakingtypes.ValidatorQueueKey,
				stakingtypes.HistoricalInfoKey, stakingtypes.UnbondingIDKey, stakingtypes.UnbondingIndexKey,
				stakingtypes.UnbondingTypeKey, stakingtypes.ValidatorUpdatesKey,
			},
			authzkeeper.StoreKey:   {authzkeeper.GrantQueuePrefix},
			feegrant.StoreKey:      {feegrant.FeeAllowanceQueueKeyPrefix},
			slashingtypes.StoreKey: {slashingtypes.ValidatorMissedBlockBitmapKeyPrefix},
		}
		assertEqualStores(tb, app, newApp, app.SimulationManager().StoreDecoders, skipPrefixes)
	})
}

// TestAppSimulationAfterImport runs simulation, exports state, imports into
// a fresh instance, and continues simulation on the imported state.
func TestAppSimulationAfterImport(t *testing.T) {
	sims.Run(t, simNew, setupStateFactory, func(tb testing.TB, ti sims.TestInstance[*App], accs []simtypes.Account) {
		tb.Helper()
		app := ti.App
		tb.Log("exporting genesis...")
		exported, err := app.ExportAppStateAndValidators(false, exportWithValidatorSet, exportAllModules)
		require.NoError(tb, err)

		tb.Log("importing genesis...")
		newTestInstance := sims.NewSimulationAppInstance(tb, ti.Cfg, simNew)
		newApp := newTestInstance.App
		_, err = newApp.InitChain(&abci.RequestInitChain{
			AppStateBytes: exported.AppState,
			ChainId:       SimAppChainID,
		})
		if isEmptyValidatorSetErr(err) {
			tb.Skip("Skipping simulation as all validators have been unbonded")
			return
		}
		require.NoError(tb, err)
		newStateFactory := setupStateFactory(newApp)
		_, _, err = simulation.SimulateFromSeedX(
			tb,
			newTestInstance.AppLogger,
			sims.WriteToDebugLog(newTestInstance.AppLogger),
			newApp.BaseApp,
			newStateFactory.AppStateFn,
			simtypes.RandomAccounts,
			simtestutil.BuildSimulationOperations(newApp, newApp.AppCodec(), newTestInstance.Cfg, newApp.TxConfig()),
			newStateFactory.BlockedAddr,
			newTestInstance.Cfg,
			newStateFactory.Codec,
			ti.ExecLogWriter,
		)
		require.NoError(tb, err)
	})
}

func isEmptyValidatorSetErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "validator set is empty after InitGenesis")
}

func TestAppStateDeterminism(t *testing.T) {
	const numTimesToRunPerSeed = 3
	var seeds []int64
	if s := simcli.NewConfigFromFlags().Seed; s != simcli.DefaultSeedValue {
		for j := 0; j < numTimesToRunPerSeed; j++ {
			seeds = append(seeds, s)
		}
	} else {
		for i := 0; i < 3; i++ {
			seed := rand.Int63()
			for j := 0; j < numTimesToRunPerSeed; j++ {
				seeds = append(seeds, seed)
			}
		}
	}
	interBlockCachingAppFactory := func(
		logger log.Logger,
		db dbm.DB,
		traceStore io.Writer,
		loadLatest bool,
		appOpts servertypes.AppOptions,
		baseAppOptions ...func(*baseapp.BaseApp),
	) *App {
		if FlagEnableStreamingValue {
			m := map[string]any{
				"streaming.abci.keys":             []string{"*"},
				"streaming.abci.plugin":           "abci_v1",
				"streaming.abci.stop-node-on-err": true,
			}
			others := appOpts
			appOpts = appOptionsFn(func(k string) any {
				if v, ok := m[k]; ok {
					return v
				}
				return others.Get(k)
			})
		}
		return simNew(logger, db, nil, true, appOpts, append(baseAppOptions, interBlockCacheOpt())...)
	}
	var mx sync.Mutex
	appHashResults := make(map[int64][][]byte)
	appSimLogger := make(map[int64][]simulation.LogWriter)
	captureAndCheckHash := func(tb testing.TB, ti sims.TestInstance[*App], _ []simtypes.Account) {
		tb.Helper()
		seed, appHash := ti.Cfg.Seed, ti.App.LastCommitID().Hash
		mx.Lock()
		otherHashes, execWriters := appHashResults[seed], appSimLogger[seed]
		if len(otherHashes) < numTimesToRunPerSeed-1 {
			appHashResults[seed], appSimLogger[seed] = append(otherHashes, appHash), append(execWriters, ti.ExecLogWriter)
		} else {
			delete(appHashResults, seed)
			delete(appSimLogger, seed)
		}
		mx.Unlock()

		var failNow bool
		for i := 0; i < len(otherHashes); i++ {
			if !assert.Equal(tb, otherHashes[i], appHash) {
				execWriters[i].PrintLogs()
				failNow = true
			}
		}
		if failNow {
			ti.ExecLogWriter.PrintLogs()
			tb.Fatalf("non-determinism in seed %d", seed)
		}
	}
	sims.RunWithSeeds(t, interBlockCachingAppFactory, setupStateFactory, seeds, []byte{}, captureAndCheckHash)
}

type comparableStoreApp interface {
	LastBlockHeight() int64
	NewContextLegacy(isCheckTx bool, header cmtproto.Header) sdk.Context
	GetKey(storeKey string) *storetypes.KVStoreKey
	GetStoreKeys() []storetypes.StoreKey
}

func assertEqualStores(
	tb testing.TB,
	app, newApp comparableStoreApp,
	storeDecoders simtypes.StoreDecoderRegistry,
	skipPrefixes map[string][][]byte,
) {
	tb.Helper()
	ctxA := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
	ctxB := newApp.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	storeKeys := app.GetStoreKeys()
	require.NotEmpty(tb, storeKeys)

	for _, appKeyA := range storeKeys {
		if _, ok := appKeyA.(*storetypes.KVStoreKey); !ok {
			continue
		}

		keyName := appKeyA.Name()
		appKeyB := newApp.GetKey(keyName)

		storeA := ctxA.KVStore(appKeyA)
		storeB := ctxB.KVStore(appKeyB)

		failedKVAs, failedKVBs := simtestutil.DiffKVStores(storeA, storeB, skipPrefixes[keyName])
		require.Equal(
			tb,
			len(failedKVAs),
			len(failedKVBs),
			"unequal sets of key-values to compare %s, key stores %s and %s", keyName, appKeyA, appKeyB,
		)

		tb.Logf("compared %d different key/value pairs between %s and %s\n", len(failedKVAs), appKeyA, appKeyB)
		if !assert.Equal(
			tb,
			0,
			len(failedKVAs),
			simtestutil.GetSimulationLog(keyName, storeDecoders, failedKVAs, failedKVBs),
		) {
			for _, v := range failedKVAs {
				tb.Logf("store mismatch: %q\n", v)
			}
			tb.FailNow()
		}
	}
}

// appOptionsFn is an adapter to the single method AppOptions interface.
type appOptionsFn func(string) any

func (f appOptionsFn) Get(k string) any {
	return f(k)
}

func FuzzFullAppSimulation(f *testing.F) {
	f.Fuzz(func(t *testing.T, rawSeed []byte) {
		if len(rawSeed) < 8 {
			t.Skip()
			return
		}
		sims.RunWithSeeds(
			t,
			simNew,
			setupStateFactory,
			[]int64{int64(binary.BigEndian.Uint64(rawSeed))},
			rawSeed[8:],
		)
	})
}
