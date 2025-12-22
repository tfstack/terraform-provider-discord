terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

# Get all members in a guild
data "discord_members" "all" {
  guild_id = "1452601985235816601" # Replace with your guild ID
}

# Output all members
output "members" {
  value = data.discord_members.all.members
}

# Output all bot members
output "bot_members" {
  value = [
    for member in data.discord_members.all.members :
    member.username
    if member.bot == true
  ]
}

# Output members with a specific role (example: everyone role)
output "members_with_everyone_role" {
  value = [
    for member in data.discord_members.all.members :
    member.username
    if contains(member.roles, "1452601985235816601") # Replace with the role ID
  ]
}
