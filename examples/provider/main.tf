provider "fusion" {
    host             = var.hm_url
    issuer_id        = var.issuer_id
    private_key_file = var.private_key
}

resource "fusion_tenant_space" "fts" {
  name         = var.tenant_space_name
  display_name = var.tenant_space_display_name
  tenant_name  = var.tenant_name
}

resource "fusion_host_access_policy" "host_access_policy" {
  name          = "testhap"
  display_name  = "TestHostAccessPlcy"
  iqn           = "iqn.year-mo.org.debian:XX:XXXXXXXXXXXX"
  personality   = "linux"
}

resource "fusion_placement_group" "placement_group" {
  name                   = "pg-name"
  display_name           = "pg-display-name"
  tenant_name            = var.tenant_name
  tenant_space_name      = fusion_tenant_space.fts.name
  region_name            = var.region_name
  availability_zone_name = var.availability_zone
  storage_service_name   = var.storage_service
}

