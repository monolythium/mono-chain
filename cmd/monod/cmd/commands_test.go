package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtcfg "github.com/cometbft/cometbft/config"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	cosmosevmserver "github.com/cosmos/evm/server"

	"github.com/monolythium/mono-chain/app"
)

const testCommittedHeight = 1

// testApp bootstraps a real app instance for tests that need txConfig or basicManager.
func testApp(t *testing.T) *app.App {
	t.Helper()
	return app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.EmptyAppOptions{})
}

func testBasicManager(t *testing.T, a *app.App) module.BasicManager {
	t.Helper()
	return module.NewBasicManagerFromManager(a.ModuleManager, map[string]module.AppModuleBasic{
		genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
	})
}

func newFlagSet(t *testing.T) *pflag.FlagSet {
	t.Helper()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String(FlagNetworkID, "", "")
	fs.String(flags.FlagChainID, "", "")
	return fs
}

func TestSetNetworkIDFlag_EmptyIsNoOp(t *testing.T) {
	fs := newFlagSet(t)
	require.NoError(t, setNetworkIDFlag(fs))
	chainID, err := fs.GetString(flags.FlagChainID)
	require.NoError(t, err)
	require.Empty(t, chainID)
}

func TestSetNetworkIDFlag_PropagatesChainID(t *testing.T) {
	fs := newFlagSet(t)
	require.NoError(t, fs.Set(FlagNetworkID, "mono_6940-1"))
	require.NoError(t, setNetworkIDFlag(fs))
	chainID, err := fs.GetString(flags.FlagChainID)
	require.NoError(t, err)
	require.Equal(t, "mono_6940-1", chainID)
}

func TestSetNetworkIDFlag_ConflictErrors(t *testing.T) {
	fs := newFlagSet(t)
	require.NoError(t, fs.Set(FlagNetworkID, "mono_6940-1"))
	require.NoError(t, fs.Set(flags.FlagChainID, "mono_6940-1"))
	require.ErrorIs(t, setNetworkIDFlag(fs), ErrInvalidFlagsCombo)
}

func TestSetNetworkIDFlag_FlagAbsentErrors(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String(flags.FlagChainID, "", "")
	require.Error(t, setNetworkIDFlag(fs))
}

func TestSetNetworkIDFlag_PreservesExistingChainIDWhenEmpty(t *testing.T) {
	fs := newFlagSet(t)
	require.NoError(t, fs.Set(flags.FlagChainID, "already-set"))
	require.NoError(t, setNetworkIDFlag(fs))
	chainID, err := fs.GetString(flags.FlagChainID)
	require.NoError(t, err)
	require.Equal(t, "already-set", chainID)
}

func TestSetNetworkIDFlag_ConflictEvenWhenValuesMatch(t *testing.T) {
	fs := newFlagSet(t)
	require.NoError(t, fs.Set(FlagNetworkID, "mono_6940-1"))
	require.NoError(t, fs.Set(flags.FlagChainID, "mono_6940-1"))
	require.ErrorIs(t, setNetworkIDFlag(fs), ErrInvalidFlagsCombo)
}

func TestMonoInitCmd_HasNetworkIDFlag(t *testing.T) {
	cmd := monoInitCmd(nil)
	f := cmd.Flags().Lookup(FlagNetworkID)
	require.NotNil(t, f)
	require.Equal(t, "", f.DefValue)
}

func TestMonoInitCmd_HasRunE(t *testing.T) {
	cmd := monoInitCmd(nil)
	require.NotNil(t, cmd.RunE)
}

func TestMonoInitCmd_UseLine(t *testing.T) {
	cmd := monoInitCmd(nil)
	require.Equal(t, "init", cmd.Name())
}

// wiredInitCmd returns a monoInitCmd wired with real client+server contexts
// and a temp home directory. Mirrors the setup root.go's PersistentPreRunE provides.
func wiredInitCmd(t *testing.T) (cmd *cobra.Command, homeDir string) {
	t.Helper()
	a := testApp(t)
	bm := testBasicManager(t, a)

	homeDir = t.TempDir()
	cmtCfg := cmtcfg.DefaultConfig()
	cmtCfg.SetRoot(homeDir)
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, "config"), 0o755))

	cmd = monoInitCmd(bm)
	clientCtx := client.Context{}.
		WithCodec(a.AppCodec()).
		WithInterfaceRegistry(a.InterfaceRegistry()).
		WithLegacyAmino(a.LegacyAmino()).
		WithTxConfig(a.TxConfig()).
		WithHomeDir(homeDir)
	serverCtx := server.NewContext(viper.New(), cmtCfg, log.NewNopLogger())
	ctx := context.Background()
	ctx = context.WithValue(ctx, client.ClientContextKey, &clientCtx)
	ctx = context.WithValue(ctx, server.ServerContextKey, serverCtx)
	cmd.SetContext(ctx)
	return cmd, homeDir
}

func TestMonoInitCmd_RunEReturnsErrorOnConflict(t *testing.T) {
	cmd, homeDir := wiredInitCmd(t)
	cmd.SetArgs([]string{"test-moniker", "--network-id", "mono_6940-1", "--chain-id", "mono_6940-1", "--home", homeDir})
	require.ErrorIs(t, cmd.Execute(), ErrInvalidFlagsCombo)
}

func TestMonoInitCmd_IntegrationWithNetworkID(t *testing.T) {
	cmd, homeDir := wiredInitCmd(t)
	cmd.SetArgs([]string{"test-moniker", "--network-id", "mono_6940-1", "--home", homeDir})
	require.NoError(t, cmd.Execute())

	genDoc, err := genutiltypes.AppGenesisFromFile(filepath.Join(homeDir, "config", "genesis.json"))
	require.NoError(t, err)
	require.Equal(t, "mono_6940-1", genDoc.ChainID)
}

func TestQueryCommand_Structure(t *testing.T) {
	cmd := queryCommand()
	require.Equal(t, "query", cmd.Use)
	require.Contains(t, cmd.Aliases, "q")
	require.NotNil(t, cmd.RunE)

	// Verify expected subcommands are registered
	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}
	for _, expected := range []string{"wait-tx", "block", "txs", "tx", "blocks", "block-results"} {
		require.True(t, subNames[expected], "query must have subcommand %q", expected)
	}
}

func TestTxCommand_Structure(t *testing.T) {
	cmd := txCommand()
	require.Equal(t, "tx", cmd.Use)
	require.NotNil(t, cmd.RunE)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}
	for _, expected := range []string{"sign", "sign-batch", "multi-sign", "broadcast", "encode", "decode"} {
		require.True(t, subNames[expected], "tx must have subcommand %q", expected)
	}
}

func TestAddModuleInitFlags_DoesNotPanic(t *testing.T) {
	require.NotPanics(t, func() {
		addModuleInitFlags(&cobra.Command{})
	})
}

func initTestHome(t *testing.T) string {
	t.Helper()
	cmd, homeDir := wiredInitCmd(t)
	cmd.SetArgs([]string{"test-moniker", "--network-id", "mono_6940-1", "--home", homeDir})
	require.NoError(t, cmd.Execute())
	return homeDir
}

func serverAppOpts(t *testing.T) *viper.Viper {
	t.Helper()
	home := initTestHome(t)
	v := viper.New()
	v.Set("pruning", "nothing")
	v.Set(flags.FlagHome, home)
	return v
}

func TestNewApp_ReturnsApplication(t *testing.T) {
	result := newApp(log.NewNopLogger(), dbm.NewMemDB(), nil, serverAppOpts(t))
	require.NotNil(t, result)
	var _ cosmosevmserver.Application = result
}

func TestSdkNewApp_ReturnsApplication(t *testing.T) {
	result := sdkNewApp(log.NewNopLogger(), dbm.NewMemDB(), nil, serverAppOpts(t))
	require.NotNil(t, result)
	var _ servertypes.Application = result
}

func TestAppExport_ErrorOnNonViperOpts(t *testing.T) {
	_, err := appExport(log.NewNopLogger(), dbm.NewMemDB(), nil, -1, false, nil, simtestutil.EmptyAppOptions{}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "application home not set")
}

func TestAppExport_ErrorOnEmptyHome(t *testing.T) {
	v := viper.New()
	_, err := appExport(log.NewNopLogger(), dbm.NewMemDB(), nil, -1, false, nil, v, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "application home not set")
}

// initChainedDB sets up a memdb with a fully init-chained app (validator, correct denoms).
// Follows the same pattern as app_test.go TestEVMLifecycle.
// Uses the given appOpts so the db state is compatible with a second app.New call.
func initChainedDB(t *testing.T, appOpts servertypes.AppOptions) dbm.DB {
	t.Helper()
	db := dbm.NewMemDB()
	a := app.New(log.NewNopLogger(), db, nil, true, appOpts)

	// Genesis validator
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err)
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{cmttypes.NewValidator(pubKey, 1)})

	// Genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(app.DefaultBondDenom, math.NewIntWithDecimal(1, 18))),
	}

	genesisState := a.DefaultGenesis()
	genesisState, err = simtestutil.GenesisStateWithValSet(a.AppCodec(), genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)
	require.NoError(t, err)

	// Patch EVM denom
	var evmGenState evmtypes.GenesisState
	a.AppCodec().MustUnmarshalJSON(genesisState[evmtypes.ModuleName], &evmGenState)
	evmGenState.Params.EvmDenom = app.DefaultBondDenom
	evmGenState.Params.ExtendedDenomOptions.ExtendedDenom = app.DefaultBondDenom
	genesisState[evmtypes.ModuleName] = a.AppCodec().MustMarshalJSON(&evmGenState)

	// Bank denom metadata
	var bankGenState banktypes.GenesisState
	a.AppCodec().MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGenState)
	bankGenState.DenomMetadata = []banktypes.Metadata{{
		Base:    app.DefaultBondDenom,
		Display: "alyth",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: app.DefaultBondDenom, Exponent: 0},
			{Denom: "alyth", Exponent: 18},
		},
	}}
	genesisState[banktypes.ModuleName] = a.AppCodec().MustMarshalJSON(&bankGenState)

	stateBytes, err := json.Marshal(genesisState)
	require.NoError(t, err)

	_, err = a.InitChain(&abci.RequestInitChain{
		AppStateBytes:   stateBytes,
		ConsensusParams: simtestutil.DefaultConsensusParams,
	})
	require.NoError(t, err)

	// InitChain state must be finalized through at least one block cycle
	// before it's fully committed to the multistore (same as app_test.go).
	_, err = a.FinalizeBlock(&abci.RequestFinalizeBlock{Height: testCommittedHeight})
	require.NoError(t, err)
	_, err = a.Commit()
	require.NoError(t, err)
	return db
}

// mapAppOpts satisfies servertypes.AppOptions with a simple map.
// Used to hit the "appOpts is not viper.Viper" branch in appExport.
type mapAppOpts map[string]interface{}

func (m mapAppOpts) Get(key string) interface{} { return m[key] }

func TestAppExport_ErrorOnNonViperWithHome(t *testing.T) {
	opts := mapAppOpts{flags.FlagHome: "/some/path"}
	_, err := appExport(log.NewNopLogger(), dbm.NewMemDB(), nil, -1, false, nil, opts, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "appOpts is not viper.Viper")
}

func TestAppExport_HappyPaths(t *testing.T) {
	home := initTestHome(t)

	v := viper.New()
	v.Set("pruning", "nothing")
	v.Set(flags.FlagHome, home)

	db := initChainedDB(t, v)

	t.Run("height -1 exports latest", func(t *testing.T) {
		exported, err := appExport(log.NewNopLogger(), db, nil, -1, false, nil, v, nil)
		require.NoError(t, err)
		require.NotNil(t, exported.AppState)
	})

	t.Run("specific height exports from committed block", func(t *testing.T) {
		exported, err := appExport(log.NewNopLogger(), db, nil, 1, false, nil, v, nil)
		require.NoError(t, err)
		require.NotNil(t, exported.AppState)
	})

	t.Run("non-existent height returns error", func(t *testing.T) {
		exported, err := appExport(log.NewNopLogger(), db, nil, testCommittedHeight+1, false, nil, v, nil)
		require.Error(t, err)
		require.Nil(t, exported.AppState)
	})
}

func TestInitRootCmd_RegistersExpectedSubcommands(t *testing.T) {
	a := testApp(t)
	bm := testBasicManager(t, a)

	rootCmd := &cobra.Command{Use: "testd"}
	initRootCmd(rootCmd, a.TxConfig(), bm)

	subNames := make(map[string]bool)
	for _, sub := range rootCmd.Commands() {
		subNames[sub.Name()] = true
	}

	for _, expected := range []string{
		"init",      // monoInitCmd
		"config",    // confixcmd
		"prune",     // pruning
		"snapshots", // snapshot
		"start",     // from cosmosevmserver.AddCommands
		"status",    // server.StatusCommand
		"genesis",   // genutilcli.Commands
		"query",     // queryCommand
		"tx",        // txCommand
		"keys",      // cosmosevmcmd.KeyCommands
	} {
		require.True(t, subNames[expected], "root must have subcommand %q, got: %v", expected, subNames)
	}
}

func TestInitRootCmd_PanicsOnAddTxFlagsError(t *testing.T) {
	a := testApp(t)
	bm := testBasicManager(t, a)

	rootCmd := &cobra.Command{Use: "testd"}
	var seen bool
	rootCmd.PersistentFlags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		if name == flags.FlagNode && seen {
			return "node-break"
		}
		if name == flags.FlagNode {
			seen = true
		}
		return pflag.NormalizedName(name)
	})

	require.Panics(t, func() {
		initRootCmd(rootCmd, a.TxConfig(), bm)
	})
}

func TestInitRootCmd_InitCmdHasNetworkIDFlag(t *testing.T) {
	a := testApp(t)
	bm := testBasicManager(t, a)

	rootCmd := &cobra.Command{Use: "testd"}
	initRootCmd(rootCmd, a.TxConfig(), bm)

	// Find the init subcommand and verify it has --network-id
	var initCmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "init" {
			initCmd = sub
			break
		}
	}
	require.NotNil(t, initCmd)
	require.NotNil(t, initCmd.Flags().Lookup(FlagNetworkID))
}
