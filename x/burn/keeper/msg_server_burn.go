package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/monolythium/mono-chain/x/burn/types"
)

func (k msgServer) Burn(ctx context.Context, msg *types.MsgBurn) (*types.MsgBurnResponse, error) {
	senderAddrBytes, err := k.addressCodec.StringToBytes(msg.FromAddress)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid from address")
	}
	senderAddr := sdk.AccAddress(senderAddrBytes)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	userBalance := k.bankKeeper.GetBalance(sdkCtx, senderAddr, msg.Amount.Denom)
	totalSupply := k.bankKeeper.GetSupply(sdkCtx, msg.Amount.Denom)
	spendableAmount := sdk.NewCoin(
		msg.Amount.Denom,
		k.bankKeeper.SpendableCoins(sdkCtx, senderAddr).AmountOf(msg.Amount.Denom),
	)

	if err = k.canBurn(spendableAmount, msg.Amount, userBalance, totalSupply); err != nil {
		return nil, err
	}

	if err := k.burnCoins(sdkCtx, senderAddr, sdk.NewCoins(msg.Amount)); err != nil {
		return nil, err
	}

	if err = k.validateBurn(sdkCtx, senderAddr, msg, userBalance, totalSupply); err != nil {
		return nil, err
	}

	if err = k.updateTrackers(sdkCtx, msg, senderAddr); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBurn,
			sdk.NewAttribute(types.AttributeKeyBurner, msg.FromAddress),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
		),
	)

	return &types.MsgBurnResponse{}, nil
}

func (k msgServer) canBurn(spendableAmount sdk.Coin, burnAmount sdk.Coin, userBalance sdk.Coin, totalSupply sdk.Coin) error {
	// Verify only the native denom can be burned
	if burnAmount.Denom != sdk.DefaultBondDenom {
		return errorsmod.Wrapf(
			types.ErrInvalidBurnDenom,
			"only native denom %s can be burned, got %s",
			sdk.DefaultBondDenom, burnAmount.Denom,
		)
	}

	// Validate the balance
	// bank.SendCoinsFromAccountToModule checks this internally,
	// but we verify explicitly in case the SDK behavior changes
	if !spendableAmount.IsGTE(burnAmount) {
		return errorsmod.Wrapf(
			types.ErrInsufficientFunds,
			"spendable balance %s is smaller than %s",
			spendableAmount, burnAmount,
		)
	}

	// Sanity check
	// Balance should never exceed total supply
	if userBalance.Amount.GT(totalSupply.Amount) {
		return errorsmod.Wrapf(
			types.ErrStateCorruption,
			"user balance %s exceeds total supply %s",
			userBalance, totalSupply,
		)
	}

	// Verify supply won't underflow after burn
	// cosmos-sdk checks this too, but we verify explicitly
	if totalSupply.IsLT(burnAmount) {
		return errorsmod.Wrapf(
			types.ErrSupplyUnderflow,
			"cannot burn %s: total supply is only %s",
			burnAmount, totalSupply,
		)
	}

	return nil
}

func (k msgServer) burnCoins(ctx context.Context, senderAddr sdk.AccAddress, burnAmount sdk.Coins) error {
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, burnAmount); err != nil {
		return err
	}

	// Burn from module account
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnAmount); err != nil {
		return errorsmod.Wrapf(types.ErrBurnFailed, "failed to burn after transfer: %v", err)
	}

	return nil
}

func (k msgServer) validateBurn(ctx context.Context, senderAddr sdk.AccAddress, msg *types.MsgBurn, userBalance sdk.Coin, totalSupply sdk.Coin) error {
	newUserBalance := k.bankKeeper.GetBalance(ctx, senderAddr, msg.Amount.Denom)
	expectedBalance := userBalance.Sub(msg.Amount)
	if !newUserBalance.Equal(expectedBalance) {
		return errorsmod.Wrapf(
			types.ErrPostBurnValidation,
			"balance mismatch - expected %s, got %s",
			expectedBalance, newUserBalance,
		)
	}

	newSupply := k.bankKeeper.GetSupply(ctx, msg.Amount.Denom)
	expectedSupply := totalSupply.Sub(msg.Amount)
	if !newSupply.Equal(expectedSupply) {
		return errorsmod.Wrapf(
			types.ErrPostBurnValidation,
			"supply mismatch - expected %s, got %s",
			expectedSupply, newSupply,
		)
	}

	return nil
}

func (k msgServer) updateTrackers(ctx context.Context, msg *types.MsgBurn, senderAddr sdk.AccAddress) error {
	if err := k.updateBurnCount(ctx); err != nil {
		return err
	}

	if err := k.updateGlobalBurnTotal(ctx, msg); err != nil {
		return err
	}

	if err := k.updateAccountBurnTotal(ctx, senderAddr, msg); err != nil {
		return err
	}

	return nil
}

func (k msgServer) updateBurnCount(ctx context.Context) error {
	if _, err := k.BurnCount.Next(ctx); err != nil {
		return errorsmod.Wrapf(types.ErrPostBurnValidation, "BurnCount increment error: %v", err)
	}

	return nil
}

func (k msgServer) updateGlobalBurnTotal(ctx context.Context, msg *types.MsgBurn) error {
	globalBurnTotal, err := k.BurnTotal.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			globalBurnTotal = sdk.NewCoin(msg.Amount.Denom, math.ZeroInt())
		} else {
			return errorsmod.Wrapf(types.ErrPostBurnValidation, "Burn total get error: %v", err)
		}
	}

	globalBurnTotal = globalBurnTotal.Add(msg.Amount)
	if err := k.BurnTotal.Set(ctx, globalBurnTotal); err != nil {
		return errorsmod.Wrapf(types.ErrPostBurnValidation, "Burn total set error: %v", err)
	}

	return nil
}

func (k msgServer) updateAccountBurnTotal(ctx context.Context, senderAddr sdk.AccAddress, msg *types.MsgBurn) error {
	accountBurnTotal, err := k.BurnAccountTotal.Get(ctx, senderAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			accountBurnTotal = sdk.NewCoin(msg.Amount.Denom, math.ZeroInt())
		} else {
			return errorsmod.Wrapf(types.ErrPostBurnValidation, "Account burn total get error: %v", err)
		}
	}

	accountBurnTotal = accountBurnTotal.Add(msg.Amount)
	if err := k.BurnAccountTotal.Set(ctx, senderAddr, accountBurnTotal); err != nil {
		return errorsmod.Wrapf(types.ErrPostBurnValidation, "Account burn total set error: %v", err)
	}

	return nil
}
