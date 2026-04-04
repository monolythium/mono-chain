package cmd

import (
	"errors"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"cosmossdk.io/log"
	confixcmd "cosmossdk.io/tools/confix/cmd"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	cosmosevmcmd "github.com/cosmos/evm/client"
	cosmosevmserver "github.com/cosmos/evm/server"
	srvflags "github.com/cosmos/evm/server/flags"

	"github.com/monolythium/mono-chain/app"
)

func initRootCmd(
	rootCmd *cobra.Command,
	txConfig client.TxConfig,
	basicManager module.BasicManager,
) {
	rootCmd.AddCommand(
		monoInitCmd(basicManager),
		NewInPlaceTestnetCmd(),
		NewTestnetMultiNodeCmd(basicManager, banktypes.GenesisBalancesIterator{}),
		confixcmd.ConfigCommand(),
		pruning.Cmd(sdkNewApp, app.DefaultNodeHome),
		snapshot.Cmd(sdkNewApp),
	)

	// cosmos/evm server commands (replaces standard server.AddCommands)
	cosmosevmserver.AddCommands(
		rootCmd,
		cosmosevmserver.NewDefaultStartOptions(newApp, app.DefaultNodeHome),
		appExport,
		addModuleInitFlags,
	)

	// cosmos/evm key commands (supports eth_secp256k1)
	rootCmd.AddCommand(
		cosmosevmcmd.KeyCommands(app.DefaultNodeHome, true),
	)

	// add keybase, auxiliary RPC, query, genesis, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		genutilcli.Commands(txConfig, basicManager, app.DefaultNodeHome),
		queryCommand(),
		txCommand(),
	)

	// EVM-specific transaction flags
	if _, err := srvflags.AddTxFlags(rootCmd); err != nil {
		panic(err)
	}
}

// monoInitCmd wraps the SDK's init command with --network-id support.
// When provided, --network-id propagates to --chain-id for genesis.json.
func monoInitCmd(basicManager module.BasicManager) *cobra.Command {
	initCmd := genutilcli.InitCmd(basicManager, app.DefaultNodeHome)
	initCmd.Flags().String(FlagNetworkID, "", FlagNetworkIDUse)

	originalRunE := initCmd.RunE
	initCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := setNetworkIDFlag(cmd.Flags()); err != nil {
			return err
		}

		return originalRunE(cmd, args)
	}

	return initCmd
}

// setNetworkIDFlag propagates --network-id to --chain-id, erroring on conflicts.
func setNetworkIDFlag(fs *pflag.FlagSet) error {
	networkID, err := fs.GetString(FlagNetworkID)
	if err != nil {
		return err
	}

	if networkID == "" {
		return nil
	}

	if fs.Changed(flags.FlagChainID) {
		return ErrInvalidFlagsCombo
	}

	return fs.Set(flags.FlagChainID, networkID)
}

// addModuleInitFlags adds more flags to the start command.
func addModuleInitFlags(startCmd *cobra.Command) {}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		rpc.WaitTxCmd(),
		rpc.ValidatorCommand(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		server.QueryBlocksCmd(),
		authcmd.QueryTxCmd(),
		server.QueryBlockResultsCmd(),
	)
	return cmd
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		flags.LineBreak,
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
	)
	return cmd
}

// newApp creates the application for cosmos/evm server (returns cosmosevmserver.Application)
func newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) cosmosevmserver.Application {
	baseappOptions := server.DefaultBaseappOptions(appOpts)
	return app.New(
		logger, db, traceStore, true,
		appOpts,
		baseappOptions...,
	)
}

// sdkNewApp wraps for SDK's pruning/snapshot commands that expect servertypes.Application
func sdkNewApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	return newApp(logger, db, traceStore, appOpts)
}

func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	var bApp *app.App

	// this check is necessary as we use the flag in x/upgrade.
	// we can exit more gracefully by checking the flag here.
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home not set")
	}

	viperAppOpts, ok := appOpts.(*viper.Viper)
	if !ok {
		return servertypes.ExportedApp{}, errors.New("appOpts is not viper.Viper")
	}

	appOpts = viperAppOpts
	if height != -1 {
		bApp = app.New(logger, db, traceStore, false, appOpts)
		if err := bApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		bApp = app.New(logger, db, traceStore, true, appOpts)
	}

	return bApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}
