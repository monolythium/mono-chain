package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/mono module sentinel errors
var (
	ErrInvalidSigner            = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidFeeBurnPercent    = errors.Register(ModuleName, 1101, "invalid fee burn percent")
	ErrInvalidRegistrationBurn  = errors.Register(ModuleName, 1102, "invalid validator registration fee")
	ErrInvalidMinSelfDelegation = errors.Register(ModuleName, 1103, "invalid validator min self delegation amount")
	ErrFeeSplitFailed           = errors.Register(ModuleName, 1104, "fee split processing failed")
	ErrProposerNotFound         = errors.Register(ModuleName, 1105, "block proposer not found")
	ErrParamsRead               = errors.Register(ModuleName, 1106, "failed to read module params")

	ErrInvalidRegistrationTx          = errors.Register(ModuleName, 1200, "MsgCreateValidator is not permitted; use MsgRegisterValidator")
	ErrRegistrationAddressMismatch    = errors.Register(ModuleName, 1201, "burn sender must match validator operator")
	ErrBurnDenomMismatch              = errors.Register(ModuleName, 1202, "burn denom does not match required denom")
	ErrBurnBelowRequired              = errors.Register(ModuleName, 1203, "registration burn below required minimum")
	ErrDelegationDenomMismatch        = errors.Register(ModuleName, 1204, "delegation denom does not match required denom")
	ErrMinSelfDelegationBelowRequired = errors.Register(ModuleName, 1205, "minimum self-delegation below chain requirement")
)
