package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

var _ codectypes.UnpackInterfacesMessage = (*MsgRegisterValidator)(nil)

// UnpackInterfaces explicitly unpacks nested MsgCreateValidator proto fields from wire.
func (msg *MsgRegisterValidator) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return msg.CreateValidator.UnpackInterfaces(unpacker)
}
