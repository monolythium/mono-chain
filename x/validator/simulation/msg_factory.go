package simulation

import (
	"context"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil/simsx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/monolythium/mono-chain/x/validator/keeper"
	"github.com/monolythium/mono-chain/x/validator/types"
)

// StakingQuerier provides read-only access to check validator existence.
type StakingQuerier interface {
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
}

// MsgRegisterValidatorFactory creates a simulation factory for MsgRegisterValidator.
// Requires the keeper to read current params (minimum burn, minimum self-delegation),
// a staking querier to skip accounts that are already validators, and the validator
// address codec for EVM bech32 encoding.
func MsgRegisterValidatorFactory(
	k keeper.Keeper,
	sk StakingQuerier,
	valAddrCodec address.Codec,
) simsx.SimMsgFactoryFn[*types.MsgRegisterValidator] {
	return func(
		ctx context.Context,
		testData *simsx.ChainDataSource,
		reporter simsx.SimulationReporter,
	) ([]simsx.SimAccount, *types.MsgRegisterValidator) {
		params, err := k.Params.Get(ctx)
		if err != nil {
			reporter.Skip("failed to get validator params")
			return nil, nil
		}

		r := testData.Rand()
		bondDenom := sdk.DefaultBondDenom

		minBurn := params.ValidatorRegistrationBurn.Amount
		minSelfDel := params.ValidatorMinSelfDelegation.Amount
		totalRequired := minBurn.Add(minSelfDel)

		// Skip accounts that are already validators (genesis or previous registrations).
		notValidator := simsx.SimAccountFilterFn(func(a simsx.SimAccount) bool {
			_, err := sk.GetValidator(ctx, sdk.ValAddress(a.Address))
			return err != nil // err means not found → eligible
		})

		sender := testData.AnyAccount(
			reporter,
			simsx.WithLiquidBalanceGTE(sdk.NewCoin(bondDenom, totalRequired)),
			notValidator,
		)
		if reporter.IsSkipped() {
			return nil, nil
		}

		available := sender.LiquidBalance().AmountOf(bondDenom)

		// Burn: random in [minBurn, available - minSelfDel]
		maxBurn := available.Sub(minSelfDel)
		burnAmt := minBurn
		if maxBurn.GT(minBurn) {
			delta, err := r.PositiveSDKIntInRange(math.ZeroInt(), maxBurn.Sub(minBurn))
			if err == nil {
				burnAmt = minBurn.Add(delta)
			}
		}

		// Self-delegation: random in [minSelfDel, available - burnAmt]
		remaining := available.Sub(burnAmt)
		selfDelAmt := minSelfDel
		if remaining.GT(minSelfDel) {
			delta, err := r.PositiveSDKIntInRange(math.ZeroInt(), remaining.Sub(minSelfDel))
			if err == nil {
				selfDelAmt = minSelfDel.Add(delta)
			}
		}

		// Block the total spend from the liquid balance
		sender.LiquidBalance().BlockAmount(sdk.NewCoin(bondDenom, burnAmt.Add(selfDelAmt)))

		valAddr, err := valAddrCodec.BytesToString(sender.Address)
		if err != nil {
			reporter.Skip("failed to encode validator address")
			return nil, nil
		}

		pubKeyAny, err := codectypes.NewAnyWithValue(sender.ConsKey.PubKey())
		if err != nil {
			reporter.Skip("failed to pack pubkey")
			return nil, nil
		}

		description := stakingtypes.NewDescription(
			r.StringN(10),
			r.StringN(10),
			r.StringN(10),
			r.StringN(10),
			r.StringN(10),
		)

		maxCommission := math.LegacyNewDecWithPrec(int64(r.IntInRange(1, 100)), 2)
		commission := stakingtypes.NewCommissionRates(
			r.DecN(maxCommission),
			maxCommission,
			r.DecN(maxCommission),
		)

		createVal := stakingtypes.MsgCreateValidator{
			Description:       description,
			Commission:        commission,
			MinSelfDelegation: selfDelAmt,
			ValidatorAddress:  valAddr,
			Pubkey:            pubKeyAny,
			Value:             sdk.NewCoin(bondDenom, selfDelAmt),
		}

		msg := &types.MsgRegisterValidator{
			Sender:          sender.AddressBech32,
			CreateValidator: createVal,
			Burn:            sdk.NewCoin(params.ValidatorRegistrationBurn.Denom, burnAmt),
		}

		return []simsx.SimAccount{sender}, msg
	}
}

// MsgUpdateParamsFactory creates a simulation factory for validator MsgUpdateParams governance proposals.
func MsgUpdateParamsFactory() simsx.SimMsgFactoryFn[*types.MsgUpdateParams] {
	return func(
		_ context.Context,
		testData *simsx.ChainDataSource,
		reporter simsx.SimulationReporter,
	) ([]simsx.SimAccount, *types.MsgUpdateParams) {
		r := testData.Rand()

		registrationBurn := sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(int64(r.IntInRange(0, 1000))))
		minSelfDelegation := sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(int64(r.IntInRange(0, 1000))))

		return nil, &types.MsgUpdateParams{
			Authority: testData.ModuleAccountAddress(reporter, types.GovModuleName),
			Params:    types.NewParams(registrationBurn, minSelfDelegation),
		}
	}
}
