//go:build sims

package app

import (
	"io"
	"testing"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/cosmos/cosmos-sdk/baseapp"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sims "github.com/cosmos/cosmos-sdk/testutil/simsx"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
)

// Profile with:
// `go test -tags sims -benchmem -run=^$ ./app -bench ^BenchmarkFullAppSimulation$ -Commit=true -cpuprofile cpu.out`
func BenchmarkFullAppSimulation(b *testing.B) {
	b.ReportAllocs()

	cfg := simcli.NewConfigFromFlags()
	cfg.ChainID = SimAppChainID

	appFactory := func(logger log.Logger, db dbm.DB, traceStore io.Writer, loadLatest bool, appOpts servertypes.AppOptions, baseAppOptions ...func(*baseapp.BaseApp)) *App {
		return simNew(logger, db, nil, true, appOpts, append(baseAppOptions, interBlockCacheOpt())...)
	}

	sims.RunWithSeed(b, cfg, appFactory, setupStateFactory, cfg.Seed, nil)
}
