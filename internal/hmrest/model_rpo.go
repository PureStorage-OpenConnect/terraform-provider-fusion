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

// Data protection RPO objective.
type Rpo struct {
	// Type of the objective. \"RPO\" or \"Retention\".
	Type_ string `json:"type"`
	// RPO objective value in seconds. Format: https://en.wikipedia.org/wiki/ISO_8601
	Rpo string `json:"rpo"`
}