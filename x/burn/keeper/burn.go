package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/monolythium/mono-chain/x/burn/types"
)

func (k Keeper) BurnFromAccount(ctx context.Context, sender sdk.AccAddress, nativeCoin sdk.Coin) error {
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		sender,
		types.ModuleName,
		sdk.NewCoins(nativeCoin),
	); err != nil {
		return err
	}

	return k.burnNative(ctx, sender, nativeCoin)
}

func (k Keeper) BurnFromModule(ctx context.Context, senderModule string, nativeCoin sdk.Coin) error {
	if err := k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		senderModule,
		types.ModuleName,
		sdk.NewCoins(nativeCoin),
	); err != nil {
		return err
	}

	return k.burnNative(ctx, authtypes.NewModuleAddress(senderModule), nativeCoin)
}

func (k Keeper) burnNative(ctx context.Context, sender sdk.AccAddress, nativeCoin sdk.Coin) error {
	if !nativeCoin.IsPositive() {
		return nil // noop - nothing to burn
	}

	if nativeCoin.Denom != sdk.DefaultBondDenom {
		return types.ErrInvalidBurnDenom
	}

	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(nativeCoin)); err != nil {
		return err
	}

	return k.updateTrackers(ctx, sender, nativeCoin)
}

func (k Keeper) updateTrackers(ctx context.Context, sender sdk.AccAddress, nativeCoin sdk.Coin) error {
	if _, err := k.GlobalBurnCount.Next(ctx); err != nil {
		return err
	}

	globalTotal, err := k.GetGlobalBurnTotal(ctx)
	if err != nil {
		return err
	}
	nextGlobalTotal := globalTotal.Add(nativeCoin.Amount)
	if err := k.GlobalBurnTotal.Set(ctx, nextGlobalTotal); err != nil {
		return err
	}

	accountTotal, err := k.GetAccountBurnTotal(ctx, sender)
	if err != nil {
		return err
	}
	nextAccountTotal := accountTotal.Add(nativeCoin.Amount)
	return k.AccountBurnTotal.Set(ctx, sender, nextAccountTotal)
}
