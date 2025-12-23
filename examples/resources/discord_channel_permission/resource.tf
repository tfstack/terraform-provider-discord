terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

resource "discord_channel_permission" "example" {
  channel_id   = "123456789012345678" # Replace with your channel ID
  type         = "role"
  overwrite_id = "987654321098765432" # Replace with your role ID
  allow        = 3072                 # VIEW_CHANNEL | SEND_MESSAGES
  deny         = 0
}
