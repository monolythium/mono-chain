package ante

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/monolythium/mono-chain/x/mono/types"
)

// RestrictedStakingMsgServer overrides CreateValidator to reject it.
// This is injected into the staking precompile.
type RestrictedStakingMsgServer struct {
	stakingtypes.MsgServer
}

func NewRestrictedStakingMsgServer(inner stakingtypes.MsgServer) *RestrictedStakingMsgServer {
	return &RestrictedStakingMsgServer{MsgServer: inner}
}

func (r *RestrictedStakingMsgServer) CreateValidator(
	ctx context.Context,
	msg *stakingtypes.MsgCreateValidator,
) (*stakingtypes.MsgCreateValidatorResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() == 0 {
		return r.MsgServer.CreateValidator(ctx, msg)
	}

	return nil, types.ErrInvalidRegistrationTx
}

// StakingCircuitBreaker blocks MsgCreateValidator via the msg router (authz, governance).
// Height 0 allowed for genesis bootstrap (gentx).
type StakingCircuitBreaker struct{}

func NewStakingCircuitBreaker() StakingCircuitBreaker {
	return StakingCircuitBreaker{}
}

func (cb StakingCircuitBreaker) IsAllowed(ctx context.Context, typeURL string) (bool, error) {
	if typeURL == sdk.MsgTypeURL(&stakingtypes.MsgCreateValidator{}) {
		return sdk.UnwrapSDKContext(ctx).BlockHeight() == 0, nil
	}

	return true, nil
}
