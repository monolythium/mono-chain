package keeper_test

import (
	"context"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	burnmoduletypes "github.com/monolythium/mono-chain/x/burn/types"
)

// mockStakingMsgServer stubs stakingtypes.MsgServer for keeper unit tests.
// Embeds UnimplementedMsgServer so only CreateValidator needs explicit handling.
type mockStakingMsgServer struct {
	stakingtypes.UnimplementedMsgServer
	createValidatorFn func(
		context.Context,
		*stakingtypes.MsgCreateValidator,
	) (*stakingtypes.MsgCreateValidatorResponse, error)
	calls []string
}

func (m *mockStakingMsgServer) CreateValidator(
	ctx context.Context,
	msg *stakingtypes.MsgCreateValidator,
) (*stakingtypes.MsgCreateValidatorResponse, error) {
	m.calls = append(m.calls, "CreateValidator")
	if m.createValidatorFn != nil {
		return m.createValidatorFn(ctx, msg)
	}
	return &stakingtypes.MsgCreateValidatorResponse{}, nil
}

// mockBurnMsgServer stubs burnmoduletypes.MsgServer for keeper unit tests.
// Embeds UnimplementedMsgServer so only Burn needs explicit handling.
type mockBurnMsgServer struct {
	burnmoduletypes.UnimplementedMsgServer
	burnFn  func(context.Context, *burnmoduletypes.MsgBurn) (*burnmoduletypes.MsgBurnResponse, error)
	calls   []string
	lastMsg *burnmoduletypes.MsgBurn
}

func (m *mockBurnMsgServer) Burn(
	ctx context.Context,
	msg *burnmoduletypes.MsgBurn,
) (*burnmoduletypes.MsgBurnResponse, error) {
	m.calls = append(m.calls, "Burn")
	m.lastMsg = msg
	if m.burnFn != nil {
		return m.burnFn(ctx, msg)
	}
	return &burnmoduletypes.MsgBurnResponse{}, nil
}
