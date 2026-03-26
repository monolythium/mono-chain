package keeper

import (
	"context"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/monolythium/mono-chain/x/validator/types"
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

func (ms msgServer) RegisterValidator(
	ctx context.Context,
	msg *types.MsgRegisterValidator,
) (*types.MsgRegisterValidatorResponse, error) {
	if err := ms.Keeper.RegisterValidator(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgRegisterValidatorResponse{}, nil
}

// UpdateParams updates the module parameters. Governance-gated.
func (ms msgServer) UpdateParams(
	ctx context.Context,
	msg *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
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
