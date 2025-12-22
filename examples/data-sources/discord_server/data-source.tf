terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

data "discord_server" "example" {
  server_id = "1452601985235816601" # Replace with your server ID
}

output "server" {
  value = data.discord_server.example
}
