package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrInvalidRegistrationBurn  = errors.Register(ModuleName, 2, "invalid validator registration fee")
	ErrInvalidMinSelfDelegation = errors.Register(ModuleName, 3, "invalid validator min self delegation amount")
	ErrParamsRead               = errors.Register(ModuleName, 4, "failed to read module params")

	ErrInvalidRegistrationTx          = errors.Register(ModuleName, 5, "MsgCreateValidator is not permitted; use MsgRegisterValidator")
	ErrRegistrationAddressMismatch    = errors.Register(ModuleName, 6, "burn sender must match validator operator")
	ErrBurnDenomMismatch              = errors.Register(ModuleName, 7, "burn denom does not match required denom")
	ErrBurnBelowRequired              = errors.Register(ModuleName, 8, "registration burn below required minimum")
	ErrDelegationDenomMismatch        = errors.Register(ModuleName, 9, "delegation denom does not match required denom")
	ErrMinSelfDelegationBelowRequired = errors.Register(ModuleName, 10, "minimum self-delegation below chain requirement")
)
