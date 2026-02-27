package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/burn module sentinel errors
var (
	ErrInvalidSigner      = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrStateCorruption    = errors.Register(ModuleName, 1101, "state corruption detected")
	ErrSupplyUnderflow    = errors.Register(ModuleName, 1102, "burn amount exceeds total supply")
	ErrPostBurnValidation = errors.Register(ModuleName, 1103, "post-burn validation failed")
	ErrBurnFailed         = errors.Register(ModuleName, 1104, "burn operation failed")
	ErrInvalidBurnDenom   = errors.Register(ModuleName, 1105, "invalid denomination for burn")
	ErrInsufficientFunds  = errors.Register(ModuleName, 1106, "insufficient funds for burn")
)
