resource "fusion_tenant" "database_team" {
  name         = "database-team"
  display_name = "Database Team"
}

resource "fusion_tenant_space" "mongodb" {
  name   = "mongodb"
  tenant = fusion_tenant.database_team.name
}
