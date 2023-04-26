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

import (
	"encoding/json"
	"errors"
)

type ProtectionPolicy struct {
	// An immutable, globally unique, system generated identifier.
	Id string `json:"id"`
	// The name of the resource, supplied by the user at creation, and used in the URI path of a resource.
	Name string `json:"name"`
	// The URI of the resource.
	SelfLink string `json:"self_link"`
	// The display name of the resource.
	DisplayName string `json:"display_name,omitempty"`
	// A JSON array of objectives
	Objectives    []OneOfProtectionPolicyObjectivesItems `json:"-"`
	RawObjectives []json.RawMessage                      `json:"objectives"`
}

func (pp *ProtectionPolicy) UnmarshalJSON(b []byte) error {
	type tmpProtectionPolicy ProtectionPolicy // to avoid infinite loop
	err := json.Unmarshal(b, (*tmpProtectionPolicy)(pp))
	if err != nil {
		return err
	}

	var i OneOfProtectionPolicyObjectivesItems
	for _, raw := range pp.RawObjectives {
		var ot ProtectionPolicyObjectiveType
		if err := json.Unmarshal(raw, &ot); err != nil {
			return err
		}

		switch ot.Type_ {
		case "RPO":
			i = &Rpo{}
		case "Retention":
			i = &Retention{}
		default:
			return errors.New("unknown objective type")
		}

		err := json.Unmarshal(raw, &i)
		if err != nil {
			return err
		}
		pp.Objectives = append(pp.Objectives, i)
	}
	pp.RawObjectives = nil
	return nil
}

func (pp ProtectionPolicy) MarshalJSON() ([]byte, error) {
	type tmpProtectionPolicy ProtectionPolicy // to avoid infinite loop
	if pp.Objectives != nil {
		for _, obj := range pp.Objectives {
			b, err := json.Marshal(obj)
			if err != nil {
				return nil, err
			}
			pp.RawObjectives = append(pp.RawObjectives, b)
		}
	}
	return json.Marshal((tmpProtectionPolicy)(pp))
}
