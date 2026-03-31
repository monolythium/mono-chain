package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	cmtcfg "github.com/cometbft/cometbft/config"
	evmconfig "github.com/cosmos/evm/config"
	"github.com/spf13/cobra"
)

const defaultMinGasPrices = "10000000000alyth"

// initCometBFTConfig helps to override default CometBFT Config values.
// return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()
	return cfg
}

// initAppConfig returns the EVM-extended app config template and default values.
// This adds JSON-RPC, TLS, and EVM config sections to app.toml.
func initAppConfig() (string, interface{}) {
	tmpl, cfg := evmconfig.InitAppConfig("alyth", evmconfig.EVMChainID)

	// Override the library default ("0alyth") with a production-safe minimum.
	// Validators can still override this in their own app.toml.
	appCfg, ok := cfg.(evmconfig.EVMAppConfig)
	if ok {
		appCfg.Config.MinGasPrices = defaultMinGasPrices
		return tmpl, appCfg
	}

	return tmpl, cfg
}

// wrapInitCmd adds a post-init hook that patches app.toml with the correct
// evm-chain-id derived from --chain-id (e.g. "mono_6940-1" → 6940).
func wrapInitCmd(initCmd *cobra.Command) *cobra.Command {
	originalRunE := initCmd.RunE
	initCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := originalRunE(cmd, args); err != nil {
			return err
		}

		chainID, _ := cmd.Flags().GetString("chain-id")
		evmID, ok := parseEVMChainID(chainID)
		if !ok {
			return nil
		}

		homeDir, _ := cmd.Flags().GetString("home")
		if homeDir == "" {
			homeDir = os.ExpandEnv("$HOME/.mono")
		}

		appTomlPath := filepath.Join(homeDir, "config", "app.toml")
		data, err := os.ReadFile(appTomlPath)
		if err != nil {
			return fmt.Errorf("failed to read app.toml: %w", err)
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "evm-chain-id") {
				lines[i] = fmt.Sprintf("evm-chain-id = %d", evmID)
				break
			}
		}

		if err := os.WriteFile(appTomlPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return fmt.Errorf("failed to write app.toml: %w", err)
		}

		fmt.Printf("Set evm-chain-id = %d (derived from %s)\n", evmID, chainID)
		return nil
	}
	return initCmd
}

// parseEVMChainID extracts the EVM chain ID from a Cosmos chain ID.
// Format: "{name}_{evm-chain-id}-{version}" e.g. "mono_6940-1" → 6940.
func parseEVMChainID(chainID string) (uint64, bool) {
	parts := strings.SplitN(chainID, "_", 2)
	if len(parts) != 2 {
		return 0, false
	}
	numParts := strings.SplitN(parts[1], "-", 2)
	if len(numParts) < 1 {
		return 0, false
	}
	id, err := strconv.ParseUint(numParts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
