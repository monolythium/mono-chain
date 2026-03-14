package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	DefaultFeeBurnPercent             = math.LegacyZeroDec()
	DefaultValidatorRegistrationBurn  = sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())
	DefaultValidatorMinSelfDelegation = sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())
)

func NewParams(
	feeBurnPercent math.LegacyDec,
	validatorRegistrationBurn sdk.Coin,
	validatorMinSelfDelegation sdk.Coin,
) Params {
	return Params{
		FeeBurnPercent:             feeBurnPercent,
		ValidatorRegistrationBurn:  validatorRegistrationBurn,
		ValidatorMinSelfDelegation: validatorMinSelfDelegation,
	}
}

func DefaultParams() Params {
	return NewParams(
		DefaultFeeBurnPercent,
		DefaultValidatorRegistrationBurn,
		DefaultValidatorMinSelfDelegation,
	)
}

func (p Params) Validate() error {
	if err := validateFeeBurnPercent(p.FeeBurnPercent); err != nil {
		return err
	}

	if err := validateRegistrationBurn(p.ValidatorRegistrationBurn); err != nil {
		return err
	}

	return validateMinSelfDelegation(p.ValidatorMinSelfDelegation)
}

func validateFeeBurnPercent(v math.LegacyDec) error {
	if v.IsNil() {
		return ErrInvalidFeeBurnPercent
	}

	if v.IsNegative() {
		return ErrInvalidFeeBurnPercent
	}

	if v.GT(math.LegacyOneDec()) {
		return ErrInvalidFeeBurnPercent
	}

	return nil
}

func validateRegistrationBurn(v sdk.Coin) error {
	if err := v.Validate(); err != nil {
		return ErrInvalidRegistrationBurn
	}

	if !v.IsZero() && v.Denom != sdk.DefaultBondDenom {
		return ErrInvalidRegistrationBurn
	}

	return nil
}

func validateMinSelfDelegation(v sdk.Coin) error {
	if err := v.Validate(); err != nil {
		return ErrInvalidMinSelfDelegation
	}

	if !v.IsZero() && v.Denom != sdk.DefaultBondDenom {
		return ErrInvalidMinSelfDelegation
	}

	return nil
}
