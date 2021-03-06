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
 * API version: 1.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package fusion

type VolumePost struct {
	// The name of the resource, supplied by the user at creation, and used in the URI path of a resource.
	Name string `json:"name"`
	// The display name of the resource.
	DisplayName string `json:"display_name,omitempty"`
	// The size of the volume to provision
	Size int64 `json:"size"`
	// The name of the Storage Class
	StorageClass string `json:"storage_class"`
	// The name of the Placement Group
	PlacementGroup string `json:"placement_group"`
	// The name of the Protection Policy
	ProtectionPolicy string `json:"protection_policy,omitempty"`
	// Unimplemented - The link to the volume snapshot to copy data from
	SourceVolumeSnapshotLink string `json:"source_volume_snapshot_link,omitempty"`
}
