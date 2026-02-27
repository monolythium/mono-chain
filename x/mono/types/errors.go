package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/mono module sentinel errors
var (
	ErrInvalidSigner          = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidFeeBurnPercent  = errors.Register(ModuleName, 1101, "invalid fee burn percent")
	ErrInvalidRegistrationFee = errors.Register(ModuleName, 1102, "invalid validator registration fee")
	ErrFeeSplitFailed         = errors.Register(ModuleName, 1103, "fee split processing failed")
	ErrProposerNotFound       = errors.Register(ModuleName, 1104, "block proposer not found")
	ErrParamsRead             = errors.Register(ModuleName, 1105, "failed to read module params")

	ErrMissingBurnInfo           = errors.Register(ModuleName, 1200, "validator registration requires burning funds")
	ErrBurnSenderMismatch        = errors.Register(ModuleName, 1201, "burn deposit sender must match validator operator")
	ErrInsufficientBurnAmount    = errors.Register(ModuleName, 1202, "burn deposit amount below required registration fee")
	ErrBurnDenomMismatch         = errors.Register(ModuleName, 1203, "burn deposit denom does not match required denom")
	ErrInvalidValidatorAddress   = errors.Register(ModuleName, 1204, "invalid validator address")
	ErrInvalidBurnAddress        = errors.Register(ModuleName, 1205, "invalid burn address in transaction")
	ErrDuplicateRegistrationInfo         = errors.Register(ModuleName, 1206, "validator registration limited to one per transaction")
	ErrInsufficientMinSelfDelegation     = errors.Register(ModuleName, 1207, "minimum self-delegation below required amount")
	ErrAlreadyValidator                  = errors.Register(ModuleName, 1208, "account is already a registered validator")
)
