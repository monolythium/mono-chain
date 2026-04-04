package cmd

import (
	"cosmossdk.io/errors"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/spf13/pflag"

	cosmosevmserverconfig "github.com/cosmos/evm/server/config"

	"github.com/monolythium/mono-chain/app"
)

const ModuleName = "monod"

const (
	// FlagNetworkID is the init flag for specifying the network identifier.
	//
	// Use: `monod init {moniker} --network-id {name}_{evm-chain-id}-{version}`
	// - testnet: "mono_6940-1"
	// - mainnet: "mono_6941-1"
	FlagNetworkID    = "network-id"
	FlagNetworkIDUse = "Convenience flag to set both chain-id and evm-chain-id (e.g., mono_6940-1|mono_6941-1)"
)

const (
	DefaultEVMChainID = cosmosevmserverconfig.DefaultEVMChainID
	MonoAppTemplate   = serverconfig.DefaultConfigTemplate + cosmosevmserverconfig.DefaultEVMConfigTemplate
)

var (
	ErrInvalidFlagsCombo = errors.Register(ModuleName, 2, "--network-id cannot be combined with --chain-id")
	ErrChainIDOverflow   = errors.Register(ModuleName, 3, "chain-id must not overflow uint64")
)

type MonoAppConfig struct {
	serverconfig.Config `mapstructure:",squash"`
	EVM                 cosmosevmserverconfig.EVMConfig     `mapstructure:"evm"`
	JSONRPC             cosmosevmserverconfig.JSONRPCConfig `mapstructure:"json-rpc"`
	TLS                 cosmosevmserverconfig.TLSConfig     `mapstructure:"tls"`
}

// InitAppConfig returns the TOML template and config struct for app.toml.
// InterceptConfigsPreRunHandler serializes it; no manual file I/O needed.
func InitAppConfig(evmChainID uint64) (string, interface{}) {
	srvCfg := serverconfig.DefaultConfig()
	srvCfg.MinGasPrices = "0" + app.DefaultBondDenom

	evmCfg := cosmosevmserverconfig.DefaultEVMConfig()
	evmCfg.EVMChainID = evmChainID

	customAppConfig := MonoAppConfig{
		Config:  *srvCfg,
		EVM:     *evmCfg,
		JSONRPC: *cosmosevmserverconfig.DefaultJSONRPCConfig(),
		TLS:     *cosmosevmserverconfig.DefaultTLSConfig(),
	}

	return MonoAppTemplate, customAppConfig
}

func resolveEVMChainID(fs *pflag.FlagSet) (uint64, error) {
	networkID, err := fs.GetString(FlagNetworkID)
	if err != nil || networkID == "" {
		return DefaultEVMChainID, nil
	}

	evmChainID, err := ParseChainID(networkID)
	if err != nil {
		return 0, err
	}

	// EIP-155 spec is uint256; EVMOS helper is *big.Int; Cosmos uses uint64
	if !evmChainID.IsUint64() {
		return 0, ErrChainIDOverflow
	}

	return evmChainID.Uint64(), nil
}
