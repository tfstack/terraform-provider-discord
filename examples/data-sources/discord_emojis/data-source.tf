terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

data "discord_emojis" "all" {
  guild_id = "1452601985235816601" # Replace with your guild ID
}

output "emoji_names" {
  value = [for emoji in data.discord_emojis.all.emojis : emoji.name]
}

output "animated_emojis" {
  value = [
    for emoji in data.discord_emojis.all.emojis :
    emoji.name
    if emoji.animated == true
  ]
}

output "emoji_ids" {
  value = [for emoji in data.discord_emojis.all.emojis : emoji.id]
}
