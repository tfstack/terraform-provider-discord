terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

# Look up role by ID
data "discord_role" "by_id" {
  role_id  = "1452601985235816601" # Replace with your role ID
  guild_id = "1452601985235816601" # Replace with your guild ID
}

# Look up role by name
data "discord_role" "by_name" {
  name     = "@everyone"
  guild_id = "1452601985235816601" # Replace with your guild ID
}

output "role_by_id" {
  value = data.discord_role.by_id
}

output "role_by_name" {
  value = data.discord_role.by_name
}
