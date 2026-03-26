package burn

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"github.com/monolythium/mono-chain/x/burn/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "BurnStats",
					Use:       "burn-stats",
					Short:     "Query global burn statistics",
				},
				{
					RpcMethod:      "AccountBurns",
					Use:            "account-burns [address]",
					Short:          "Query burn total for an account",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: types.Msg_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod:      "UpdateParams",
					Use:            "update-params-proposal [params]",
					Short:          "Submit a proposal to update burn module params. Note: the entire params must be provided.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "params"}},
					GovProposal:    true,
				},
				{
					RpcMethod:      "Burn",
					Use:            "burn [amount]",
					Short:          "Send a burn tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "amount"}},
				},
			},
		},
	}
}
