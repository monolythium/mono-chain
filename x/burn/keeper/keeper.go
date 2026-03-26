package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/monolythium/mono-chain/x/burn/types"
)

const (
	GlobalBurnCountName  = "global_burn_count"
	GlobalBurnTotalName  = "global_burn_total"
	AccountBurnTotalName = "account_burn_total"
)

type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec

	authority string

	Schema collections.Schema
	Params collections.Item[types.Params]

	bankKeeper types.BankKeeper

	// Burn tracking
	GlobalBurnCount  collections.Sequence
	GlobalBurnTotal  collections.Item[math.Int]
	AccountBurnTotal collections.Map[sdk.AccAddress, math.Int]
}

func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,

	authority string,

	bankKeeper types.BankKeeper,
) Keeper {
	if _, err := addressCodec.StringToBytes(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", err))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,

		authority: authority,

		bankKeeper: bankKeeper,
		Params:     collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),

		GlobalBurnCount:  collections.NewSequence(sb, types.GlobalBurnCountPrefix, GlobalBurnCountName),
		GlobalBurnTotal:  collections.NewItem(sb, types.GlobalBurnTotalPrefix, GlobalBurnTotalName, sdk.IntValue),
		AccountBurnTotal: collections.NewMap(sb, types.AccountBurnTotalPrefix, AccountBurnTotalName, sdk.AccAddressKey, sdk.IntValue),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) GetGlobalBurnTotal(ctx context.Context) (math.Int, error) {
	globalTotal, err := k.GlobalBurnTotal.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return math.ZeroInt(), nil
		}
		return globalTotal, err
	}

	return globalTotal, nil
}

func (k Keeper) GetAccountBurnTotal(ctx context.Context, address sdk.AccAddress) (math.Int, error) {
	accountTotal, err := k.AccountBurnTotal.Get(ctx, address)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return math.ZeroInt(), nil
		}
		return accountTotal, err
	}

	return accountTotal, nil
}
