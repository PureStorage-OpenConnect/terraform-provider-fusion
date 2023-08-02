resource "fusion_region" "us_east" {
  name         = "us-east"
  display_name = "US East"
}

resource "fusion_availability_zone" "east_dc_1" {
  name         = "east-dc-1"
  display_name = "East DC 1"
  region       = fusion_region.us_east.name
}

resource "fusion_storage_endpoint" "east_dc_1_iscsi" {
  name              = "${fusion_availability_zone.east_dc_1.name}-iscsi"
  display_name      = "${fusion_availability_zone.east_dc_1.display_name} ISCSI Storage Endpoint"
  availability_zone = fusion_availability_zone.east_dc_1.name
  region            = fusion_availability_zone.east_dc_1.region
  iscsi {
    address                  = "172.17.1.1/16"
    gateway                  = "172.17.1.1"
    network_interface_groups = ["${fusion_availability_zone.east_dc_1.name}-primary"]
  }
}
