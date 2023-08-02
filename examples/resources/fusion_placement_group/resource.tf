resource "fusion_placement_group" "db_shard_1" {
  name              = "db-shard-1"
  display_name      = "DB shard 1"
  tenant            = "database-team"
  tenant_space      = "mongodb"
  availability_zone = "east-dc-1"
  region            = "us-east"
  storage_service   = "storage-service-generic"

  // Be careful! This will remove all snapshots in this placement group on deletion
  destroy_snapshots_on_delete = true
}
