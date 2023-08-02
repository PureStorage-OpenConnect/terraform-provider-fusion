/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Network Interfaces are, similarly to arrays, a fixed resource which we cannot create or destroy
// on demand so we have to pull testing input either from env or from control plane we test on
// and do everything serially to not clash with other tests

// assumptions: All of the interfaces are on a single array. This is aimed at Fusion testing
// infrastructure, real arrays will obviously not satisfy this requirement.

type NetworkInterfacesTestData struct {
	EthInterfaces []hmrest.NetworkInterface
	FcInterfaces  []hmrest.NetworkInterface
}

const (
	USE_NETWORK_INTERFACES_IN_ENV = "TF_ACC_USE_ENV_NETWORK_INTERFACES" // set this to any non-empty value to disable control plane scan and use env variables below
	// all of the variables below are suffixed with "_FC_N" or "_ETH_N" when N is index of the needed interface
	REGION_ENV            = "TF_ACC_NI_REGION" // region of the array containing network interfaces to test on
	AVAILABILITY_ZONE_ENV = "TF_ACC_NI_AZ"     // availability zone of the array containing network interfaces to test on
	ARRAY_ENV             = "TF_ACC_NI_ARRAY"  // name of the array containing network interfaces to test on
	NI_NAME_ENV           = "TF_ACC_NI_NAME"   // name of network interface to test on
)

func FindNetworkInterfacesForTests(t *testing.T, ethCount, fcCount int) (data NetworkInterfacesTestData, revertFunc func()) {
	if testURL == "" {
		ConfigureApiClientForTests(t)
	}
	ctx := setupTestCtx(t)

	tflog.Trace(ctx, "looking for network interfaces for tests")

	client, err := newTestHMClient(ctx, testURL, testIssuer, testPrivKey, testPrivKeyPassword)
	if err != nil {
		tflog.Error(ctx, "failed to create Fusion API client", "error", err, "callsite", "FindNetworkInterfacesForTests")
		t.Fatalf("NewHMClient(): %v", err)
	}

	var interfaces NetworkInterfacesTestData
	if os.Getenv(USE_NETWORK_INTERFACES_IN_ENV) == "" {
		interfaces = locateNetworkInterfacesInControlPlane(ctx, t, client, ethCount, fcCount)
	} else {
		interfaces = readNetworkInterfacesFromEnv(ctx, t, client, ethCount, fcCount)
	}
	var revertFuncs []func()
	for _, iface := range interfaces.EthInterfaces {
		fn := createNetworkInterfaceRevertFunc(ctx, t, client, iface)
		revertFuncs = append(revertFuncs, fn)
	}
	for _, iface := range interfaces.FcInterfaces {
		fn := createNetworkInterfaceRevertFunc(ctx, t, client, iface)
		revertFuncs = append(revertFuncs, fn)
	}
	return interfaces, func() {
		for _, fn := range revertFuncs {
			fn()
		}
	}
}

func locateNetworkInterfacesInControlPlane(ctx context.Context, t *testing.T, client *hmrest.APIClient, ethCount, fcCount int) NetworkInterfacesTestData {
	tflog.Trace(ctx, "looking for network interfaces to test on in control plane", "region", preexistingRegion, "availability_zone", optionAvailabilityZone)

	arrays, _, err := client.ArraysApi.ListArrays(ctx, preexistingRegion, preexistingAvailabilityZone, nil)
	if err != nil {
		tflog.Error(ctx, "failed to list assumed preexisting arrays to read network interfaces for tests from", "error", err, "region", preexistingRegion, "availability_zone", optionAvailabilityZone)
		t.Fatalf("hmClient.ArraysApi.ListArrays(): %v", err)
	}
	var lastErr error
	if len(arrays.Items) == 0 {
		lastErr = fmt.Errorf("there are no registered arrays in expected region '%s'/ availability zone '%s'", preexistingRegion, preexistingAvailabilityZone)
	}
	for _, array := range arrays.Items {
		nis, _, err := client.NetworkInterfacesApi.ListNetworkInterfaces(ctx, preexistingRegion, preexistingAvailabilityZone, array.Name, nil)
		if err != nil {
			lastErr = err
			continue // silently ignore as there may be another array that would serve
		}
		result := NetworkInterfacesTestData{}
		for _, ni := range nis.Items {
			switch ni.InterfaceType {
			case "eth":
				if len(result.EthInterfaces) < ethCount {
					result.EthInterfaces = append(result.EthInterfaces, ni)
				}
			case "fc":
				if len(result.FcInterfaces) < fcCount {
					result.FcInterfaces = append(result.FcInterfaces, ni)
				}
			}
			if len(result.EthInterfaces) == ethCount && len(result.FcInterfaces) == fcCount {
				// we're done
				return result
			}
		}
	}
	tflog.Error(ctx, "failed to find at least one array which would have both Ethernet-based interface(s) and Fiber Channel-based interface(s)", "error", lastErr)
	t.Fatalf("hmClient.ArraysApi.ListArrays(): failed to find at least one array which would have both Ethernet-based interface(s) and Fiber Channel-based interface(s), err: %v", err)
	return NetworkInterfacesTestData{} // unreachable line; Go compiler doesn't account for unreachable paths
}

func readNetworkInterfacesFromEnv(ctx context.Context, t *testing.T, client *hmrest.APIClient, ethCount, fcCount int) NetworkInterfacesTestData {
	tflog.Trace(ctx, "reading interfaces to test on from env")

	var eth []hmrest.NetworkInterface
	var fc []hmrest.NetworkInterface
	for i := 0; i < ethCount; i++ {
		iface := queryNetworkInterface(ctx, t, client, "eth", i+1)
		eth = append(eth, iface)
	}
	for i := 0; i < fcCount; i++ {
		iface := queryNetworkInterface(ctx, t, client, "fc", i+1)
		fc = append(fc, iface)
	}

	return NetworkInterfacesTestData{
		EthInterfaces: eth,
		FcInterfaces:  fc,
	}
}

func queryNetworkInterface(ctx context.Context, t *testing.T, client *hmrest.APIClient, ifaceType string, ifaceIndex int) hmrest.NetworkInterface {
	suffix := fmt.Sprintf("%s_%d", strings.ToUpper(ifaceType), ifaceIndex)
	region := MustGetenv(t, fmt.Sprintf("%s_%s", REGION_ENV, suffix))
	availabilityZone := MustGetenv(t, fmt.Sprintf("%s_%s", AVAILABILITY_ZONE_ENV, suffix))
	array := MustGetenv(t, fmt.Sprintf("%s_%s", ARRAY_ENV, suffix))
	ifaceName := MustGetenv(t, fmt.Sprintf("%s_%s", NI_NAME_ENV, suffix))

	iface, _, err := client.NetworkInterfacesApi.GetNetworkInterface(ctx, region, availabilityZone, array, ifaceName, nil)
	if err != nil {
		tflog.Error(ctx, "could not get info about network interface specified in env variables", "error", err, "network_interface", ifaceName, "interface_type", ifaceType, "interface_index", ifaceIndex)
		t.Fatalf("hmClient.ArraysApi.GetNetworkInterface('%s'): failed to fetch %s network interface #%d, err: %v", ifaceName, ifaceType, ifaceIndex, err)
	}
	return iface
}

func createNetworkInterfaceRevertFunc(ctx context.Context, t *testing.T, client *hmrest.APIClient, origIface hmrest.NetworkInterface) func() {
	// safety function intended to revert the network interface into original state
	return func() {
		niAfter, _, err := client.NetworkInterfacesApi.GetNetworkInterface(ctx, origIface.Region.Name, origIface.AvailabilityZone.Name, origIface.Array.Name, origIface.Name, nil)
		if err != nil {
			tflog.Error(ctx, "failed to get network interface (revert after)", "region", origIface.Region.Name, "availability_zone", origIface.AvailabilityZone.Name, "array", origIface.Array.Name, "network_interface", origIface.Name, "error", err)
			return
		}
		hasChanges := false
		var patch hmrest.NetworkInterfacePatch
		if origIface.DisplayName != niAfter.DisplayName {
			tflog.Debug(ctx, "reverting Network Interface 'display_name'", "network_interface", origIface.Name, "original", origIface.DisplayName, "revertable", niAfter.DisplayName)
			patch.DisplayName = &hmrest.NullableString{Value: origIface.DisplayName}
			hasChanges = true
		}
		if origIface.Enabled != niAfter.Enabled {
			tflog.Debug(ctx, "reverting Network Interface 'enabled'", "network_interface", origIface.Name, "original", origIface.Enabled, "revertable", niAfter.Enabled)
			patch.Enabled = &hmrest.NullableBoolean{Value: origIface.Enabled}
			hasChanges = true
		}
		switch origIface.InterfaceType {
		case "eth":
			patch.Eth = &hmrest.NetworkInterfacePatchEth{}
			origNifg := ""
			if origIface.NetworkInterfaceGroup != nil {
				origNifg = origIface.NetworkInterfaceGroup.Name
			}
			afterNifg := ""
			if niAfter.NetworkInterfaceGroup != nil {
				afterNifg = niAfter.NetworkInterfaceGroup.Name
			}
			if origNifg != afterNifg {
				tflog.Debug(ctx, "reverting Network Interface 'network_interface_group'", "network_interface", origIface.Name, "original", origNifg, "revertable", afterNifg)
				patch.NetworkInterfaceGroup = &hmrest.NullableString{Value: origNifg}
				hasChanges = true
			}

			if origIface.Eth.Address != niAfter.Eth.Address {
				tflog.Debug(ctx, "reverting Network Interface 'eth.address'", "network_interface", origIface.Name, "original", origIface.Eth.Address, "revertable", niAfter.Eth.Address)
				patch.Eth.Address = &hmrest.NullableString{Value: origIface.Eth.Address}
				hasChanges = true
			}
			//case "fc":
			// FC is read-only at the moment
		}
		if hasChanges {
			tflog.Debug(ctx, "reverting network interface", "network_interface", origIface.Name, "patch", patch)
			op, _, err := client.NetworkInterfacesApi.UpdateNetworkInterface(ctx, patch, origIface.Region.Name, origIface.AvailabilityZone.Name, origIface.Array.Name, origIface.Name, nil)
			if err != nil {
				tflog.Error(ctx, "failed to update network interface (revert)", "region", origIface.Region.Name, "availability_zone", origIface.AvailabilityZone.Name, "array", origIface.Array.Name, "network_interface", origIface.Name, "error", err)
				t.Fatalf("hmClient.NetworkInterfacesApi.UpdateNetworkInterface('%s', '%s', '%s', '%s'): %v", origIface.Region.Name, origIface.AvailabilityZone.Name, origIface.Array.Name, origIface.Name, err)
			}

			_, _ = utilities.WaitOnOperation(ctx, &op, client)
		}
	}
}
