package types_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/x/mono/types"
)

func TestMsgRegisterValidator_UnpackInterfaces(t *testing.T) {
	pubKey := ed25519.GenPrivKey().PubKey()
	anyPub, err := codectypes.NewAnyWithValue(pubKey)
	require.NoError(t, err)

	msg := &types.MsgRegisterValidator{
		CreateValidator: stakingtypes.MsgCreateValidator{
			Pubkey: anyPub,
		},
	}

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)

	err = msg.UnpackInterfaces(registry)
	require.NoError(t, err)

	cached := msg.CreateValidator.Pubkey.GetCachedValue()
	require.NotNil(t, cached)
	_, ok := cached.(cryptotypes.PubKey)
	require.True(t, ok, "cached value must be a cryptotypes.PubKey")
}

func TestMsgRegisterValidator_UnpackInterfaces_NilPubkey(t *testing.T) {
	msg := &types.MsgRegisterValidator{
		CreateValidator: stakingtypes.MsgCreateValidator{
			Pubkey: nil,
		},
	}

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)

	err := msg.UnpackInterfaces(registry)
	require.NoError(t, err, "nil pubkey must not cause error or panic")
}

func TestMsgRegisterValidator_TypeURL(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	typeURL := sdk.MsgTypeURL(&types.MsgRegisterValidator{})
	resolved, err := registry.Resolve(typeURL)

	require.NoError(t, err)
	require.NotNil(t, resolved, "MsgRegisterValidator must be resolvable from its type URL")
}
