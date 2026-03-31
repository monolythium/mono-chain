package cmd

import (
	"strconv"
	"strings"

	cmtcfg "github.com/cometbft/cometbft/config"
	evmconfig "github.com/cosmos/evm/config"
)

const defaultMinGasPrices = "10000000000alyth"

// initCometBFTConfig helps to override default CometBFT Config values.
// return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

// initAppConfig returns the EVM-extended app config template and default values.
// This adds JSON-RPC, TLS, and EVM config sections to app.toml.
//
// If chainID is provided in Cosmos format (e.g. "mono_6940-1"), the EVM chain
// ID is derived automatically. Otherwise falls back to the library default.
func initAppConfig(chainID string) (string, interface{}) {
	var evmChainID uint64 = evmconfig.EVMChainID
	if id, ok := parseEVMChainID(chainID); ok {
		evmChainID = id
	}

	tmpl, cfg := evmconfig.InitAppConfig("alyth", evmChainID)

	// Override the library default ("0alyth") with a production-safe minimum.
	// Validators can still override this in their own app.toml.
	appCfg, ok := cfg.(evmconfig.EVMAppConfig)
	if ok {
		appCfg.Config.MinGasPrices = defaultMinGasPrices
		return tmpl, appCfg
	}

	return tmpl, cfg
}

// parseEVMChainID extracts the EVM chain ID from a Cosmos chain ID.
// Format: "{name}_{evm-chain-id}-{version}" e.g. "mono_6940-1" → 6940.
func parseEVMChainID(chainID string) (uint64, bool) {
	// Split on underscore: ["mono", "6940-1"]
	parts := strings.SplitN(chainID, "_", 2)
	if len(parts) != 2 {
		return 0, false
	}
	// Split on dash: ["6940", "1"]
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
