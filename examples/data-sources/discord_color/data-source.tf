terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

# Get color from hex
data "discord_color" "blue" {
  hex = "#4287f5"
}

# Get color from RGB
data "discord_color" "green" {
  rgb = "rgb(46, 204, 113)"
}

# Use the colors in roles
resource "discord_role" "blue" {
  name     = "Blue Role"
  guild_id = "1452601985235816601" # Replace with your guild ID
  color    = data.discord_color.blue.dec
}

resource "discord_role" "green" {
  name     = "Green Role"
  guild_id = "1452601985235816601" # Replace with your guild ID
  color    = data.discord_color.green.dec
}

output "blue_color_decimal" {
  value = data.discord_color.blue.dec
}

output "green_color_decimal" {
  value = data.discord_color.green.dec
}
