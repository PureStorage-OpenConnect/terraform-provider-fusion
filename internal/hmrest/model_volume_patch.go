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

type VolumePatch struct {
	DisplayName              *NullableString  `json:"display_name,omitempty"`
	SourceVolumeSnapshotLink *NullableString  `json:"source_volume_snapshot_link,omitempty"`
	Size                     *NullableSize    `json:"size,omitempty"`
	StorageClass             *NullableString  `json:"storage_class,omitempty"`
	PlacementGroup           *NullableString  `json:"placement_group,omitempty"`
	ProtectionPolicy         *NullableString  `json:"protection_policy,omitempty"`
	HostAccessPolicies       *NullableString  `json:"host_access_policies,omitempty"`
	Destroyed                *NullableBoolean `json:"destroyed,omitempty"`
}
