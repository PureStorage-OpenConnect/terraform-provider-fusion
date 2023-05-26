/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

// Code generated DO NOT EDIT.
/*
 * Pure Fusion API
 *
 * Pure Fusion is fully API-driven. Most APIs which change the system (POST, PATCH, DELETE) return an Operation in status \"Pending\" or \"Running\". You can poll (GET) the operation to check its status, waiting for it to change to \"Succeeded\" or \"Failed\".
 *
 * API version: 1.1
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package fusion

type Space struct {
	Resource *ResourceReference `json:"resource,omitempty"`
	// Total physical space occupied by system, shared space, volume, and snapshot data. Measured in bytes.
	TotalPhysicalSpace int64 `json:"total_physical_space"`
	// The unique physical space occupied by customer data. Unique physical space does not include shared space, snapshots, and internal array metadata. Measured in bytes.
	UniqueSpace int64 `json:"unique_space"`
	// The sum of total physical space occupied by one or more snapshots associated with the objects. Measured in bytes.
	SnapshotSpace int64 `json:"snapshot_space"`
}
