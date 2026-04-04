package cmd

import (
	"math/big"
	"reflect"
	"strings"
	"testing"
	"text/template"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	cosmosevmserverconfig "github.com/cosmos/evm/server/config"

	"github.com/monolythium/mono-chain/app"
)

func TestMonoAppConfig_EmbedHasSquashTag(t *testing.T) {
	field, ok := reflect.TypeOf(MonoAppConfig{}).FieldByName("Config")
	require.True(t, ok, "MonoAppConfig must embed serverconfig.Config")
	tag := field.Tag.Get("mapstructure")
	require.Equal(t, ",squash", tag,
		"embedded Config must have mapstructure:\",squash\" for correct TOML serialization")
}

func TestMonoAppConfig_EVMFieldTags(t *testing.T) {
	tests := []struct {
		field string
		tag   string
	}{
		{"EVM", "evm"},
		{"JSONRPC", "json-rpc"},
		{"TLS", "tls"},
	}
	rt := reflect.TypeOf(MonoAppConfig{})
	for _, tc := range tests {
		f, ok := rt.FieldByName(tc.field)
		require.True(t, ok, "MonoAppConfig must have field %s", tc.field)
		require.Equal(t, tc.tag, f.Tag.Get("mapstructure"),
			"field %s mapstructure tag mismatch", tc.field)
	}
}

func TestMonoAppTemplate_ExactComposition(t *testing.T) {
	expected := serverconfig.DefaultConfigTemplate + cosmosevmserverconfig.DefaultEVMConfigTemplate
	require.Equal(t, expected, MonoAppTemplate)
}

func TestInitAppConfig(t *testing.T) {
	tmpl, cfg := InitAppConfig(6940)
	appCfg, ok := cfg.(MonoAppConfig)
	require.True(t, ok, "must return MonoAppConfig")

	t.Run("template has required sections", func(t *testing.T) {
		for _, section := range []string{"[evm]", "[json-rpc]", "[tls]"} {
			require.Contains(t, tmpl, section)
		}
	})

	t.Run("template has evm-chain-id directive", func(t *testing.T) {
		require.Contains(t, tmpl, "{{ .EVM.EVMChainID }}")
	})

	t.Run("template renders with struct", func(t *testing.T) {
		parsed, err := template.New("app").Parse(tmpl)
		require.NoError(t, err)
		var buf strings.Builder
		require.NoError(t, parsed.Execute(&buf, cfg))
		require.Contains(t, buf.String(), "evm-chain-id = 6940")
		require.Contains(t, buf.String(), "0"+app.DefaultBondDenom)
	})

	t.Run("MinGasPrices", func(t *testing.T) {
		require.Equal(t, "0"+app.DefaultBondDenom, appCfg.Config.MinGasPrices)
	})

	t.Run("EVM defaults", func(t *testing.T) {
		defaults := cosmosevmserverconfig.DefaultEVMConfig()
		require.Equal(t, uint64(6940), appCfg.EVM.EVMChainID)
		require.Equal(t, defaults.Tracer, appCfg.EVM.Tracer)
		require.Equal(t, defaults.MaxTxGasWanted, appCfg.EVM.MaxTxGasWanted)
		require.Equal(t, defaults.EnablePreimageRecording, appCfg.EVM.EnablePreimageRecording)
		require.Equal(t, defaults.MinTip, appCfg.EVM.MinTip)
		require.Equal(t, defaults.GethMetricsAddress, appCfg.EVM.GethMetricsAddress)
		require.Equal(t, defaults.Mempool, appCfg.EVM.Mempool)
	})

	t.Run("JSONRPC defaults", func(t *testing.T) {
		defaults := cosmosevmserverconfig.DefaultJSONRPCConfig()
		require.Equal(t, defaults.Address, appCfg.JSONRPC.Address)
		require.Equal(t, defaults.WsAddress, appCfg.JSONRPC.WsAddress)
		require.Equal(t, defaults.GasCap, appCfg.JSONRPC.GasCap)
		require.Equal(t, defaults.EVMTimeout, appCfg.JSONRPC.EVMTimeout)
		require.Equal(t, defaults.FilterCap, appCfg.JSONRPC.FilterCap)
		require.Equal(t, defaults.FeeHistoryCap, appCfg.JSONRPC.FeeHistoryCap)
		require.Equal(t, defaults.BatchRequestLimit, appCfg.JSONRPC.BatchRequestLimit)
		require.Equal(t, defaults.BatchResponseMaxSize, appCfg.JSONRPC.BatchResponseMaxSize)
	})

	t.Run("TLS defaults", func(t *testing.T) {
		defaults := cosmosevmserverconfig.DefaultTLSConfig()
		require.Equal(t, defaults.CertificatePath, appCfg.TLS.CertificatePath)
		require.Equal(t, defaults.KeyPath, appCfg.TLS.KeyPath)
	})

	t.Run("SDK server defaults", func(t *testing.T) {
		defaults := serverconfig.DefaultConfig()
		require.Equal(t, defaults.API.Enable, appCfg.Config.API.Enable)
		require.Equal(t, defaults.GRPC.Enable, appCfg.Config.GRPC.Enable)
		require.Equal(t, defaults.GRPCWeb.Enable, appCfg.Config.GRPCWeb.Enable)
		require.Equal(t, defaults.Pruning, appCfg.Config.Pruning)
	})
}

func TestInitAppConfig_EVMChainIDPropagated(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  uint64
		expect uint64
	}{
		{"testnet", 6940, 6940},
		{"mainnet", 6941, 6941},
		{"library default", cosmosevmserverconfig.DefaultEVMChainID, cosmosevmserverconfig.DefaultEVMChainID},
		{"custom", 999999, 999999},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, cfg := InitAppConfig(tc.input)
			require.Equal(t, tc.expect, cfg.(MonoAppConfig).EVM.EVMChainID)
		})
	}
}

func TestDefaultEVMChainID_MatchesLibrary(t *testing.T) {
	require.Equal(t, uint64(cosmosevmserverconfig.DefaultEVMChainID), uint64(DefaultEVMChainID),
		"DefaultEVMChainID must match cosmosevmserverconfig.DefaultEVMChainID")
}

func TestResolveEVMChainID_OverflowError(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String(FlagNetworkID, "", "")
	overflowID := new(big.Int).Add(new(big.Int).SetUint64(^uint64(0)), big.NewInt(1))
	_ = fs.Set(FlagNetworkID, "mono_"+overflowID.String()+"-1")

	_, err := resolveEVMChainID(fs)
	require.ErrorIs(t, err, ErrChainIDOverflow)
}

func TestResolveEVMChainID(t *testing.T) {
	const noFlag = "\x00" // sentinel: don't register the flag at all
	tests := []struct {
		name      string
		networkID string // value to set; "" = registered but empty; noFlag = not registered
		expect    uint64
		wantErr   bool
	}{
		{"flag absent returns default", noFlag, DefaultEVMChainID, false},
		{"flag empty returns default", "", DefaultEVMChainID, false},
		{"valid testnet ID", "mono_6940-1", 6940, false},
		{"valid mainnet ID", "mono_6941-1", 6941, false},
		{"uint64 max is valid", "mono_18446744073709551615-1", ^uint64(0), false},
		{"invalid format", "bad-chain-id", 0, true},
		{"leading zero rejected", "mono_0123-1", 0, true},
		{"uppercase rejected", "MONO_6940-1", 0, true},
		{"overflow beyond uint64", "mono_18446744073709551616-1", 0, true},
		{"special chars in name", "mono-chain_6940-1", 0, true},
		{"hex chain ID rejected", "mono_0xFF-1", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			if tc.networkID != noFlag {
				fs.String(FlagNetworkID, "", "")
				if tc.networkID != "" {
					_ = fs.Set(FlagNetworkID, tc.networkID)
				}
			}
			got, err := resolveEVMChainID(fs)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expect, got)
		})
	}
}
