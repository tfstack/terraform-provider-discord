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

# Create a basic webhook (without avatar)
resource "discord_webhook" "basic" {
  channel_id = discord_channel.general.id
  name       = "Terraform Webhook"
}

# Note: To set a webhook avatar, you need a Discord image hash.
# Avatar hashes can be obtained from existing Discord images (user avatars, server icons, etc.)
# or by uploading an image through the Discord API.
# Example (commented out - uncomment and replace with actual hash):
# resource "discord_webhook" "with_avatar" {
#   channel_id = discord_channel.general.id
#   name       = "Webhook with Avatar"
#   avatar     = "a_abc123def456" # Replace with actual Discord image hash
# }

output "basic_webhook_url" {
  value     = discord_webhook.basic.url
  sensitive = true
}

output "basic_webhook_token" {
  value     = discord_webhook.basic.token
  sensitive = true
}

# Uncomment if using webhook with avatar:
# output "with_avatar_webhook_url" {
#   value     = discord_webhook.with_avatar.url
#   sensitive = true
# }
