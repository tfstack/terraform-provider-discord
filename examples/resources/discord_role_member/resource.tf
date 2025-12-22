terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

# Create a role first
resource "discord_role" "admin" {
  name     = "Admin"
  guild_id = "1452601985235816601" # Replace with your guild ID
}

# Add a user to the role
resource "discord_role_member" "admin_user1" {
  guild_id = "1452601985235816601" # Replace with your guild ID
  role_id  = discord_role.admin.id
  user_id  = "1452597490959515802" # Replace with a user ID in your guild
}

# Add another user to the same role
resource "discord_role_member" "admin_user2" {
  guild_id = "1452601985235816601" # Replace with your guild ID
  role_id  = discord_role.admin.id
  user_id  = "111111111111111111" # Replace with another user ID
}

output "admin_role_id" {
  value = discord_role.admin.id
}
