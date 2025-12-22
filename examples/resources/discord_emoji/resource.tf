terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

# Create an emoji from a URL (using a small public emoji image)
# Note: Replace with your own emoji image URL or use image_path for local files
# Example using a small test emoji from a public CDN:
resource "discord_emoji" "from_url" {
  guild_id  = "1452601985235816601" # Replace with your guild ID
  name      = "rocket_emoji"
  image_url = "https://raw.githubusercontent.com/twitter/twemoji/master/assets/72x72/1f680.png"
  # Alternative: Use any publicly accessible PNG/JPG/GIF image URL
}

# Create an emoji from a local file
# Uncomment and adjust the path to use a local image file:
# resource "discord_emoji" "from_file" {
#   guild_id   = "1452601985235816601" # Replace with your guild ID
#   name       = "local_emoji"
#   image_path = "/path/to/your/emoji.png" # Path to a PNG, JPG, or GIF file
# }

# Create an emoji with role restrictions
# resource "discord_emoji" "restricted" {
#   guild_id  = "1452601985235816601" # Replace with your guild ID
#   name      = "admin_emoji"
#   image_url = "https://raw.githubusercontent.com/discord/discord-api-docs/main/docs/assets/example-emoji.png" # Replace with a valid emoji image URL
#   roles     = ["1452601985235816601"]                                                                         # Replace with role IDs that can use this emoji
# }

# Example using base64-encoded image (small 1x1 pixel PNG as example)
# In practice, you would use a real emoji image encoded in base64
# resource "discord_emoji" "from_base64" {
#   guild_id = "1452601985235816601" # Replace with your guild ID
#   name     = "base64_emoji"
#   image    = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==" # Base64-encoded PNG
# }

output "emoji_id" {
  value = discord_emoji.from_url.id
}

output "emoji_name" {
  value = discord_emoji.from_url.name
}
