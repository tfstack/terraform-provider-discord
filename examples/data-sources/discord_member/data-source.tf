terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

# Get a specific member by user ID and guild ID
data "discord_member" "example" {
  user_id  = "1205759353127313409" # Replace with the user ID
  guild_id = "1452601985235816601" # Replace with your guild ID
}

output "member_username" {
  value = data.discord_member.example.username
}

output "member_nickname" {
  value = data.discord_member.example.nickname
}

output "member_roles" {
  value = data.discord_member.example.roles
}

output "member_joined_at" {
  value = data.discord_member.example.joined_at
}
