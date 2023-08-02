/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func deleteSnapshot(ctx context.Context, snapshot hmrest.Snapshot, client *hmrest.APIClient) error {
	patchBody := hmrest.SnapshotPatch{Destroyed: &hmrest.NullableBoolean{Value: true}}
	op, _, err := client.SnapshotsApi.UpdateSnapshot(ctx, patchBody, snapshot.Tenant.Name, snapshot.TenantSpace.Name, snapshot.Name, nil)

	utilities.TraceOperation(ctx, &op, "Destroying Snapshot")
	if err != nil {
		return err
	}

	succeeded, err := utilities.WaitOnOperation(ctx, &op, client)
	if err != nil {
		return err
	}

	if !succeeded {
		return fmt.Errorf("operation failed Message:%s ID:%s", op.Error_.Message, op.Id)
	}

	op, _, err = client.SnapshotsApi.DeleteSnapshot(ctx, snapshot.Tenant.Name, snapshot.TenantSpace.Name, snapshot.Name, nil)

	utilities.TraceOperation(ctx, &op, "Deleting Snapshot")
	if err != nil {
		return err
	}

	succeeded, err = utilities.WaitOnOperation(ctx, &op, client)
	if err != nil {
		return err
	}

	if !succeeded {
		return fmt.Errorf("operation failed Message:%s ID:%s", op.Error_.Message, op.Id)
	}

	return nil
}

func createSnapshot(ctx context.Context, snapshotPost *hmrest.SnapshotPost, tenant, tenantSpace string, client *hmrest.APIClient) error {
	op, _, err := client.SnapshotsApi.CreateSnapshot(ctx, *snapshotPost, tenant, tenantSpace, nil)
	if err != nil {
		return fmt.Errorf("cannot create snapshot %s", err)
	}
	ok, err := utilities.WaitOnOperation(ctx, &op, client)
	if err != nil {
		return fmt.Errorf("cannot create snapshot %s", err)
	}
	if !ok {
		return fmt.Errorf("cannot create snapshot op_id:%s", op.Id)
	}
	return nil
}

func deleteSnapshots(ctx context.Context, snapshots *hmrest.SnapshotList, client *hmrest.APIClient) {
	for _, snap := range snapshots.Items {
		tflog.Trace(ctx, "Deleting Snapshot", "name", snap.Name)

		// Might be possible to run in goroutines
		_ = deleteSnapshot(ctx, snap, client)
	}
}
