/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
)

// Contains specific user (randomly picked)
func TestAccUserDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("users_ds")
	config := testUserDataSourceConfig(dsNameConfig)
	ctx := setupTestCtx(t)
	hmClient := testAccPreCheckWithReturningClient(ctx, t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			// Check if user datasource contains a user that we pick from the REST API
			{
				Config: config,
				Check: func(s *terraform.State) error {
					users, _, err := hmClient.IdentityManagerApi.ListUsers(ctx, nil)
					if err != nil {
						return err
					}

					if len(users) == 0 {
						return resource.TestCheckResourceAttr("fusion_user."+dsNameConfig, "items.#", "0")(s)
					}

					user := users[0]

					userMap := []map[string]interface{}{
						{
							"id":           user.Id,
							"name":         user.Name,
							"email":        user.Email,
							"display_name": user.DisplayName,
						},
					}

					return utilities.TestCheckDataSource("fusion_user", dsNameConfig, "items", userMap)(s)
				},
			},
		},
	})
}

func testUserDataSourceConfig(dsName string) string {
	return fmt.Sprintf(`data "fusion_user" "%[1]s" {
	}`, dsName)
}
