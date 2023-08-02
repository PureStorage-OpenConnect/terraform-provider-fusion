data "fusion_snapshot" "snapshot_list" {
    volume = "volume1"
    tenant = "database-team"
    tenant_space = "mongodb"
}
