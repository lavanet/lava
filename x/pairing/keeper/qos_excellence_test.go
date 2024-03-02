package keeper_test

import (
	"testing"
)

// TODO: All tests are not implemented since providerQosFS update and Qos score are not implemented yet

// TestProviderQosMap checks that getting a providers' Qos map for specific chainID and cluster works properly
func TestProviderQosMap(t *testing.T) {
}

// TestGetQos checks that using GetQos() returns the right Qos
func TestGetQos(t *testing.T) {
}

// TestQosReqForSlots checks that if Qos req is active, all slots are assigned with Qos req
func TestQosReqForSlots(t *testing.T) {
}

// TestQosScoreCluster that consumer pairing uses the correct cluster for QoS score calculations.
func TestQosScoreCluster(t *testing.T) {
}

// TestQosScore checks that the qos score component is as expected (score == ComputeQos(), new users (sub usage less
// than a month) are not influenced by Qos score, invalid Qos score == 1)
func TestQosScore(t *testing.T) {
}

// TestUpdateClusteringCriteria checks that updating the clustering criteria doesn't make different version clusters to be mixed
func TestUpdateClusteringCriteria(t *testing.T) {
}
