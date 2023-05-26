/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

/* Utility code in this file is mostly used by array tests to extract preconfigured arrays
out of their az/region and later to return them exactly to the state they were in for use
in other tests. This process is slow and, from observed metrics, at the moment probably
not parallelized on the backend side. */

const (
	USE_ARRAYS_IN_ENV = "TF_ACC_USE_ENV_ARRAYS"     // set this to any non-empty value to disable poaching and use env variables below
	APPLIANCE_ID_ENV  = "TF_ACC_ARRAY_APPLIANCE_ID" // appliance ID of an array, use with suffix _N for index of said array
	HOST_NAME_ENV     = "TF_ACC_ARRAY_HOST_NAME"    // host name of an array, use with suffix _N for index of said array
	HARDWARE_TYPE_ENV = "TF_ACC_HARDWARE_TYPE"      // hardware type of an array, use with suffix _N for index of said array

	retryFor     = 20 * time.Second
	neededArrays = 3
)

type ArrayTestData struct {
	ApplianceId     string
	HostName        string
	HardwareType    string
	DisplayName     *string
	Name            *string
	ApartmentId     *string
	MaintenanceMode *bool
	UnavailableMode *bool
}

func FindArraysForTests(t *testing.T, arrayCount int) (foundArrays []ArrayTestData, deferrableReturnArrays func()) {
	if testURL == "" {
		ConfigureApiClientForTests(t)
	}
	ctx := setupTestCtx(t)

	tflog.Trace(ctx, "looking for arrays for tests", "count", arrayCount)
	if os.Getenv(USE_ARRAYS_IN_ENV) == "" {
		return poachArrays(ctx, t, arrayCount)
	} else {
		return readArraysFromEnv(ctx, t, arrayCount), func() {}
	}
}

func poachArrays(ctx context.Context, t *testing.T, arrayCount int) (poachedArrays []ArrayTestData, deferrableReturnArrays func()) {

	tflog.Trace(ctx, "poaching arrays for tests", "count", arrayCount)

	hmClient, err := NewHMClient(ctx, testURL, testIssuer, testPrivKey)
	if err != nil {
		tflog.Error(ctx, "failed to create Fusion API client", "error", err)
		t.Fatalf("NewHMClient(): %v", err)
	}

	prepareTestingEnvironment(t, ctx, hmClient)

	list, _, err := hmClient.ArraysApi.ListArrays(ctx, optionPreexistingRegion, optionPreexistingAvailabilityZone, nil)
	if err != nil {
		tflog.Error(ctx, "hmClient.ArraysApi.ListArrays()", "error", err)
		t.Fatalf("hmClient.ArraysApi.ListArrays(): %v", err)
	}
	tflog.Trace(ctx, "trying to poach arrays for tests", "region", optionPreexistingRegion, "availability_zone", optionPreexistingAvailabilityZone, "found_arrays", len(list.Items), "need_arrays", arrayCount)
	if len(list.Items) < arrayCount {
		tflog.Error(ctx, "not enough arrays to poach for tests", "need", arrayCount, "got", len(list.Items))
		t.Fatalf("%v/%v does not have enough arrays to poach for tests (%v < %v)", optionPreexistingRegion, optionPreexistingAvailabilityZone, len(list.Items), arrayCount)
	}

	wg := &sync.WaitGroup{}
	wg.Add(arrayCount)
	fh := &failHandler{}
	arrays := make([]ArrayTestData, arrayCount)
	for i := 0; i < arrayCount; i++ {
		i := i
		go poachOneArray(ctx, hmClient, i, list.Items[i], &arrays[i], wg, fh)
	}

	wg.Wait()
	fh.EvaluateFailure(t)

	tflog.Trace(ctx, "arrays successfully poached for tests", "count", len(arrays))

	releaseCallback := func() { releasePoachedArrays(ctx, t, arrays) }

	return arrays, releaseCallback
}

func poachOneArray(ctx context.Context, hmClient *hmrest.APIClient, arrayIdx int, src hmrest.Array, dst *ArrayTestData, wg *sync.WaitGroup, fail *failHandler) {
	defer wg.Done()

	var displayName *string
	if src.DisplayName != "" {
		displayName = &src.DisplayName
	}

	tflog.Trace(ctx, "poaching array for tests", "idx", arrayIdx, "name", src.Name, "appliance_id", src.ApplianceId, "host_name", src.HostName, "hardware_type", src.HardwareType.Name, "display_name", displayName, "apartment_id", src.ApartmentId)

	op, _, err := hmClient.ArraysApi.DeleteArray(ctx, optionPreexistingRegion, optionPreexistingAvailabilityZone, src.Name, nil)
	if err != nil {
		tflog.Error(ctx, "hmClient.ArraysApi.DeleteArray()", "region", optionPreexistingRegion, "availability-zone", optionPreexistingAvailabilityZone, "name", src.Name, "error", err)
		fail.SetFatal(fmt.Errorf("hmClient.ArraysApi.DeleteArray(%v): %w", src.Name, err))
		return
	}

	tflog.Trace(ctx, "created poaching operation", "region", optionPreexistingRegion, "availability_zone", optionPreexistingAvailabilityZone, "name", src.Name, "operation_id", op.Id)

	if succeeded, err := utilities.WaitOnOperation(ctx, &op, hmClient); err != nil || !succeeded {
		fail.SetFatal(fmt.Errorf("hmClient.ArraysApi.DeleteArray(%v): wait call error %v / op error %v", src.Name, err, getOperationError(&op)))
		return
	}

	*dst = ArrayTestData{
		ApplianceId:     src.ApplianceId,
		HostName:        src.HostName,
		HardwareType:    src.HardwareType.Name,
		DisplayName:     displayName,
		Name:            &src.Name,
		ApartmentId:     &src.ApartmentId,
		MaintenanceMode: &src.MaintenanceMode,
		UnavailableMode: &src.UnavailableMode,
	}

	tflog.Trace(ctx, "successfully poached array", "idx", arrayIdx)
}

func releasePoachedArrays(ctx context.Context, t *testing.T, poachedArrays []ArrayTestData) {
	tflog.Trace(ctx, "returning poached arrays back to their original AZ", "count", len(poachedArrays))

	hmClient, err := NewHMClient(ctx, testURL, testIssuer, testPrivKey)
	if err != nil {
		tflog.Error(ctx, "failed to create Fusion API client", "error", err)
		t.Fatalf("NewHMClient(): %v", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(poachedArrays))
	fh := &failHandler{}
	for i, array := range poachedArrays {
		i := i
		array := array
		go releaseOneArray(ctx, hmClient, i, array, wg, fh)
	}

	wg.Wait()
	fh.EvaluateFailure(t)

	tflog.Trace(ctx, "all arrays poached for tests successfully returned", "count", len(poachedArrays))
}

func releaseOneArray(ctx context.Context, hmClient *hmrest.APIClient, arrayIdx int, array ArrayTestData, wg *sync.WaitGroup, fail *failHandler) {
	defer wg.Done()
	tflog.Trace(ctx, "returning poached array", "idx", arrayIdx, "name", array.Name, "appliance_id", array.ApplianceId, "host_name", array.HostName, "hardware_type", array.HardwareType, "display_name", array.DisplayName, "apartment_id", array.ApartmentId)

	display_name := ""
	if array.DisplayName != nil {
		display_name = *array.DisplayName
	}
	op, _, err := hmClient.ArraysApi.CreateArray(ctx, hmrest.ArrayPost{
		Name:         *array.Name,
		DisplayName:  display_name,
		ApartmentId:  *array.ApartmentId,
		HostName:     array.HostName,
		HardwareType: array.HardwareType,
		ApplianceId:  array.ApplianceId,
	}, optionPreexistingRegion, optionPreexistingAvailabilityZone, nil)
	if err != nil {
		tflog.Error(ctx, "hmClient.ArraysApi.CreateArray()", "error", err)
		fail.SetFatal(fmt.Errorf("hmClient.ArraysApi.CreateArray(%v): %v", *array.Name, err))
		return
	}

	if succeeded, err := utilities.WaitOnOperation(ctx, &op, hmClient); err != nil || !succeeded {
		fail.SetFatal(fmt.Errorf("hmClient.ArraysApi.CreateArray(%v): wait call error %v / op error %v", *array.Name, err, getOperationError(&op)))
		return
	}

	createdArray, _, err := hmClient.ArraysApi.GetArray(ctx, optionPreexistingRegion, optionPreexistingAvailabilityZone, *array.Name, nil)
	if err != nil {
		tflog.Error(ctx, "hmClient.ArraysApi.GetArray()", "error", err)
		fail.SetFatal(fmt.Errorf("hmClient.ArraysApi.GetArray(%v): %v", *array.Name, err))
		return
	}
	var patches []hmrest.ArrayPatch
	if createdArray.MaintenanceMode != *array.MaintenanceMode {
		patches = append(patches, hmrest.ArrayPatch{
			MaintenanceMode: &hmrest.NullableBoolean{Value: *array.MaintenanceMode},
		})
	}

	if createdArray.UnavailableMode != *array.UnavailableMode {
		patches = append(patches, hmrest.ArrayPatch{
			MaintenanceMode: &hmrest.NullableBoolean{Value: *array.UnavailableMode},
		})
	}

	for _, patch := range patches {
		op, _, err = hmClient.ArraysApi.UpdateArray(ctx, patch, optionPreexistingRegion, optionPreexistingAvailabilityZone, *array.Name, nil)
		if err != nil {
			tflog.Error(ctx, "hmClient.ArraysApi.UpdateArray()", "error", err)
			fail.SetFatal(fmt.Errorf("hmClient.ArraysApi.UpdateArray(%v): %v", *array.Name, err))
			return
		}
		if succeeded, err := utilities.WaitOnOperation(ctx, &op, hmClient); err != nil || !succeeded {
			fail.SetFatal(fmt.Errorf("hmClient.ArraysApi.CreateArray(%v): wait call error %v / op error %v", *array.Name, err, getOperationError(&op)))
			return
		}
	}

	tflog.Trace(ctx, "poached array returned", "idx", arrayIdx)
}

func readArraysFromEnv(ctx context.Context, t *testing.T, arrayCount int) []ArrayTestData {
	tflog.Trace(ctx, "reading arrays to test from env", "count", arrayCount)

	arrays := make([]ArrayTestData, 0, arrayCount)
	for i := 0; i < arrayCount; i++ {
		arrays = append(arrays, ArrayTestData{
			ApplianceId:  MustGetenv(t, fmt.Sprintf("%s_%d", APPLIANCE_ID_ENV, i)),
			HostName:     MustGetenv(t, fmt.Sprintf("%s_%d", HOST_NAME_ENV, i)),
			HardwareType: MustGetenv(t, fmt.Sprintf("%s_%d", HARDWARE_TYPE_ENV, i)),
		})
	}

	return arrays
}

func MustGetenv(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("'%v' environment variable not set", key)
	}
	return value
}

type failHandler struct {
	lock sync.Mutex
	err  error
}

func (h *failHandler) SetFatal(err error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.err = err
}

func (h *failHandler) EvaluateFailure(t *testing.T) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.err != nil {
		t.Fatalf("%v", h.err)
	}
}

func prepareTestingEnvironment(t *testing.T, ctx context.Context, hmClient *hmrest.APIClient) {
	// We need arrays to test and we can't create new ones out of thin air ourselves.
	// Luckily the precreated test control plane usually comes with several of them
	// already registered. Unluckily we need to unregister them and there are
	// other infrastructure objects (e.g. placement groups) linked to them which
	// prevent us doing so, so we need to tear those down first. Unfortunately
	// the control plane is really racy around PG removals, so we need to really harden
	// this pretest teardown. Therefore the logic may look weird, there may be retries
	// and guards where they don't make sense - but really, they were placed there
	// for a reason.
	// Workflow:
	//   try to delete all Volumes, then all Placement Groups

	// await all outstanding operations first
	_ = awaitAllOperations(ctx, hmClient)

	// await until there actually are visible three arrays
	var arrays hmrest.ArrayList
	var err error
	arrays, _, err = hmClient.ArraysApi.ListArrays(ctx, optionPreexistingRegion, optionPreexistingAvailabilityZone, nil)
	if err != nil {
		t.Fatalf("hmClient.TenantSpacesApi.ListArrays(%v, %v): %v", optionPreexistingRegion, optionPreexistingAvailabilityZone, err)
	}

	// TODO: remove once HM-5548 is fixed
	// skipped only when not running in Jenkins to not skip by accident
	if os.Getenv("JENKINS_HOME") == "" {
		for _, array := range arrays.Items {
			if strings.Contains(array.HostName, "doubleagent") {
				t.Skipf("Array acceptance tests currently fail with doubleagents due to HM-5548. Skipping.")
			}
		}
	}

	if len(arrays.Items) < neededArrays {
		var names []string
		for _, array := range arrays.Items {
			names = append(names, fmt.Sprintf("'%v'", array.Name))
		}
		t.Fatalf("tried to wait until three arrays shown themselves in the control plane, but they did not (in the end got %v: %v)", len(names), strings.Join(names, ", "))
	}

	isRaceError := func(op *hmrest.Operation) bool {
		// when removing placement groups right after volume it frequently fails with no
		// details except FAILED_PRECONDITION, this is an apparent race and there seems
		// to be no no fix except to wait for sufficiently long time
		return op.Error_ != nil &&
			op.Error_.PureCode == "FAILED_PRECONDITION" &&
			op.Error_.Message == "deletion not allowed while resource is in use" &&
			len(op.Error_.Details) == 0
	}

	tenants, _, err := hmClient.TenantsApi.ListTenants(ctx, nil)
	if err != nil {
		t.Fatalf("hmClient.TenantsApi.ListTenants(): %v", err)
	}
	for _, tenant := range tenants.Items {
		tenantSpaces, _, err := hmClient.TenantSpacesApi.ListTenantSpaces(ctx, tenant.Name, nil)
		if err != nil {
			t.Fatalf("hmClient.TenantSpacesApi.ListTenantSpaces(%v): %v", tenant.Name, err)
		}
		for _, tenantSpace := range tenantSpaces.Items {
			// nuke volumes
			volumes, _, err := hmClient.VolumesApi.ListVolumes(ctx, tenant.Name, tenantSpace.Name, nil)
			if err != nil {
				t.Fatalf("hmClient.VolumesApi.ListVolumes(%v, %v): %v", tenant.Name, tenantSpace.Name, err)
			}
			for _, volume := range volumes.Items {
				tflog.Debug(ctx, "cleaning up dangling volume", "name", volume.Name)
				op, _, err := hmClient.VolumesApi.UpdateVolume(ctx, hmrest.VolumePatch{
					Destroyed: &hmrest.NullableBoolean{Value: true},
				}, tenant.Name, tenantSpace.Name, volume.Name, nil)
				if err != nil {
					t.Fatalf("hmClient.VolumesApi.UpdateVolume(%v, destroyed=true): %v", volume.Name, err)
				}
				succeeded, err := utilities.WaitOnOperation(ctx, &op, hmClient)
				if err != nil || !succeeded {
					t.Fatalf("hmClient.VolumesApi.UpdateVolume(%v, destroyed=true): call error %v / op error %v", volume.Name, err, getOperationError(&op))
				}
				startedAt := time.Now()
				removed := false
				for time.Since(startedAt) < retryFor {
					op, resp, err := hmClient.VolumesApi.DeleteVolume(ctx, tenant.Name, tenantSpace.Name, volume.Name, nil)
					if err != nil {
						if resp.StatusCode == http.StatusNotFound {
							removed = true
							break
						}
						t.Fatalf("hmClient.VolumesApi.DeleteVolume(%v, destroyed=true): %v", volume.Name, err)
					}
					succeeded, err = utilities.WaitOnOperation(ctx, &op, hmClient)
					if err != nil || (!succeeded && !isRaceError(&op)) {
						t.Fatalf("hmClient.VolumesApi.DeleteVolume(%v): wait call error %v / op error %v", volume.Name, err, getOperationError(&op))
					}
					if err == nil && succeeded {
						removed = true
						break
					}
				}
				if !removed {
					t.Fatalf("hmClient.VolumesApi.DeleteVolume(%v): failed to remove in sufficiently short time frame due to racy operations", volume.Name)
				}
			}

			// nuke placement groups
			placementGroups, _, err := hmClient.PlacementGroupsApi.ListPlacementGroups(ctx, tenant.Name, tenantSpace.Name, nil)
			if err != nil {
				t.Fatalf("hmClient.PlacementGroupsApi.ListPlacementGroups(%v, %v): %v", tenant.Name, tenantSpace.Name, err)
			}
			for _, placementGroup := range placementGroups.Items {
				tflog.Debug(ctx, "cleaning up placement group", "name", placementGroup.Name)
				startedAt := time.Now()
				removed := false
				for time.Since(startedAt) < retryFor {
					op, resp, err := hmClient.PlacementGroupsApi.DeletePlacementGroup(ctx, tenant.Name, tenantSpace.Name, placementGroup.Name, nil)
					if err != nil {
						if resp.StatusCode == http.StatusNotFound {
							removed = true
							break
						}
						t.Fatalf("hmClient.PlacementGroupsApi.DeletePlacementGroup(%v): %v", placementGroup.Name, err)
					}
					succeeded, err := utilities.WaitOnOperation(ctx, &op, hmClient)
					if err != nil || (!succeeded && !isRaceError(&op)) {
						t.Fatalf("hmClient.PlacementGroupsApi.DeletePlacementGroup(%v): wait call error %v / op error %v", placementGroup.Name, err, getOperationError(&op))
					}
					if err == nil && succeeded {
						removed = true
						break
					}
				}
				if !removed {
					t.Fatalf("hmClient.PlacementGroupsApi.DeletePlacementGroup(%v): failed to remove in sufficiently short time frame due to racy operations", placementGroup.Name)
				}
			}
		}
	}
}

func awaitAllOperations(ctx context.Context, hmClient *hmrest.APIClient) error {
	var wg = &sync.WaitGroup{}
	awaitOp := func(op hmrest.Operation) {
		// this is intentionally silent and does not log anything
		for op.Status != "Succeeded" && op.Status != "Failed" {
			delay := time.Duration(op.RetryIn) * time.Millisecond
			time.Sleep(delay)
			newOp, resp, err := hmClient.OperationsApi.GetOperation(ctx, op.Id, nil)
			if err != nil && resp.StatusCode == http.StatusNotFound {
				break
			}
			op = newOp
		}
		wg.Done()
	}

	ops, _, err := hmClient.OperationsApi.ListOperations(ctx, nil)
	if err != nil {
		return err
	}
	wg.Add(len(ops.Items))
	for _, op := range ops.Items {
		op := op
		go awaitOp(op)
	}
	wg.Wait()

	return nil
}

func getOperationError(op *hmrest.Operation) interface{} {
	if op.Error_ != nil {
		return *op.Error_
	}
	return nil
}
