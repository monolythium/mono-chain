//go:build sims

package app

import (
	"encoding/json"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simulationtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func init() {
	cfg := sdk.GetConfig()
	SetBech32Prefixes(cfg)
	SetBip44CoinType(cfg)
}

func SimDefaultGenesis(app *App) simulationtypes.AppStateFn {
	return simtestutil.AppStateFnWithExtendedCbs(
		app.AppCodec(),
		app.SimulationManager(),
		app.DefaultGenesis(),
		nil,
		func(rawState map[string]json.RawMessage) {
			simDefaultEVMGenesis(app, rawState)

			simDefaultBankGenesis(app, rawState)

			simDefaultFeeMarket(app, rawState)
		},
	)
}

func simDefaultEVMGenesis(app *App, rawState map[string]json.RawMessage) {
	var evmGenesis evmtypes.GenesisState
	app.appCodec.MustUnmarshalJSON(rawState[evmtypes.ModuleName], &evmGenesis)
	evmGenesis.Params.EvmDenom = DefaultBondDenom
	if evmGenesis.Params.ExtendedDenomOptions == nil {
		evmGenesis.Params.ExtendedDenomOptions = &evmtypes.ExtendedDenomOptions{}
	}
	evmGenesis.Params.ExtendedDenomOptions.ExtendedDenom = DefaultBondDenom
	rawState[evmtypes.ModuleName] = app.appCodec.MustMarshalJSON(&evmGenesis)
}

func simDefaultBankGenesis(app *App, rawState map[string]json.RawMessage) {
	var bankGenesis banktypes.GenesisState
	app.appCodec.MustUnmarshalJSON(rawState[banktypes.ModuleName], &bankGenesis)
	bankGenesis.DenomMetadata = append(bankGenesis.DenomMetadata, banktypes.Metadata{
		Description: "The native token of Monolythium",
		Base:        "alyth",
		Display:     "lyth",
		Name:        "Monolythium",
		Symbol:      "LYTH",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "alyth", Exponent: 0, Aliases: []string{"attolyth"}},
			{Denom: "lyth", Exponent: 18, Aliases: []string{}},
		},
	})
	rawState[banktypes.ModuleName] = app.appCodec.MustMarshalJSON(&bankGenesis)
}

func simDefaultFeeMarket(app *App, rawState map[string]json.RawMessage) {
	var feemarketGenesis feemarkettypes.GenesisState
	app.appCodec.MustUnmarshalJSON(rawState[feemarkettypes.ModuleName], &feemarketGenesis)
	feemarketGenesis.Params.NoBaseFee = true
	feemarketGenesis.Params.BaseFee = math.LegacyZeroDec()
	rawState[feemarkettypes.ModuleName] = app.appCodec.MustMarshalJSON(&feemarketGenesis)
}
