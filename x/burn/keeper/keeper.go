package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/monolythium/mono-chain/x/burn/types"
)

var (
	// BurnCount tracks the total number of burns globally
	BurnCountName   = "burn_count"
	BurnCountPrefix = collections.NewPrefix(0)

	// BurnTotal tracks the total amount of funds burned globally
	BurnTotalName   = "burn_total"
	BurnTotalPrefix = collections.NewPrefix(1)

	// BurnAccountTotal tracks the total amount of funds burned by an account
	BurnAccountTotalName   = "burn_account_total"
	BurnAccountTotalPrefix = collections.NewPrefix(2)
)

type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	authority    []byte

	Schema collections.Schema
	Params collections.Item[types.Params]

	bankKeeper types.BankKeeper

	// Burn tracking
	BurnCount        collections.Sequence
	BurnTotal        collections.Item[sdk.Coin]
	BurnAccountTotal collections.Map[sdk.AccAddress, sdk.Coin]
}

func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,

	bankKeeper types.BankKeeper,
) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,

		bankKeeper: bankKeeper,
		Params:     collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),

		BurnCount: collections.NewSequence(
			sb,
			BurnCountPrefix,
			BurnCountName,
		),
		BurnTotal: collections.NewItem(
			sb,
			BurnTotalPrefix,
			BurnTotalName,
			codec.CollValue[sdk.Coin](cdc),
		),
		BurnAccountTotal: collections.NewMap(
			sb,
			BurnAccountTotalPrefix,
			BurnAccountTotalName,
			sdk.AccAddressKey,
			codec.CollValue[sdk.Coin](cdc),
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}
