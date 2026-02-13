package encoding

import (
	"github.com/cosmos/gogoproto/proto"

	"cosmossdk.io/x/tx/signing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	enccodec "github.com/monolythium/mono-chain/encoding/codec"
)

// Config specifies the concrete encoding types to use for a given app.
// This is provided for compatibility between protobuf and amino implementations.
type Config struct {
	InterfaceRegistry types.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

// MakeConfig creates an encoding Config for the application.
func MakeConfig() Config {
	cdc := codec.NewLegacyAmino()
	signingOptions := signing.Options{
		AddressCodec:          address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: address.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
	}

	interfaceRegistry, _ := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles:     proto.HybridResolver,
		SigningOptions: signingOptions,
	})
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	enccodec.RegisterLegacyAminoCodec(cdc)
	enccodec.RegisterInterfaces(interfaceRegistry)

	return Config{
		InterfaceRegistry: interfaceRegistry,
		Codec:             appCodec,
		TxConfig:          tx.NewTxConfig(appCodec, tx.DefaultSignModes),
		Amino:             cdc,
	}
}
