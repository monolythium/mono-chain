package simulation

import (
	"math/rand"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/monolythium/mono-chain/x/burn/keeper"
	"github.com/monolythium/mono-chain/x/burn/types"
)

// SimulateMsgBurn generates a random MsgBurn
func SimulateMsgBurn(
	ak types.AuthKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
	txGen client.TxConfig,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Select a random account to burn from
		simAccount, _ := simtypes.RandomAcc(r, accs)
		account := ak.GetAccount(ctx, simAccount.Address)

		// Get spendable balance
		spendableCoins := bk.SpendableCoins(ctx, account.GetAddress())
		if spendableCoins.Empty() {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgBurn{}), "no spendable coins"), nil, nil
		}

		// Filter for bond denom only (alyth)
		bondDenom := sdk.DefaultBondDenom
		spendableBondCoins := sdk.NewCoins()
		for _, coin := range spendableCoins {
			if coin.Denom == bondDenom {
				spendableBondCoins = spendableBondCoins.Add(coin)
				break
			}
		}

		if spendableBondCoins.Empty() {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgBurn{}), "no spendable bond denom"), nil, nil
		}

		// Generate random burn amount (1% to 50% of balance)
		bondCoin := spendableBondCoins[0]
		minBurn := math.LegacyOneDec().Quo(math.LegacyNewDec(100)) // 1%

		// Random percentage between 1% and 50%
		percentage := minBurn.Add(math.LegacyNewDecFromIntWithPrec(math.NewInt(int64(r.Intn(49))), 2))
		burnAmountDec := math.LegacyNewDecFromInt(bondCoin.Amount).Mul(percentage)
		burnAmount := burnAmountDec.TruncateInt()

		// Ensure we burn at least 1
		if burnAmount.IsZero() {
			burnAmount = math.OneInt()
		}

		// Don't burn more than we have
		if burnAmount.GT(bondCoin.Amount) {
			burnAmount = bondCoin.Amount
		}

		burnCoin := sdk.NewCoin(bondDenom, burnAmount)

		msg := &types.MsgBurn{
			FromAddress: simAccount.Address.String(),
			Amount:      burnCoin,
		}

		// Create and sign transaction
		txCtx := simulation.OperationInput{
			R:               r,
			App:             app,
			TxGen:           txGen,
			Cdc:             nil,
			Msg:             msg,
			Context:         ctx,
			SimAccount:      simAccount,
			AccountKeeper:   ak,
			Bankkeeper:      bk,
			ModuleName:      types.ModuleName,
			CoinsSpentInMsg: sdk.NewCoins(burnCoin),
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

// Helper function to generate random burn parameters for testing
func RandomBurnAmount(r *rand.Rand, spendable sdk.Coin) sdk.Coin {
	if spendable.IsZero() {
		return sdk.NewCoin(spendable.Denom, math.ZeroInt())
	}

	// Burn between 1 and 100% of spendable
	percentage := r.Intn(100) + 1
	amount := spendable.Amount.MulRaw(int64(percentage)).QuoRaw(100)

	if amount.IsZero() {
		amount = math.OneInt()
	}

	return sdk.NewCoin(spendable.Denom, amount)
}
