package types

import (
	"cosmossdk.io/core/address"
)

// DefaultGenesisState returns the default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate(addressCodec address.Codec) error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	if !gs.Total.IsNil() && gs.Total.IsNegative() {
		return ErrInvalidBurnTotal
	}

	for _, record := range gs.AccountTotals {
		if _, err := addressCodec.StringToBytes(record.Address); err != nil {
			return err
		}

		if record.Total.IsNil() || record.Total.IsNegative() {
			return ErrInvalidBurnTotal
		}
	}

	return nil
}
