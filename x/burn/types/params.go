package types

import (
	"cosmossdk.io/math"
)

var DefaultFeeBurnPercent = math.LegacyZeroDec()

func NewParams(feeBurnPercent math.LegacyDec) Params {
	return Params{
		FeeBurnPercent: feeBurnPercent,
	}
}

func DefaultParams() Params {
	return NewParams(DefaultFeeBurnPercent)
}

func (p Params) Validate() error {
	return validateFeeBurnPercent(p.FeeBurnPercent)
}

func validateFeeBurnPercent(v math.LegacyDec) error {
	if v.IsNil() ||
		v.IsNegative() ||
		v.GT(math.LegacyOneDec()) {
		return ErrInvalidFeeBurnPercent
	}

	return nil
}
