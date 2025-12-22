terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {}

data "discord_servers" "all" {}

output "servers" {
  value = data.discord_servers.all.servers
}
