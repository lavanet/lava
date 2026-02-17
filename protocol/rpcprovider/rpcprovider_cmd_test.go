package rpcprovider

import "testing"

func TestCreateRPCProviderCobraCommand_HasDisableConsistencyFlag(t *testing.T) {
	cmd := CreateRPCProviderCobraCommand()
	flag := cmd.Flags().Lookup("disable-consistency-checks")
	if flag == nil {
		t.Fatalf("expected flag disable-consistency-checks to exist")
	}
	if flag.DefValue != "false" {
		t.Fatalf("expected default false, got %q", flag.DefValue)
	}
}
