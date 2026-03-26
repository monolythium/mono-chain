package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/monolythium/mono-chain/x/burn/types"
)

// ProcessFeeBurn burns a designated percent of native block fees,
// leaving the remainder in fee_collector for x/distribution.
//
// MUST run in BeginBlock before x/distribution and x/mint.
func (k Keeper) ProcessFeeBurn(ctx context.Context) error {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	nativeCoin := k.bankKeeper.GetBalance(
		ctx,
		authtypes.NewModuleAddress(authtypes.FeeCollectorName),
		sdk.DefaultBondDenom,
	)

	// e.g., 0.01 * amt
	// the remainder stays in fee_collector for x/dist handling
	burnableAmount := params.FeeBurnPercent.MulInt(nativeCoin.Amount).TruncateInt()
	if !burnableAmount.IsPositive() {
		return nil
	}

	burnableCoin := sdk.NewCoin(sdk.DefaultBondDenom, burnableAmount)

	if err := k.BurnFromModule(ctx, authtypes.FeeCollectorName, burnableCoin); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(types.EventTypeFeeBurn,
			sdk.NewAttribute(types.AttributeKeyBurnPercent, params.FeeBurnPercent.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, burnableCoin.String()),
		),
	)

	return nil
}
