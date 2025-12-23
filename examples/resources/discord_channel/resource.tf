terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

# Create a text channel
resource "discord_channel" "text" {
  name     = "terraform-text-channel"
  type     = "text"
  guild_id = "1452601985235816601" # Replace with your guild ID
}

# Create a voice channel
resource "discord_channel" "voice" {
  name     = "terraform-voice-channel"
  type     = "voice"
  guild_id = "1452601985235816601" # Replace with your guild ID
}

# Create a category channel
resource "discord_channel" "category" {
  name     = "Terraform Managed"
  type     = "category"
  guild_id = "1452601985235816601" # Replace with your guild ID
}

# Note: News channels cannot be created directly - they must be converted from text channels
# Create a text channel first, then convert it to news in Discord UI or via API
# resource "discord_channel" "news" {
#   name     = "terraform-news"
#   type     = "news"  # Not supported - news channels must be converted from text
#   guild_id = "1452601985235816601"
# }

# Note: Stage channels cannot be created by bots - Discord API limitation (error 50024)
# Stage channels must be created manually in Discord or via user OAuth2 tokens
# resource "discord_channel" "stage" {
#   name     = "terraform-stage"
#   type     = "stage"  # Not supported - bots cannot create stage channels
#   guild_id = "1452601985235816601"
# }

# Note: Forum channels cannot be created by bots - Discord API limitation (error 50024)
# Forum channels must be created manually in Discord or via user OAuth2 tokens
# resource "discord_channel" "forum" {
#   name     = "terraform-forum"
#   type     = "forum"  # Not supported - bots cannot create forum channels
#   guild_id = "1452601985235816601"
# }

# Create a text channel in a category
resource "discord_channel" "categorized_text" {
  name        = "terraform-categorized-text"
  type        = "text"
  guild_id    = "1452601985235816601" # Replace with your guild ID
  category_id = discord_channel.category.id
}

output "text_channel_id" {
  value = discord_channel.text.id
}

output "voice_channel_id" {
  value = discord_channel.voice.id
}

output "category_id" {
  value = discord_channel.category.id
}

output "categorized_text_channel_id" {
  value = discord_channel.categorized_text.id
}
