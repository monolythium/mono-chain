package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/monolythium/mono-chain/x/burn/types"
)

func (k msgServer) Burn(ctx context.Context, msg *types.MsgBurn) (*types.MsgBurnResponse, error) {
	fromAddr, err := k.addressCodec.StringToBytes(msg.FromAddress)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid from address")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	coins := sdk.NewCoins(msg.Amount)

	// Validate the balance
	// bank.SendCoinsFromAccountToModule checks this internally,
	// but we verify explicitly in case the SDK behavior changes
	spendableCoins := k.bankKeeper.SpendableCoins(sdkCtx, fromAddr)
	if !spendableCoins.IsAllGTE(coins) {
		return nil, errorsmod.Wrapf(
			sdkerrors.ErrInsufficientFunds,
			"spendable balance %s is smaller than %s",
			spendableCoins, coins,
		)
	}

	// Sanity check
	// Balance should never exceed total supply
	userBalance := k.bankKeeper.GetBalance(sdkCtx, fromAddr, msg.Amount.Denom)
	totalSupply := k.bankKeeper.GetSupply(sdkCtx, msg.Amount.Denom)
	if userBalance.Amount.GT(totalSupply.Amount) {
		panic(errorsmod.Wrapf(
			types.ErrStateCorruption,
			"user balance %s exceeds total supply %s",
			userBalance, totalSupply,
		))
	}

	// Verify supply won't underflow after burn
	// cosmos-sdk checks this too, but we verify explicitly
	if totalSupply.Amount.LT(msg.Amount.Amount) {
		return nil, errorsmod.Wrapf(
			types.ErrSupplyUnderflow,
			"cannot burn %s: total supply is only %s",
			msg.Amount, totalSupply,
		)
	}

	// Transfer to module account
	err = k.bankKeeper.SendCoinsFromAccountToModule(
		sdkCtx,
		fromAddr,
		types.ModuleName,
		coins,
	)
	if err != nil {
		return nil, err
	}

	// Burn from module account
	err = k.bankKeeper.BurnCoins(sdkCtx, types.ModuleName, coins)
	if err != nil {
		panic(errorsmod.Wrapf(types.ErrBurnFailed, "failed to burn after transfer: %v", err))
	}

	// Post-burn verification
	// Ensure state changes were applied correctly
	newUserBalance := k.bankKeeper.GetBalance(sdkCtx, fromAddr, msg.Amount.Denom)
	expectedBalance := userBalance.Sub(msg.Amount)
	if !newUserBalance.Equal(expectedBalance) {
		panic(errorsmod.Wrapf(
			types.ErrPostBurnValidation,
			"balance mismatch - expected %s, got %s",
			expectedBalance, newUserBalance,
		))
	}

	newSupply := k.bankKeeper.GetSupply(sdkCtx, msg.Amount.Denom)
	expectedSupply := totalSupply.Sub(msg.Amount)
	if !newSupply.Equal(expectedSupply) {
		panic(errorsmod.Wrapf(
			types.ErrPostBurnValidation,
			"supply mismatch - expected %s, got %s",
			expectedSupply, newSupply,
		))
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBurn,
			sdk.NewAttribute(types.AttributeKeyBurner, msg.FromAddress),
			sdk.NewAttribute(types.AttributeKeyAmount, coins.String()),
		),
	)

	return &types.MsgBurnResponse{}, nil
}
