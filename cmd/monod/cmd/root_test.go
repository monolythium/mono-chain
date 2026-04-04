package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestNewRootCmd_DoesNotPanic(t *testing.T) {
	require.NotPanics(t, func() {
		NewRootCmd()
	})
}

func TestNewRootCmd_InitWritesCorrectEVMChainID(t *testing.T) {
	homeDir := t.TempDir()
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"init", "test-moniker", "--network-id", "mono_6940-1", "--home", homeDir})
	require.NoError(t, rootCmd.Execute())

	appToml, err := os.ReadFile(filepath.Join(homeDir, "config", "app.toml"))
	require.NoError(t, err)
	require.Contains(t, string(appToml), "evm-chain-id = 6940")

	genDoc, err := os.ReadFile(filepath.Join(homeDir, "config", "genesis.json"))
	require.NoError(t, err)
	require.Contains(t, string(genDoc), `"chain_id": "mono_6940-1"`)
}

func TestNewRootCmd_InitInvalidNetworkIDErrors(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"init", "test-moniker", "--network-id", "INVALID", "--home", t.TempDir()})
	require.Error(t, rootCmd.Execute())
}

func TestNewRootCmd_BadKeyringBackendErrors(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"init", "test-moniker", "--home", t.TempDir(), "--" + flags.FlagKeyringBackend, "garbage"})
	require.Error(t, rootCmd.Execute())
}

func TestNewRootCmd_CorruptClientConfigErrors(t *testing.T) {
	homeDir := t.TempDir()
	cfgDir := filepath.Join(homeDir, "config")
	require.NoError(t, os.MkdirAll(cfgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "client.toml"), []byte("{{{{not toml"), 0o644))

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"init", "test-moniker", "--home", homeDir})
	require.Error(t, rootCmd.Execute())
}

func TestNewRootCmd_SetCmdClientContextHandlerError(t *testing.T) {
	// Poison normalize("keyring-backend") call #5 which is inside
	// SetCmdClientContextHandler's internal ReadPersistentCommandFlags.
	home := t.TempDir()
	rootCmd := NewRootCmd()
	orig := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.Flags().String("kb-poison", "bogus", "")
		var c int
		cmd.Flags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
			if name == "keyring-backend" {
				c++
				if c == 5 {
					return "kb-poison"
				}
			}
			return pflag.NormalizedName(name)
		})
		return orig(cmd, args)
	}
	rootCmd.SetArgs([]string{"init", "test-moniker", "--keyring-backend", "test", "--network-id", "mono_6940-1", "--home", home})
	err := rootCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown keyring backend")
}

func TestNewRootCmd_InitWithoutNetworkIDUsesDefault(t *testing.T) {
	homeDir := t.TempDir()
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"init", "test-moniker", "--chain-id", "test_99-1", "--home", homeDir})
	require.NoError(t, rootCmd.Execute())

	appToml, err := os.ReadFile(filepath.Join(homeDir, "config", "app.toml"))
	require.NoError(t, err)
	require.Contains(t, string(appToml), "evm-chain-id = 262144",
		"without --network-id, evm-chain-id must fall back to DefaultEVMChainID")
}
