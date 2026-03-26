package burn

import (
	"context"

	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/monolythium/mono-chain/x/burn/types"
)

// BeginBlock contains the logic that is automatically triggered at the beginning of each block.
func (am AppModule) BeginBlock(ctx context.Context) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, telemetry.Now(), telemetry.MetricKeyBeginBlocker)
	return am.keeper.ProcessFeeBurn(ctx)
}
