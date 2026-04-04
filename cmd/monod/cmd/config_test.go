package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitCometBFTConfig_ReturnsDefault(t *testing.T) {
	cfg := initCometBFTConfig()
	require.NotNil(t, cfg)
	require.NotEmpty(t, cfg.RPC.ListenAddress)
	require.NotEmpty(t, cfg.P2P.ListenAddress)
	require.True(t, cfg.Consensus.CreateEmptyBlocks)
}
