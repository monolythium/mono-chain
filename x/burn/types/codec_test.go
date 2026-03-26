package types_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/monolythium/mono-chain/x/burn/types"
)

func TestRegisterInterfaces_MsgBurnResolvable(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	resolved, err := registry.Resolve(sdk.MsgTypeURL(&types.MsgBurn{}))
	require.NoError(t, err)
	require.NotNil(t, resolved)
}

func TestRegisterInterfaces_MsgUpdateParamsResolvable(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	resolved, err := registry.Resolve(sdk.MsgTypeURL(&types.MsgUpdateParams{}))
	require.NoError(t, err)
	require.NotNil(t, resolved)
}
