resource "fusion_region" "us_east" {
  name         = "us-east"
  display_name = "US East"
}

resource "fusion_availability_zone" "east_dc_1" {
  name         = "east-dc-1"
  display_name = "East DC 1"
  region       = fusion_region.us_east.name
}

resource "fusion_network_interface_group" "east_dc_1_primary" {
  name              = "${fusion_availability_zone.east_dc_1.name}-primary"
  display_name      = "${fusion_availability_zone.east_dc_1.display_name} Primary NIG"
  availability_zone = fusion_availability_zone.east_dc_1.name
  region            = fusion_availability_zone.east_dc_1.region
  group_type        = "eth"
  eth {
    prefix  = "172.17.1.1/16"
    gateway = "172.17.1.1"
  }
}
