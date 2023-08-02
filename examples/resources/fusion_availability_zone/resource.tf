resource "fusion_region" "us_east" {
  name         = "us-east"
  display_name = "US East"
}

resource "fusion_availability_zone" "east_dc_1" {
  name         = "east-dc-1"
  display_name = "East DC 1"
  region       = fusion_region.us_east.name
}

resource "fusion_availability_zone" "east_dc_2" {
  name         = "east-dc-2"
  display_name = "East DC 2"
  region       = fusion_region.us_east.name
}
