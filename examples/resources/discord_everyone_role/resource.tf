terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

# Manage the @everyone role
# Note: The @everyone role cannot be created or deleted, only modified
resource "discord_everyone_role" "example" {
  guild_id    = "1452601985235816601" # Replace with your guild ID
  permissions = 104324673             # Example: View Channels, Send Messages, etc.
  mentionable = false
  hoist       = false
}

output "everyone_role_id" {
  value = discord_everyone_role.example.id
}
