/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Creates and destroys
func TestAccRoleAssignment_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	rApiName := acctest.RandomWithPrefix("api_test")
	apiDisplayName := acctest.RandomWithPrefix("api-display-name")
	publicKey, err := generatePublicKey()
	if err != nil {
		t.FailNow()
	}

	tenantName := acctest.RandomWithPrefix("ra_tenant")
	tenantSpaceName := acctest.RandomWithPrefix("ra_ts")

	rOrgRoleAssignmentName := acctest.RandomWithPrefix("role_assignment")
	rTenantRoleAssignmentName := acctest.RandomWithPrefix("role_assignment")
	rTSRoleAssignmentName := acctest.RandomWithPrefix("role_assignment")

	apiConfig := testApiClientConfig(rApiName, apiDisplayName, publicKey)

	tenantConfig := testTenantConfig(tenantName, tenantName, tenantName)
	tsConfig := testTenantSpaceConfigWithRefs(tenantSpaceName, tenantSpaceName, tenantSpaceName, tenantName)

	orgRoleAssignmentConfig := testRoleAssignmentConfig(rOrgRoleAssignmentName, "az-admin", rApiName, "", "")
	tenantRoleAssignmentConfig := testRoleAssignmentConfig(rTenantRoleAssignmentName, "tenant-admin", rApiName, tenantName, "")
	tenantSpaceRoleAssignmentConfig := testRoleAssignmentConfig(rTSRoleAssignmentName, "tenant-space-admin", rApiName, tenantName, tenantSpaceName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckRoleAssignmentDestroy,
		Steps: []resource.TestStep{
			// Create Api Client and Role Assignment
			{
				Config: apiConfig + orgRoleAssignmentConfig + tenantConfig + tsConfig,
				Check: resource.ComposeTestCheckFunc(
					testApiClientExists("fusion_api_client."+rApiName),
					testRoleAssignmentExists("fusion_role_assignment."+rOrgRoleAssignmentName),
					resource.TestCheckResourceAttr("fusion_role_assignment."+rOrgRoleAssignmentName, "role_name", "az-admin"),
					resource.TestCheckNoResourceAttr("fusion_role_assignment."+rOrgRoleAssignmentName, "scope.0.tenant"),
					resource.TestCheckNoResourceAttr("fusion_role_assignment."+rOrgRoleAssignmentName, "scope.0.tenant_space"),
				),
			},
			{
				Config: apiConfig + orgRoleAssignmentConfig + tenantRoleAssignmentConfig + tenantConfig + tsConfig,
				Check: resource.ComposeTestCheckFunc(
					testRoleAssignmentExists("fusion_role_assignment."+rTenantRoleAssignmentName),
					resource.TestCheckResourceAttr("fusion_role_assignment."+rTenantRoleAssignmentName, "role_name", "tenant-admin"),
					resource.TestCheckResourceAttr("fusion_role_assignment."+rTenantRoleAssignmentName, "scope.0.tenant", tenantName),
					resource.TestCheckResourceAttr("fusion_role_assignment."+rTenantRoleAssignmentName, "scope.0.tenant_space", ""),
				),
			},
			{
				Config: apiConfig + orgRoleAssignmentConfig + tenantRoleAssignmentConfig + tenantSpaceRoleAssignmentConfig + tenantConfig + tsConfig,
				Check: resource.ComposeTestCheckFunc(
					testRoleAssignmentExists("fusion_role_assignment."+rTSRoleAssignmentName),
					resource.TestCheckResourceAttr("fusion_role_assignment."+rTSRoleAssignmentName, "role_name", "tenant-space-admin"),
					resource.TestCheckResourceAttr("fusion_role_assignment."+rTSRoleAssignmentName, "scope.0.tenant", tenantName),
					resource.TestCheckResourceAttr("fusion_role_assignment."+rTSRoleAssignmentName, "scope.0.tenant_space", tenantSpaceName),
				),
			},
		},
	})
}

func TestAccRoleAssignment_import(t *testing.T) {
	utilities.CheckTestSkip(t)

	rApiName := acctest.RandomWithPrefix("api_test")
	apiDisplayName := acctest.RandomWithPrefix("api-display-name")
	publicKey, err := generatePublicKey()
	if err != nil {
		t.FailNow()
	}

	tenantName := acctest.RandomWithPrefix("ra_tenant")
	tenantSpaceName := acctest.RandomWithPrefix("ra_ts")

	rOrgRoleAssignmentName := acctest.RandomWithPrefix("role_assignment")
	resourceName := "fusion_role_assignment." + rOrgRoleAssignmentName
	apiConfig := testApiClientConfig(rApiName, apiDisplayName, publicKey)

	tenantConfig := testTenantConfig(tenantName, tenantName, tenantName)
	tsConfig := testTenantSpaceConfigWithRefs(tenantSpaceName, tenantSpaceName, tenantSpaceName, tenantName)

	orgRoleAssignmentConfig := testRoleAssignmentConfig(rOrgRoleAssignmentName, "az-admin", rApiName, "", "")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckRoleAssignmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: apiConfig + orgRoleAssignmentConfig + tenantConfig + tsConfig,
				Check: resource.ComposeTestCheckFunc(
					testApiClientExists("fusion_api_client."+rApiName),
					testRoleAssignmentExists("fusion_role_assignment."+rOrgRoleAssignmentName),
					resource.TestCheckResourceAttr("fusion_role_assignment."+rOrgRoleAssignmentName, "role_name", "az-admin"),
					resource.TestCheckNoResourceAttr("fusion_role_assignment."+rOrgRoleAssignmentName, "scope.0.tenant"),
					resource.TestCheckNoResourceAttr("fusion_role_assignment."+rOrgRoleAssignmentName, "scope.0.tenant_space"),
				),
			},
			{
				ImportState:       true,
				ResourceName:      fmt.Sprintf("fusion_role_assignment.%s", rOrgRoleAssignmentName),
				ImportStateIdFunc: testGenerateSelfLinkForImport(resourceName, "az-admin"),
				ImportStateVerify: true,
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_role_assignment.%s", rOrgRoleAssignmentName),
				ImportStateId: fmt.Sprintf("/roles/%[1]s/role-assignments/wrong", "az-admin"),
				ExpectError:   regexp.MustCompile("Not Found"),
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_role_assignment.%s", rOrgRoleAssignmentName),
				ImportStateId: "role-assignments/",
				ExpectError:   regexp.MustCompile("invalid role_assignment import path. Expected path in format '/roles/<role>/role-assignments/<role-assignment>'"),
			},
		},
	})
}

func testRoleAssignmentExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfRoleAssignment, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfRoleAssignment.Type != "fusion_role_assignment" {
			return fmt.Errorf("expected type: fusion_role_assignment. Found: %s", tfRoleAssignment.Type)
		}
		attrs := tfRoleAssignment.Primary.Attributes

		goclientRoleAssignment, _, err := testAccProvider.Meta().(*hmrest.APIClient).RoleAssignmentsApi.GetRoleAssignmentById(context.Background(), attrs["id"], nil)
		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s by id: %s. Error: %s", attrs["name"], attrs["id"], err)
		}

		if strings.Compare(goclientRoleAssignment.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientRoleAssignment.Role.Name, attrs["role_name"]) != 0 ||
			strings.Compare(goclientRoleAssignment.Principal, attrs["principal"]) != 0 {
			return fmt.Errorf("terraform role assignment doesn't match goclients role assignment")
		}
		return nil
	}
}

func testCheckRoleAssignmentDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_role_assignment" {
			continue
		}
		attrs := rs.Primary.Attributes

		_, resp, err := client.RoleAssignmentsApi.GetRoleAssignmentById(context.Background(), attrs["id"], nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		}

		return fmt.Errorf("role assignment may still exist. Expected response code 404, got code %d", resp.StatusCode)
	}
	return nil
}

func testRoleAssignmentConfig(rName, roleName, user, tenant, tenantSpace string) string {
	scope := ""
	if tenant != "" {
		scope += fmt.Sprintf("\ntenant = fusion_tenant.%s.name", tenant)
	}

	if tenantSpace != "" {
		scope += fmt.Sprintf("\ntenant_space = fusion_tenant_space.%s.name", tenantSpace)
	}

	return fmt.Sprintf(`
	resource "fusion_role_assignment" "%[1]s" {
		role_name = "%[2]s"
		principal = fusion_api_client.%[3]s.name
		scope {
			%[4]s
		}
	}
	`, rName, roleName, user, scope)
}

func testGenerateSelfLinkForImport(resourceName, role string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		tfApiClient, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		attrs := tfApiClient.Primary.Attributes
		return fmt.Sprintf("/roles/%[1]s/role-assignments/%[2]s", role, attrs["id"]), nil
	}
}
