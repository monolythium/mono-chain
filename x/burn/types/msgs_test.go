package types_test

import (
	"os"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/app"
	"github.com/monolythium/mono-chain/x/burn/types"
)

// TestMain sets sdk.DefaultBondDenom = "alyth" via app init().
func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	app.SetBech32Prefixes(cfg)
	app.SetBip44CoinType(cfg)
	os.Exit(m.Run())
}

func TestNewMsgBurn(t *testing.T) {
	coin := sdk.NewCoin("alyth", math.NewInt(100))
	msg := types.NewMsgBurn("mono1sender", coin)
	require.Equal(t, "mono1sender", msg.FromAddress)
	require.Equal(t, coin, msg.Amount)
}
