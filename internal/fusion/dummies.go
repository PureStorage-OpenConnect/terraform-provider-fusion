/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

var DummyOperation hmrest.Operation = hmrest.Operation{
	Status: "Succeeded",
}

func MakeDummyCreateOperation(resourceId string) *hmrest.Operation {
	return &hmrest.Operation{
		Status: "Succeeded",
		Result: &hmrest.OperationResult{
			Resource: &hmrest.ResourceReference{
				Id: resourceId,
			},
		},
	}
}

func DummyInvokeWriteAPI(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (operation *hmrest.Operation, err error) {
	return &DummyOperation, nil
}

func MakeDummyCreateInvokeWriteAPI(resourceId string) InvokeWriteAPI {
	return func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (operation *hmrest.Operation, err error) {
		return MakeDummyCreateOperation(resourceId), nil
	}
}
