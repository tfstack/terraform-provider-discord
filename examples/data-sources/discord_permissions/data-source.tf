terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 1.0"
    }
  }
}

provider "discord" {}

# Build a readable role permission bitmask
data "discord_permissions" "moderator" {
  kick_members         = "allow"
  ban_members          = "allow"
  manage_messages      = "allow"
  moderate_members     = "allow"
  view_channel         = "allow"
  send_messages        = "allow"
  read_message_history = "allow"
}

# Channel overwrite: allow send, deny manage
data "discord_permissions" "channel_mod" {
  view_channel    = "allow"
  send_messages   = "allow"
  manage_messages = "deny"
}

# Example role using computed permissions
resource "discord_role" "moderator" {
  name        = "Moderator"
  guild_id    = "1452601985235816601" # Replace with your guild ID
  permissions = data.discord_permissions.moderator.permissions
}

output "moderator_permissions" {
  value = data.discord_permissions.moderator.permissions
}

output "channel_allow_bits" {
  value = data.discord_permissions.channel_mod.allow_bits
}

output "channel_deny_bits" {
  value = data.discord_permissions.channel_mod.deny_bits
}
