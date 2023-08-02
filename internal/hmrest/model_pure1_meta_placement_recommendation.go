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

type Pure1MetaPlacementRecommendation struct {
	// Raw output from Pure1 Meta Recommendation engine in JSON string format
	Raw                           string                                      `json:"raw,omitempty"`
	Objectives                    *Pure1MetaPlacementRecommendationObjectives `json:"objectives,omitempty"`
	LoadValues                    *Pure1MetaPlacementRecommendationLoadValues `json:"load_values,omitempty"`
	CapacityValues                []Pure1MetaValue                            `json:"capacity_values,omitempty"`
	DaysToReach90PercentCapacity  float64                                     `json:"days_to_reach_90_percent_capacity,omitempty"`
	DaysToReach100PercentCapacity float64                                     `json:"days_to_reach_100_percent_capacity,omitempty"`
	Error_                        string                                      `json:"error,omitempty"`
	Warnings                      []Pure1MetaWarning                          `json:"warnings,omitempty"`
}
