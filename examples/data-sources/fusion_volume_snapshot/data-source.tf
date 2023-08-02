data "fusion_volume_snapshot" "volume_snapshot_list" {
    snapshot = "snapshot1"
    tenant = "database-team"
    tenant_space = "mongodb"
    volume_id = "<volume guid>"
}
