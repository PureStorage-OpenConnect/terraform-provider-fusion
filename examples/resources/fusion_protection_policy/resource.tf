resource "fusion_protection_policy" "fifteen_minutes" {
  name            = "fifteen-minutes"
  display_name    = "FifteenMin: RPO 15min, retention 24h"
  local_rpo       = 15
  local_retention = "24H"


  // Be careful! This will remove all snapshots in this protection policy on deletion
  destroy_snapshots_on_delete = true
}

resource "fusion_protection_policy" "daily_for_month" {
  name            = "daily-for-month"
  display_name    = "DailyForMonth: RPO 1 day, retention 30days"
  local_rpo       = 1440
  local_retention = "30D"
}
