package types_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/x/validator/types"
)

func TestMsgRegisterValidator_UnpackInterfaces(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)

	pubKey := ed25519.GenPrivKey().PubKey()
	anyPubKey, err := codectypes.NewAnyWithValue(pubKey)
	require.NoError(t, err)

	msg := &types.MsgRegisterValidator{
		CreateValidator: stakingtypes.MsgCreateValidator{
			Pubkey: anyPubKey,
		},
	}

	err = msg.UnpackInterfaces(registry)
	require.NoError(t, err)
}

func TestMsgRegisterValidator_UnpackInterfaces_Unresolvable(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()

	msg := &types.MsgRegisterValidator{
		CreateValidator: stakingtypes.MsgCreateValidator{
			Pubkey: &codectypes.Any{TypeUrl: "/invalid.Type"},
		},
	}

	err := msg.UnpackInterfaces(registry)
	require.Error(t, err)
}
