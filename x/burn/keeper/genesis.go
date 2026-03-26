package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/monolythium/mono-chain/x/burn/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	if err := k.Params.Set(ctx, genState.Params); err != nil {
		return err
	}

	if err := k.GlobalBurnCount.Set(ctx, genState.Count); err != nil {
		return err
	}

	if !genState.Total.IsNil() {
		if err := k.GlobalBurnTotal.Set(ctx, genState.Total); err != nil {
			return err
		}
	}

	for _, record := range genState.AccountTotals {
		addr, err := k.addressCodec.StringToBytes(record.Address)
		if err != nil {
			return err
		}

		if err := k.AccountBurnTotal.Set(ctx, addr, record.Total); err != nil {
			return err
		}
	}

	return nil
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	globalBurnCount, err := k.GlobalBurnCount.Peek(ctx)
	if err != nil {
		return nil, err
	}

	globalBurnTotal, err := k.GetGlobalBurnTotal(ctx)
	if err != nil {
		return nil, err
	}

	var accountBurnTotals []types.AccountBurnRecord
	err = k.AccountBurnTotal.Walk(ctx, nil, func(addr sdk.AccAddress, amount math.Int) (bool, error) {
		addrStr, err := k.addressCodec.BytesToString(addr)
		if err != nil {
			return true, err
		}
		accountBurnTotals = append(accountBurnTotals, types.AccountBurnRecord{
			Address: addrStr,
			Total:   amount,
		})
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.GenesisState{
		Params:        params,
		Count:         globalBurnCount,
		Total:         globalBurnTotal,
		AccountTotals: accountBurnTotals,
	}, nil
}
