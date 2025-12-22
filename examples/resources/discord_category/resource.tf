terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

resource "discord_category" "example" {
  name     = "Terraform Example"
  guild_id = "123456789012345678" # Replace with your guild ID
}

output "category_id" {
  value = discord_category.example.id
}
