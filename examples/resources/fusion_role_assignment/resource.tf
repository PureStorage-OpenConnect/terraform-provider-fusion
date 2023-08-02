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
