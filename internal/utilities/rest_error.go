/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package utilities

import (
	"fmt"
	"strconv"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

type RestError struct {
	OperationType string
	PureCode      string
	HttpCode      string
	Message       string
}

func NewRestErrorFromOperation(operation *hmrest.Operation) *RestError {
	pureCode := "unknown"
	httpCode := "unknown"
	message := "reason unknown"
	if operation.Error_ != nil {
		pureCode = operation.Error_.PureCode
		httpCode = strconv.FormatInt(int64(operation.Error_.HttpCode), 10)
		message = operation.Error_.Message
	}
	return &RestError{
		OperationType: operation.RequestType,
		PureCode:      pureCode,
		HttpCode:      httpCode,
		Message:       message,
	}
}

func (e *RestError) Error() string {
	return fmt.Sprintf("operation '%v' failed: %v (Pure '%v', Http %v)", e.OperationType, e.Message, e.PureCode, e.HttpCode)
}
