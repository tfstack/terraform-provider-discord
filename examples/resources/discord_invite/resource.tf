terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

# Create a channel first
resource "discord_channel" "general" {
  name     = "general"
  type     = "text"
  guild_id = "1452601985235816601" # Replace with your guild ID
}

# Create a permanent invite (never expires, unlimited uses)
resource "discord_invite" "permanent" {
  channel_id = discord_channel.general.id
  max_age    = 0 # Never expires
  max_uses   = 0 # Unlimited uses
  temporary  = false
  unique     = false
}

# Create a temporary invite (expires in 7 days, 10 uses max)
resource "discord_invite" "temporary" {
  channel_id = discord_channel.general.id
  max_age    = 604800 # 7 days in seconds
  max_uses   = 10
  temporary  = true
  unique     = false
}

output "permanent_invite_url" {
  value = discord_invite.permanent.url
}

output "temporary_invite_url" {
  value = discord_invite.temporary.url
}
