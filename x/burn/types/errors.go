package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrInvalidBurnAmount     = errors.Register(ModuleName, 2, "invalid burn amount")
	ErrInvalidBurnDenom      = errors.Register(ModuleName, 3, "invalid burn denomination")
	ErrInvalidBurnTotal      = errors.Register(ModuleName, 4, "burn total cannot be nil or negative")
	ErrInvalidFeeBurnPercent = errors.Register(ModuleName, 5, "invalid fee burn percent")
)
