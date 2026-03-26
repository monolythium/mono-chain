package types

import "cosmossdk.io/collections"

const (
	ModuleName = "burn"

	StoreKey = ModuleName

	// GovModuleName duplicates the gov module's name to avoid a dependency with x/gov.
	// It should be synced with the gov module's name if it is ever changed.
	// See: https://github.com/cosmos/cosmos-sdk/blob/v0.52.0-beta.2/x/gov/types/keys.go#L9
	GovModuleName = "gov"
)

var (
	ParamsKey              = collections.NewPrefix(0)
	GlobalBurnCountPrefix  = collections.NewPrefix(1)
	GlobalBurnTotalPrefix  = collections.NewPrefix(2)
	AccountBurnTotalPrefix = collections.NewPrefix(3)
)
