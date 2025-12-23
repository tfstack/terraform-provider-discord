terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

resource "discord_server" "example" {
  name = "Terraform Managed Server"
}

output "server_id" {
  value = discord_server.example.id
}
