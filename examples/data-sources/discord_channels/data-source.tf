terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

data "discord_channels" "all" {
  guild_id = "123456789012345678" # Replace with your guild ID
}

output "channels" {
  value = data.discord_channels.all.channels
}
