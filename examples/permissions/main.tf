data "codefresh_team" "admins" {
  name = "admins"
}

data "codefresh_team" "developers" {
  name = "developers"
}

resource "codefresh_permission" "dev_pipeline" {
  for_each = toset(["run", "create", "update", "delete", "read"])
  team = data.codefresh_team.developers.id
  action = each.value
  resource = "pipeline"
  tags = [ "dev", "untagged"]
}

resource "codefresh_permission" "admin_pipeline" {
  for_each = toset(["run", "create", "update", "delete", "read", "approve"])
  team = data.codefresh_team.admins.id
  action = each.value
  resource = "pipeline"
  tags = [ "production", "*"]
}
