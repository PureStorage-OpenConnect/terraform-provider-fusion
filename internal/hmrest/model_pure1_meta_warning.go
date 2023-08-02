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
 * API version: 1.2
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package fusion

// Warnings returned from the PerfPlanner Recommendation Engine in Pure1Meta. Warnings do not prevent an array from being selected for Placement, but are noteworthy issues that a Provider should take a look, and might cause an array to have a lower recommendation score.
type Pure1MetaWarning struct {
	// Description of the warning
	Message string `json:"message,omitempty"`
	// Unique code identifying the warning
	WarningCode string `json:"warning_code,omitempty"`
}
