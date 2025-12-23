terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

# Look up emoji by ID
data "discord_emoji" "by_id" {
  emoji_id = "1452601985235816601" # Replace with your emoji ID
  guild_id = "1452601985235816601" # Replace with your guild ID
}

# Look up emoji by name
data "discord_emoji" "by_name" {
  name     = "custom_emoji"
  guild_id = "1452601985235816601" # Replace with your guild ID
}

output "emoji_by_id" {
  value = data.discord_emoji.by_id
}

output "emoji_by_name" {
  value = data.discord_emoji.by_name
}
