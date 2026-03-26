package burn_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppModule_AutoCLIOptions(t *testing.T) {
	f := setupModule(t)
	opts := f.am.AutoCLIOptions()

	require.NotNil(t, opts.Query)
	require.NotNil(t, opts.Tx)
	require.Len(t, opts.Query.RpcCommandOptions, 3)
	require.Len(t, opts.Tx.RpcCommandOptions, 2)
}
