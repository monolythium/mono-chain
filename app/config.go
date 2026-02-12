package app

import (
	clienthelpers "cosmossdk.io/client/v2/helpers"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	AppName = "mono"

	Bech32Prefix         = "mono"
	Bech32PrefixAccAddr  = Bech32Prefix
	Bech32PrefixAccPub   = Bech32Prefix + sdk.PrefixPublic
	Bech32PrefixValAddr  = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixOperator
	Bech32PrefixValPub   = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixOperator + sdk.PrefixPublic
	Bech32PrefixConsAddr = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixConsensus
	Bech32PrefixConsPub  = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixConsensus + sdk.PrefixPublic

	CoinType = 60 // Ethereum

	DefaultBondDenom = "alyth"
)

// DefaultNodeHome default home directory for the application daemon.
var DefaultNodeHome string

func init() {
	sdk.DefaultBondDenom = DefaultBondDenom

	clienthelpers.EnvPrefix = AppName

	var err error
	DefaultNodeHome, err = clienthelpers.GetNodeHomeDirectory("." + AppName)
	if err != nil {
		panic(err)
	}
}

// SetBech32Prefixes sets the global prefixes for Bech32 address serialization.
func SetBech32Prefixes(config *sdk.Config) {
	config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(Bech32PrefixValAddr, Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(Bech32PrefixConsAddr, Bech32PrefixConsPub)
}

// SetBip44CoinType sets the global coin type for HD wallets.
func SetBip44CoinType(config *sdk.Config) {
	config.SetCoinType(CoinType)
	config.SetPurpose(sdk.Purpose)
}

// GetMaccPerms returns a copy of the module account permissions.
func GetMaccPerms() map[string][]string {
	dup := make(map[string][]string)
	for _, perms := range moduleAccPerms {
		dup[perms.GetAccount()] = perms.GetPermissions()
	}
	return dup
}

// BlockedAddresses returns all blocked account addresses.
func BlockedAddresses() map[string]bool {
	result := make(map[string]bool)
	if len(blockAccAddrs) > 0 {
		for _, addr := range blockAccAddrs {
			result[addr] = true
		}
	} else {
		for addr := range GetMaccPerms() {
			result[addr] = true
		}
	}
	return result
}
