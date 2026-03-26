package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// BurnKeeper defines the expected interface for the x/burn module.
type BurnKeeper interface {
	BurnFromAccount(ctx context.Context, senderAddr sdk.AccAddress, amount sdk.Coin) error
}

// StakingMsgServer defines the expected interface for the Staking MsgServer.
type StakingMsgServer interface {
	CreateValidator(context.Context, *stakingtypes.MsgCreateValidator) (*stakingtypes.MsgCreateValidatorResponse, error)
}
