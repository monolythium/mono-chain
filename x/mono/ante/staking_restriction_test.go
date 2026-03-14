package ante_test

import (
	"context"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	monoante "github.com/monolythium/mono-chain/x/mono/ante"
	"github.com/monolythium/mono-chain/x/mono/types"
)

type mockInnerStakingServer struct {
	stakingtypes.UnimplementedMsgServer
	createValidatorCalled bool
	delegateCalled        bool
}

func (m *mockInnerStakingServer) CreateValidator(
	_ context.Context,
	_ *stakingtypes.MsgCreateValidator,
) (*stakingtypes.MsgCreateValidatorResponse, error) {
	m.createValidatorCalled = true
	return &stakingtypes.MsgCreateValidatorResponse{}, nil
}

func (m *mockInnerStakingServer) Delegate(
	_ context.Context,
	_ *stakingtypes.MsgDelegate,
) (*stakingtypes.MsgDelegateResponse, error) {
	m.delegateCalled = true
	return &stakingtypes.MsgDelegateResponse{}, nil
}

func newTestCtx(t *testing.T, height int64) sdk.Context {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey("test")
	return testutil.DefaultContextWithDB(
		t,
		storeKey,
		storetypes.NewTransientStoreKey("t"),
	).Ctx.WithBlockHeight(height)
}

func TestRestrictedStakingMsgServer(t *testing.T) {
	t.Run("CreateValidator blocked at height > 0", func(t *testing.T) {
		inner := &mockInnerStakingServer{}
		restricted := monoante.NewRestrictedStakingMsgServer(inner)
		ctx := newTestCtx(t, 10)

		_, err := restricted.CreateValidator(ctx, &stakingtypes.MsgCreateValidator{})

		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrInvalidRegistrationTx)
		require.False(t, inner.createValidatorCalled, "inner server must not be called")
	})

	t.Run("CreateValidator allowed at height 0 (genesis)", func(t *testing.T) {
		inner := &mockInnerStakingServer{}
		restricted := monoante.NewRestrictedStakingMsgServer(inner)
		ctx := newTestCtx(t, 0)

		_, err := restricted.CreateValidator(ctx, &stakingtypes.MsgCreateValidator{})

		require.NoError(t, err)
		require.True(t, inner.createValidatorCalled, "inner server must be called at genesis")
	})

	t.Run("other staking ops pass through at any height", func(t *testing.T) {
		inner := &mockInnerStakingServer{}
		restricted := monoante.NewRestrictedStakingMsgServer(inner)
		ctx := newTestCtx(t, 100)

		_, err := restricted.Delegate(ctx, &stakingtypes.MsgDelegate{})

		require.NoError(t, err)
		require.True(t, inner.delegateCalled, "Delegate must delegate to inner via embedding")
	})

	t.Run("nil msg at height > 0 returns error without deref", func(t *testing.T) {
		inner := &mockInnerStakingServer{}
		restricted := monoante.NewRestrictedStakingMsgServer(inner)
		ctx := newTestCtx(t, 10)

		_, err := restricted.CreateValidator(ctx, nil)

		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrInvalidRegistrationTx)
	})

	t.Run("nil inner server panics at height 0", func(t *testing.T) {
		restricted := monoante.NewRestrictedStakingMsgServer(nil)
		ctx := newTestCtx(t, 0)

		require.Panics(t, func() {
			_, _ = restricted.CreateValidator(ctx, &stakingtypes.MsgCreateValidator{})
		}, "nil inner server must panic on passthrough at genesis height")
	})
}

func TestStakingCircuitBreaker(t *testing.T) {
	cb := monoante.NewStakingCircuitBreaker()
	createValURL := sdk.MsgTypeURL(&stakingtypes.MsgCreateValidator{})

	testCases := []struct {
		name        string
		height      int64
		typeURL     string
		wantAllowed bool
	}{
		{
			name:        "MsgCreateValidator blocked post-genesis",
			height:      10,
			typeURL:     createValURL,
			wantAllowed: false,
		},
		{
			name:        "MsgCreateValidator allowed at genesis",
			height:      0,
			typeURL:     createValURL,
			wantAllowed: true,
		},
		{
			name:        "non-CreateValidator always allowed",
			height:      10,
			typeURL:     sdk.MsgTypeURL(&stakingtypes.MsgDelegate{}),
			wantAllowed: true,
		},
		{
			name:        "MsgRegisterValidator passes through",
			height:      10,
			typeURL:     sdk.MsgTypeURL(&types.MsgRegisterValidator{}),
			wantAllowed: true,
		},
		{
			name:        "empty typeURL no false positive",
			height:      10,
			typeURL:     "",
			wantAllowed: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := newTestCtx(t, tc.height)

			allowed, err := cb.IsAllowed(ctx, tc.typeURL)

			require.NoError(t, err)
			require.Equal(t, tc.wantAllowed, allowed)
		})
	}
}
