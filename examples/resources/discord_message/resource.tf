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

# Send a simple message
resource "discord_message" "hello" {
  channel_id = discord_channel.general.id
  content    = "Hello from Terraform! 👋"
}

# Send a message with text-to-speech
resource "discord_message" "tts_example" {
  channel_id = discord_channel.general.id
  content    = "This message will be read aloud!"
  tts        = true
}

output "hello_message_id" {
  value = discord_message.hello.message_id
}

output "tts_message_id" {
  value = discord_message.tts_example.message_id
}
