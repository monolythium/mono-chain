package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/monolythium/mono-chain/x/burn/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// Burn is the thin MsgServer translator. Pattern: x/bank's msgServer.Send.
// Decodes the message, delegates to Keeper.BurnFromAccount(). Emits nothing.
func (ms msgServer) Burn(ctx context.Context, msg *types.MsgBurn) (*types.MsgBurnResponse, error) {
	senderBytes, err := ms.addressCodec.StringToBytes(msg.FromAddress)
	if err != nil {
		return nil, err
	}

	if err := ms.Keeper.BurnFromAccount(ctx, sdk.AccAddress(senderBytes), msg.Amount); err != nil {
		return nil, err
	}

	return &types.MsgBurnResponse{}, nil
}

// UpdateParams updates the module parameters. Governance-gated.
func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.authority != msg.GetAuthority() {
		return nil, govtypes.ErrInvalidSigner
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	if err := ms.Params.Set(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
