package app

import sdk "github.com/cosmos/cosmos-sdk/types"

func init() {
	cfg := sdk.GetConfig()
	SetBech32Prefixes(cfg)
	SetBip44CoinType(cfg)
}
