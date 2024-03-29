---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fusion_role_assignment Resource - public"
subcategory: ""
description: |-
  A role assignment records that a principal (User or API Client) is assigned to a role, scoped to a particular resource and its chidren.
---

# fusion_role_assignment (Resource)

A role assignment records that a principal (User or API Client) is assigned to a role, scoped to a particular resource and its chidren.

## Example Usage

```terraform
data "fusion_user" "jane_doe" {
  name = "Jane Doe"
}

resource "fusion_role_assignment" "ra_jane_doe_database_team_admin" {
  role_name = "tenant-admin"
  principal = data.fusion_user.jane_doe.items[0].id # user as principal
  scope {
    tenant = "database-team"
  }
}

data "local_file" "john_doe_pubkey" {
  filename = "/path/to/john_doe_rsa.pub"
}

resource "fusion_api_client" "john_doe" {
  display_name = "John Doe"
  public_key   = local_file.john_doe_pubkey.content
}

resource "fusion_role_assignment" "ra_john_doe_mongodb_admin" {
  role_name = "tenant-space-admin"
  principal = fusion_api_client.john_doe.id # API client as principal
  scope {
    tenant       = "database-team" # when assigning on the tenant-space level, both tenant and tenant_space must be provided
    tenant_space = "mongodb"
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `principal` (String) The unique ID of the principal (User or API Client) to assign to the Role.
- `role_name` (String) The name of the Role to be assigned.
- `scope` (Block List, Min: 1, Max: 1) The level to which the Role is assigned. Empty scope sets the scope to the whole organization. (see [below for nested schema](#nestedblock--scope))

### Read-Only

- `id` (String) The ID of this resource.
- `name` (String) The name of the Role Assignment.

<a id="nestedblock--scope"></a>
### Nested Schema for `scope`

Optional:

- `tenant` (String) The name of the Tenant the user has the Role applied to.
- `tenant_space` (String) The name of the Tenant Space the user has the Role applied to.

## Import

Import is supported using the following syntax:

```shell
terraform import fusion_role_assignment.ra_john_doe_mongodb_admin "/roles/tenant-space-admin/role-assignments/john-doe-mongodb-admin"
```
