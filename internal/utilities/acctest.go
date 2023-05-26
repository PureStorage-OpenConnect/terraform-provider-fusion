/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package utilities

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestCheckDataSource(
	dataSourceType, dataSourceName, listFieldName string, items []map[string]interface{},
) resource.TestCheckFunc {
	// Fail when actual size is less than expected
	// We don't test (in)equality, because there might be more resources than expected (e.g., created in other tests)
	sizeComparator := func(actual, expected int) bool {
		return actual < expected
	}

	return testCheckDataSourceGeneric(dataSourceType, dataSourceName, listFieldName, items, sizeComparator)
}

func TestCheckDataSourceExact(
	dataSourceType, dataSourceName, listFieldName string, items []map[string]interface{},
) resource.TestCheckFunc {
	// Test that there are only expected resources
	sizeComparator := func(actual, expected int) bool {
		return actual != expected
	}

	return testCheckDataSourceGeneric(dataSourceType, dataSourceName, listFieldName, items, sizeComparator)
}

func testCheckDataSourceGeneric(
	dataSourceType, dataSourceName, listFieldName string, items []map[string]interface{}, sizeComparator func(int, int) bool,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		dataSource := s.RootModule().Resources[fmt.Sprintf("data.%s.%s", dataSourceType, dataSourceName)]

		actualSize, err := strconv.Atoi(dataSource.Primary.Attributes[listFieldName+".#"])
		if err != nil {
			return err
		}

		if sizeComparator(actualSize, len(items)) {
			return fmt.Errorf("unexpected %s size: %d, expected: %d", dataSourceName, actualSize, len(items))
		}

		// Iterate over expected items and try to find them in the data source list
		for _, item := range items {
			found := false
			expectedName := item["name"].(string)

			for i := 0; i < actualSize; i++ {
				actualName := dataSource.Primary.Attributes[fmt.Sprintf("%s.%d.name", listFieldName, i)]

				if actualName != expectedName {
					continue
				}

				for key, value := range item {
					actualValue := dataSource.Primary.Attributes[fmt.Sprintf("%s.%d.%s", listFieldName, i, key)]
					if boolValue, ok := value.(bool); ok {
						value = strconv.FormatBool(boolValue) // convert bool to string to match Terraform representation (Terraform stores all attributes as string)
					}

					if _, ok := value.(string); len(actualValue) == 0 && !ok {
						continue // Skip if the actualValue is not a scalar (thus is empty)
					}

					if actualValue != value {
						return fmt.Errorf(
							"unexpected %s value %s of key %s. Expected %s", dataSourceName, actualValue, key, value,
						)
					}
				}

				found = true
				break
			}

			if !found {
				return fmt.Errorf("cannot find resource with name %s in %s data source", expectedName, dataSourceName)
			}
		}

		return nil
	}
}

func CheckStrAttribute(t *testing.T, attributeName, found, stored string) bool {
	if found != stored {
		t.Errorf("attribute '%v' mismatch, TF: '%v', real: '%v'", attributeName, found, stored)
		return false
	}
	return true
}

func CheckBoolAttribute(t *testing.T, attributeName string, found bool, stored string) bool {
	foundStr := "false"
	if found {
		foundStr = "true"
	}
	if foundStr != stored {
		t.Errorf("attribute '%v' mismatch, TF: '%v', real: '%v'", attributeName, found, stored)
		return false
	}
	return true
}
