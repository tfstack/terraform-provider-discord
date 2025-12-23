terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

# Create a basic role
resource "discord_role" "basic" {
  name     = "Terraform Role"
  guild_id = "1452601985235816601" # Replace with your guild ID
}

# Create a role with color and permissions
resource "discord_role" "colored" {
  name        = "Colored Role"
  guild_id    = "1452601985235816601" # Replace with your guild ID
  color       = 3447003               # Blue color (decimal)
  hoist       = true
  mentionable = true
}

output "basic_role_id" {
  value = discord_role.basic.id
}

output "colored_role_id" {
  value = discord_role.colored.id
}
