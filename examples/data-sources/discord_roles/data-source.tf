terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

data "discord_roles" "all" {
  guild_id = "1452601985235816601" # Replace with your guild ID
}

output "role_names" {
  value = [for role in data.discord_roles.all.roles : role.name]
}

output "role_ids" {
  value = [for role in data.discord_roles.all.roles : role.id]
}
