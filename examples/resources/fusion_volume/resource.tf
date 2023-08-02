resource "fusion_placement_group" "db_shard_1" {
  name              = "db-shard-1"
  display_name      = "DB shard 1"
  tenant            = "database-team"
  tenant_space      = "mongodb"
  availability_zone = "east-dc-1"
  region            = "us-east"
  storage_service   = "storage-service-generic"
}

resource "fusion_host_access_policy" "hap_host_0" {
  name = "hap-host0"
  iqn  = "iqn.2003.05.com.redhat:xxx"
}

resource "fusion_volume" "vol1" {
  name                 = "vol1"
  display_name         = "DB volume 1"
  size                 = "4G"
  storage_class        = "storage-class-db-standard"
  tenant               = "database-team"
  tenant_space         = "mongodb"
  placement_group      = fusion_placement_group.db_shard_1.name
  host_access_policies = [fusion_host_access_policy.hap_host_0.name]
  protection_policy    = "fifteen-minutes"

  // Be careful using the below property, as this will make your volume un-recoverable
  eradicate_on_delete = true
}
