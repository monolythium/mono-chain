package app

import (
	"github.com/spf13/cast"

	evmante "github.com/cosmos/evm/ante"
	antetypes "github.com/cosmos/evm/ante/types"
	evmaddress "github.com/cosmos/evm/encoding/address"
	srvflags "github.com/cosmos/evm/server/flags"

	"github.com/cosmos/cosmos-sdk/client"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	monoante "github.com/monolythium/mono-chain/x/mono/ante"
)

// setAnteHandler builds the cosmos/evm ante handler and wraps it
// with the validator registration burn decorator.
func (app *App) setAnteHandler(txConfig client.TxConfig, appOpts servertypes.AppOptions) {
	options := evmante.HandlerOptions{
		Cdc:                    app.appCodec,
		AccountKeeper:          app.AccountKeeper,
		BankKeeper:             app.BankKeeper,
		ExtensionOptionChecker: antetypes.HasDynamicFeeExtensionOption,
		EvmKeeper:              app.EVMKeeper,
		FeegrantKeeper:         app.FeeGrantKeeper,
		IBCKeeper:              app.IBCKeeper,
		FeeMarketKeeper:        app.FeeMarketKeeper,
		SignModeHandler:        txConfig.SignModeHandler(),
		SigGasConsumer:         evmante.SigVerificationGasConsumer,
		MaxTxGasWanted:         cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted)),
		DynamicFeeChecker:      true,
		PendingTxListener:      app.onPendingTx,
	}

	if err := options.Validate(); err != nil {
		panic(err)
	}

	evmAnteHandler := evmante.NewAnteHandler(options)

	burnDecorator := monoante.NewValidatorRegistrationBurnDecorator(
		app.MonoKeeper,
		app.BurnKeeper,
		app.StakingKeeper,
		evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
	)

	app.SetAnteHandler(func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		if !ctx.IsCheckTx() && ctx.BlockHeight() == 0 {
			return ctx, nil
		}
		newCtx, err := evmAnteHandler(ctx, tx, simulate)
		if err != nil {
			return newCtx, err
		}
		return burnDecorator.AnteHandle(newCtx, tx, simulate, passthrough)
	})
}

func passthrough(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
	return ctx, nil
}
