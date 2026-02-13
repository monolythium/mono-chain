package app

import (
	"encoding/json"
	"fmt"
	"io"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkaddress "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/monolythium/mono-chain/docs"
	monoencoding "github.com/monolythium/mono-chain/encoding"
	burnmodulekeeper "github.com/monolythium/mono-chain/x/burn/keeper"
	burnmodule "github.com/monolythium/mono-chain/x/burn/module"
	burnmoduletypes "github.com/monolythium/mono-chain/x/burn/types"
	monomodulekeeper "github.com/monolythium/mono-chain/x/mono/keeper"
	monomodule "github.com/monolythium/mono-chain/x/mono/module"
	monomoduletypes "github.com/monolythium/mono-chain/x/mono/types"
)

var (
	_ runtime.AppI            = (*App)(nil)
	_ servertypes.Application = (*App)(nil)
)

// App extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
// App extends BaseApp with explicit module wiring.
type App struct {
	*baseapp.BaseApp
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	// module management
	ModuleManager      *module.Manager
	BasicModuleManager module.BasicManager
	sm                 *module.SimulationManager

	// store keys
	keys map[string]*storetypes.KVStoreKey

	// keepers
	ConsensusParamsKeeper consensuskeeper.Keeper
	AuthKeeper            authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	BurnKeeper            burnmodulekeeper.Keeper
	MonoKeeper            monomodulekeeper.Keeper
}

// New returns a reference to an initialized App.
func New(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *App {
	// Encoding config
	encodingConfig := monoencoding.MakeConfig()
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry
	appCodec := encodingConfig.Codec
	txConfig := encodingConfig.TxConfig

	// add to default baseapp options
	// enable optimistic execution
	// BaseApp
	baseAppOptions = append(baseAppOptions, baseapp.SetOptimisticExecution())
	bApp := baseapp.NewBaseApp(
		AppName,
		logger,
		db,
		txConfig.TxDecoder(),
		baseAppOptions...,
	)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(txConfig.TxEncoder())

	// Store keys
	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		distrtypes.StoreKey,
		consensustypes.StoreKey,
		burnmoduletypes.StoreKey,
		monomoduletypes.StoreKey,
	)
	bApp.MountKVStores(keys)

	app := &App{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		txConfig:          txConfig,
		interfaceRegistry: interfaceRegistry,
		keys:              keys,
	}

	// Address codecs
	addressCodec := sdkaddress.NewBech32Codec(Bech32Prefix)
	validatorAddressCodec := sdkaddress.NewBech32Codec(Bech32PrefixValAddr)
	consensusAddressCodec := sdkaddress.NewBech32Codec(Bech32PrefixConsAddr)

	// Authority for governance-gated messages (gov module account address).
	// Valid even without the gov module installed — it's a deterministic address.
	authAddr := authtypes.NewModuleAddress("gov").String()

	// Keepers (dependency order)
	app.ConsensusParamsKeeper = consensuskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[consensustypes.StoreKey]),
		authAddr,
		runtime.EventService{},
	)
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	app.AuthKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		GetMaccPerms(),
		addressCodec,
		Bech32Prefix,
		authAddr,
	)

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AuthKeeper,
		BlockedAddresses(),
		authAddr,
		logger,
	)

	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		app.AuthKeeper,
		app.BankKeeper,
		authAddr,
		validatorAddressCodec,
		consensusAddressCodec,
	)

	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		app.AuthKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		authtypes.FeeCollectorName,
		authAddr,
	)

	app.BurnKeeper = burnmodulekeeper.NewKeeper(
		runtime.NewKVStoreService(keys[burnmoduletypes.StoreKey]),
		appCodec,
		addressCodec,
		authtypes.NewModuleAddress("gov"),
		app.BankKeeper,
	)

	app.MonoKeeper = monomodulekeeper.NewKeeper(
		runtime.NewKVStoreService(keys[monomoduletypes.StoreKey]),
		appCodec,
		addressCodec,
		authtypes.NewModuleAddress("gov"),
		app.BankKeeper,
		app.StakingKeeper,
	)

	// Staking hooks for distribution slashing
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			app.DistrKeeper.Hooks(),
		),
	)

	// Module manager
	app.ModuleManager = module.NewManager(
		auth.NewAppModule(appCodec, app.AuthKeeper, authsims.RandomGenesisAccounts, nil),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AuthKeeper, nil),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AuthKeeper, app.BankKeeper, nil),
		distribution.NewAppModule(appCodec, app.DistrKeeper, app.AuthKeeper, app.BankKeeper, app.StakingKeeper, nil),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		vesting.NewAppModule(app.AuthKeeper, app.BankKeeper),
		genutil.NewAppModule(app.AuthKeeper, app.StakingKeeper, app, txConfig),
		burnmodule.NewAppModule(appCodec, app.BurnKeeper, app.AuthKeeper, app.BankKeeper),
		monomodule.NewAppModule(appCodec, app.MonoKeeper, app.AuthKeeper, app.BankKeeper),
	)

	// BasicModuleManager handles codec registration, genesis validation, and gateway routes.
	app.BasicModuleManager = module.NewBasicManagerFromManager(
		app.ModuleManager,
		map[string]module.AppModuleBasic{
			genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		},
	)
	app.BasicModuleManager.RegisterLegacyAminoCodec(legacyAmino)
	app.BasicModuleManager.RegisterInterfaces(interfaceRegistry)

	// Module ordering
	app.ModuleManager.SetOrderPreBlockers(
		authtypes.ModuleName,
	)
	app.ModuleManager.SetOrderBeginBlockers(
		monomoduletypes.ModuleName,
		burnmoduletypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
	)
	app.ModuleManager.SetOrderEndBlockers(
		monomoduletypes.ModuleName,
		burnmoduletypes.ModuleName,
		stakingtypes.ModuleName,
	)
	app.ModuleManager.SetOrderInitGenesis(
		consensustypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		vestingtypes.ModuleName,
		burnmoduletypes.ModuleName,
		monomoduletypes.ModuleName,
		genutiltypes.ModuleName,
	)

	// Service registration
	cfg := module.NewConfigurator(appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	if err := app.ModuleManager.RegisterServices(cfg); err != nil {
		panic(err)
	}

	/**** Module Options ****/
	// ABCI lifecycle
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.SetInitChainer(app.InitChainer)

	/**** Module Options ****/
	// Set custom ante handler (SkipAnteHandler=true in tx config disables the default)
	app.setAnteHandler(txConfig)

	// create the simulation manager and define the order of the modules for deterministic simulations
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, make(map[string]module.AppModuleSimulation))
	app.sm.RegisterStoreDecoders()

	// A custom InitChainer can be set if extra pre-init-genesis logic is required.
	// By default, when using app wiring enabled module, this is not required.
	// For instance, the upgrade module will set automatically the module version map in its init genesis thanks to app wiring.
	// However, when registering a module manually (i.e. that does not support app wiring), the module version map
	// must be set manually as follow. The upgrade module will de-duplicate the module version map.
	//
	// app.SetInitChainer(func(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	// 	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap())
	// 	return app.App.InitChainer(ctx, req)
	// })

	// Load
	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			panic(err)
		}
	}

	return app
}

// PreBlocker wraps ModuleManager.PreBlock to match the sdk.PreBlocker signature.
func (app *App) PreBlocker(ctx sdk.Context, _ *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	return app.ModuleManager.PreBlock(ctx)
}

// BeginBlocker wraps ModuleManager.BeginBlock.
func (app *App) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.ModuleManager.BeginBlock(ctx)
}

// EndBlocker wraps ModuleManager.EndBlock.
func (app *App) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.ModuleManager.EndBlock(ctx)
}

// LoadHeight loads the app at a given height.
func (app *App) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// InitChainer handles chain init from genesis state.
func (app *App) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState map[string]json.RawMessage
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	return app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
}

// ExecuteGenesisTx implements genesis.TxHandler for genutil module.
func (app *App) ExecuteGenesisTx(txBytes []byte) error {
	decodedTx, err := app.txConfig.TxDecoder()(txBytes)
	if err != nil {
		return err
	}
	bz, err := app.txConfig.TxEncoder()(decodedTx)
	if err != nil {
		return err
	}
	res, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Txs:    [][]byte{bz},
		Height: app.LastBlockHeight() + 1,
	})
	if err != nil {
		return err
	}
	if len(res.TxResults) != 1 {
		return fmt.Errorf("unexpected number of tx results: %d", len(res.TxResults))
	}
	if res.TxResults[0].Code != abci.CodeTypeOK {
		return fmt.Errorf("genesis tx failed: %s", res.TxResults[0].Log)
	}
	return nil
}

// DefaultGenesis returns default genesis state for all modules.
func (app *App) DefaultGenesis() map[string]json.RawMessage {
	return app.BasicModuleManager.DefaultGenesis(app.appCodec)
}

// LegacyAmino returns App's amino codec.
func (app *App) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns App's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *App) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns App's InterfaceRegistry.
func (app *App) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns App's TxConfig
func (app *App) TxConfig() client.TxConfig {
	return app.txConfig
}

// GetKey returns the KVStoreKey for the provided store key.
func (app *App) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetStoreKeys returns all the stored store keys.
func (app *App) GetStoreKeys() []storetypes.StoreKey {
	keys := make([]storetypes.StoreKey, 0, len(app.keys))
	for _, key := range app.keys {
		keys = append(keys, key)
	}

	return keys
}

// SimulationManager implements the SimulationApp interface
func (app *App) SimulationManager() *module.SimulationManager {
	return app.sm
}

// AutoCliOpts returns options for the AutoCLI module.
func (app *App) AutoCliOpts() autocli.AppOptions {
	modules := make(map[string]appmodule.AppModule)
	for _, m := range app.ModuleManager.Modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			if appModule, ok := m.(appmodule.AppModule); ok {
				modules[moduleWithName.Name()] = appModule
			}
		}
	}
	return autocli.AppOptions{
		Modules:               modules,
		ModuleOptions:         runtimeservices.ExtractAutoCLIOptions(app.ModuleManager.Modules),
		AddressCodec:          sdkaddress.NewBech32Codec(Bech32Prefix),
		ValidatorAddressCodec: sdkaddress.NewBech32Codec(Bech32PrefixValAddr),
		ConsensusAddressCodec: sdkaddress.NewBech32Codec(Bech32PrefixConsAddr),
	}
}

// RegisterAPIRoutes registers all application module routes with the provided API server.
func (app *App) RegisterAPIRoutes(apiSvr *api.Server, apiConfig serverconfig.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	app.BasicModuleManager.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	if err := server.RegisterSwaggerAPI(apiSvr.ClientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}

	// register app's OpenAPI routes.
	docs.RegisterOpenAPIService(AppName, apiSvr.Router)
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *App) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.GRPCQueryRouter(), clientCtx, app.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *App) RegisterTendermintService(clientCtx client.Context) {
	cmtApp := server.NewCometABCIWrapper(app)
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.GRPCQueryRouter(),
		app.interfaceRegistry,
		cmtApp.Query,
	)
}

// RegisterNodeService implements the Application.RegisterNodeService method.
func (app *App) RegisterNodeService(clientCtx client.Context, cfg serverconfig.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// setAnteHandler creates and sets the custom ante handler chain.
func (app *App) setAnteHandler(txConfig client.TxConfig) {
	anteHandler, err := NewAnteHandler(HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   app.AuthKeeper,
			BankKeeper:      app.BankKeeper,
			SignModeHandler: txConfig.SignModeHandler(),
			SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
		},
		MonoKeeper: app.MonoKeeper,
	})
	if err != nil {
		panic(err)
	}
	app.SetAnteHandler(anteHandler)
}
