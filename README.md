# Terraform Provider for Discord

A Terraform provider for managing Discord servers, channels, roles, members, emojis, and other resources using the Terraform Plugin Framework.

## Features

- **Server Management**: Create and manage Discord servers (guilds)
- **Channel Management**: Create and manage text, voice, category, and other channel types
- **Role Management**: Create, update, and manage roles including the @everyone role
- **Member Management**: Assign roles to members and query member information
- **Emoji Management**: Create and manage custom emojis
- **Webhooks & Messages**: Create webhooks and send/manage messages
- **Invites**: Create and manage channel invites
- **Data Sources**: Query Discord servers, channels, roles, members, and more

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (to build the provider plugin)

## Installation

### Using Terraform Registry

Add the provider to your Terraform configuration:

```hcl
terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {
  token = var.discord_bot_token
}
```

### Building from Source

1. Clone the repository:

   ```bash
   git clone https://github.com/tfstack/terraform-provider-discord.git
   cd terraform-provider-discord
   ```

2. Build the provider:

   ```bash
   go install
   ```

3. Install the provider to your local Terraform plugins directory:

   ```bash
   mkdir -p ~/.terraform.d/plugins/registry.terraform.io/tfstack/discord/0.1.0/linux_amd64
   cp $GOPATH/bin/terraform-provider-discord ~/.terraform.d/plugins/registry.terraform.io/tfstack/discord/0.1.0/linux_amd64/
   ```

## Authentication

The provider requires a Discord bot token to authenticate with the Discord API.

### Option 1: Provider Configuration Block

```hcl
provider "discord" {
  token = "your-bot-token-here"
}
```

### Option 2: Environment Variable

Set the `DISCORD_BOT_TOKEN` environment variable:

```bash
export DISCORD_BOT_TOKEN="your-bot-token-here"
```

Then use the provider without the token attribute:

```hcl
provider "discord" {
  # token will be read from DISCORD_BOT_TOKEN environment variable
}
```

### Getting a Bot Token

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application or select an existing one
3. Navigate to the "Bot" section
4. Click "Reset Token" or "Copy" to get your bot token
5. Make sure to enable the necessary bot permissions and intents (see below)

### Privileged Gateway Intents

Some data sources require privileged intents to be enabled in the Discord Developer Portal:

- **`GUILD_MEMBERS` Intent** - Required for:
  - `discord_member` data source
  - `discord_members` data source
  - `discord_role_member` resource

To enable privileged intents:

1. Go to your application in the [Discord Developer Portal](https://discord.com/developers/applications)
2. Navigate to the "Bot" section
3. Scroll down to "Privileged Gateway Intents"
4. Enable the required intents (e.g., "Server Members Intent")
5. Save changes

**Note**: Privileged intents require verification for bots in 100+ servers. See [Discord's documentation](https://discord.com/developers/docs/topics/gateway#privileged-intents) for more information.

### Required Bot Permissions

The bot requires the following permissions in the Discord server (guild) where you want to manage resources:

#### Required Permissions

- **Manage Channels** (`MANAGE_CHANNELS`) - **REQUIRED**
  - Required for: Creating, updating, and deleting channels and categories
  - Required for: Setting channel permission overwrites
  - This is the primary permission needed for all channel management operations

#### Recommended Permissions

- **View Channels** (`VIEW_CHANNELS`) - **RECOMMENDED**
  - Required for: Reading channel information via data sources
  - Required for: Listing channels in a guild
  - While not strictly required for all operations, it's needed for data sources to work

#### Permission Summary by Resource

| Resource/Data Source                                     | Required Permissions                                                                             |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `discord_channel` (create/update/delete)                 | `MANAGE_CHANNELS`                                                                                |
| `discord_category` (create/update/delete)                | `MANAGE_CHANNELS`                                                                                |
| `discord_channel_permission` (create/update/delete)      | `MANAGE_CHANNELS`                                                                                |
| `discord_role` (create/update/delete)                    | `MANAGE_ROLES` + bot role above target role                                                      |
| `discord_role_member` (add/remove)                       | `MANAGE_ROLES` + bot role above target role                                                      |
| `discord_emoji` (create/update/delete)                   | `MANAGE_EMOJIS_AND_STICKERS`                                                                     |
| `discord_everyone_role` (update color/hoist/mentionable) | `MANAGE_ROLES` + bot role above @everyone                                                        |
| `discord_everyone_role` (update permissions)             | `MANAGE_ROLES` + bot role above @everyone + **all permissions being granted** OR `ADMINISTRATOR` |
| `discord_invite` (create/update/delete)                  | `CREATE_INSTANT_INVITE`                                                                          |
| `discord_webhook` (create/update/delete)                 | `MANAGE_WEBHOOKS`                                                                                |
| `discord_message` (create)                               | `SEND_MESSAGES`                                                                                  |
| `discord_message` (update/delete)                        | `MANAGE_MESSAGES`                                                                                |
| `discord_channel` (data source)                          | `VIEW_CHANNELS`                                                                                  |
| `discord_channels` (data source)                         | `VIEW_CHANNELS`                                                                                  |
| `discord_category` (data source)                         | `VIEW_CHANNELS`                                                                                  |
| `discord_role` (data source)                             | `VIEW_SERVER` or `MANAGE_ROLES`                                                                  |
| `discord_roles` (data source)                            | `VIEW_SERVER` or `MANAGE_ROLES`                                                                  |

#### How to Set Bot Permissions

1. **In Discord Developer Portal:**

   - Go to your application → OAuth2 → URL Generator
   - Select the `bot` scope
   - Select the following permissions:
     - ✅ **Manage Channels**
     - ✅ **View Channels** (for data sources)
   - Copy the generated URL and open it in your browser
   - Select the server and authorize the bot

2. **In Discord Server:**
   - Go to Server Settings → Roles
   - Find your bot's role
   - Enable the **Manage Channels** permission
   - Enable the **View Channels** permission (if using data sources)

**Note**: The bot must be a member of the server before it can perform any operations. Make sure to invite the bot to your server using the OAuth2 URL or manually add it.

## Quick Start

Here's a simple example to get you started:

```hcl
terraform {
  required_providers {
    discord = {
      source  = "tfstack/discord"
      version = "~> 0.1"
    }
  }
}

provider "discord" {
  # Token will be read from DISCORD_BOT_TOKEN environment variable
}

# Get information about a server
data "discord_server" "main" {
  server_id = "123456789012345678" # Replace with your server ID
}

# Create a text channel
resource "discord_channel" "general" {
  name     = "general"
  type     = "text"
  guild_id = data.discord_server.main.id
}

# Create a role
resource "discord_role" "member" {
  name     = "Member"
  guild_id = data.discord_server.main.id
  color    = data.discord_color.blue.decimal
}

# Convert hex color to decimal for role
data "discord_color" "blue" {
  hex = "#3498db"
}

output "channel_id" {
  value = discord_channel.general.id
}
```

For more examples, see the [`examples/`](examples/) directory.

## Data Sources

- [`discord_channel`](docs/data-sources/channel.md) - Retrieves a single Discord channel by its ID
- [`discord_category`](docs/data-sources/category.md) - Retrieves a Discord category channel by ID or name
- [`discord_channels`](docs/data-sources/channels.md) - Retrieves channels from a Discord guild (server)
- [`discord_color`](docs/data-sources/color.md) - Converts hex or RGB color values to decimal integers for Discord role colors
- [`discord_member`](docs/data-sources/member.md) - Retrieves a single Discord member from a guild (server)
- [`discord_members`](docs/data-sources/members.md) - Retrieves all members from a Discord guild (server)
- [`discord_server`](docs/data-sources/server.md) - Retrieves a single Discord server (guild) by its ID
- [`discord_servers`](docs/data-sources/servers.md) - Retrieves a list of Discord servers (guilds) that the bot is a member of
- [`discord_role`](docs/data-sources/role.md) - Retrieves a single Discord role by ID or name
- [`discord_roles`](docs/data-sources/roles.md) - Retrieves all roles from a Discord guild (server)
- [`discord_emoji`](docs/data-sources/emoji.md) - Retrieves a single Discord custom emoji by ID or name
- [`discord_emojis`](docs/data-sources/emojis.md) - Retrieves all custom emojis from a Discord guild (server)

## Resources

- [`discord_channel`](docs/resources/channel.md) - Creates and manages a Discord channel in a guild (server)
- [`discord_category`](docs/resources/category.md) - Creates and manages a Discord category channel
- [`discord_channel_permission`](docs/resources/channel_permission.md) - Creates and manages Discord channel permission overwrites
- [`discord_invite`](docs/resources/invite.md) - Creates and manages Discord invites for channels
- [`discord_webhook`](docs/resources/webhook.md) - Creates and manages Discord webhooks for channels
- [`discord_message`](docs/resources/message.md) - Creates and manages Discord messages in channels
- [`discord_server`](docs/resources/server.md) - Creates and manages a Discord server (guild)
- [`discord_role`](docs/resources/role.md) - Creates and manages a Discord role in a guild (server)
- [`discord_role_member`](docs/resources/role_member.md) - Manages the membership of a user in a Discord role
- [`discord_emoji`](docs/resources/emoji.md) - Creates and manages a Discord custom emoji in a guild (server)
- [`discord_everyone_role`](docs/resources/everyone_role.md) - Manages the @everyone role in a guild (server)

## Local Testing (Development Container)

When developing in the devcontainer, you can test the provider locally using the following steps:

### 1. Build the Provider

Build the provider binary:

```bash
make build
# or
go build -o terraform-provider-discord -buildvcs=false
```

### 2. Install Provider Locally

Install the provider to Terraform's local plugin directory so Terraform can find it:

**Option A: Using Make (Recommended)**

```bash
make install-local
```

**Option B: Manual installation**

```bash
# Create the plugin directory structure
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/tfstack/discord/0.1.0/linux_amd64

# Copy the built binary
cp terraform-provider-discord ~/.terraform.d/plugins/registry.terraform.io/tfstack/discord/0.1.0/linux_amd64/
```

**Note:** The version number (`0.1.0`) should match the version in your Terraform configuration's `required_providers` block.

### 3. Initialize Examples (Automated)

**Option A: Initialize all examples automatically**

```bash
make init-examples
```

This will:

- Build and install the provider locally
- Initialize Terraform in all example directories
- Skip examples that require variables (you'll need to set those manually)

**Option B: Initialize a specific example**

```bash
make init-example EXAMPLE=examples/data-sources/discord_channels
```

**Option C: Manual initialization**

Navigate to the example directory and initialize manually:

```bash
cd examples/data-sources/discord_channels
terraform init
```

### 4. Test with Example Configuration

After initialization, navigate to any example directory and test the provider:

```bash
cd examples/data-sources/discord_channels

# Option 1: Use .env file (recommended - edit .env with your values)
# Copy .env.example to .env and fill in your values, then:
source .env

# Option 2: Set environment variables manually
# Set your Discord bot token
export DISCORD_BOT_TOKEN="your-bot-token-here"

# Set your guild ID (enable Developer Mode in Discord to get this)
export TF_VAR_guild_id="123456789012345678"

# Optionally set category name
export TF_VAR_category_name="General"

# Plan to see what Terraform will do
terraform plan

# Apply to test the provider
terraform apply
```

### 5. Run Unit Tests

Run the unit tests:

```bash
make test
# or
go test -v ./...
```

### 6. Run Test Coverage

Generate a test coverage report:

**Option A: Using Make (Recommended)**

```bash
make test-coverage
```

This will:

- Run tests with coverage
- Display coverage summary in the terminal
- Generate an HTML coverage report (`coverage.html`)

**Option B: Manual commands**

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage report in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# View coverage for specific package
go test -cover ./internal/provider/
```

**Coverage Options:**

- `-coverprofile=coverage.out` - Generate coverage profile file
- `-covermode=count` - Show how many times each statement was executed (default: `set`)
- `-covermode=atomic` - Same as count but thread-safe (useful for parallel tests)
- `-coverpkg=./...` - Include coverage for all packages, not just tested ones

**Example output:**

```
github.com/tfstack/terraform-provider-discord/internal/provider/data_source_channel.go:Metadata    100.0%
github.com/tfstack/terraform-provider-discord/internal/provider/data_source_channel.go:Schema      100.0%
...
total:                                                                    (statements)    85.5%
```

### 7. Run Acceptance Tests

Acceptance tests make real API calls to Discord. Set the `TF_ACC` environment variable to enable them:

```bash
export DISCORD_BOT_TOKEN="your-bot-token-here"
export TF_ACC=1
make testacc
# or
TF_ACC=1 go test -v ./...
```

**Warning:** Acceptance tests create and destroy real resources. Use a test Discord server and bot token.

### 8. Quick Setup Scripts

Helper scripts are available to automate common tasks:

**Install Provider Locally:**

```bash
make install-local
```

**Initialize All Examples:**

```bash
make init-examples
```

**Initialize Specific Example:**

```bash
make init-example EXAMPLE=examples/data-sources/discord_channels
```

### Troubleshooting

- **Provider not found:** Ensure the version in your Terraform config matches the directory version (`0.1.0`)
- **Permission denied:** Make sure the plugin directory is writable: `chmod -R 755 ~/.terraform.d/plugins/`
- **Provider version mismatch:** Update the version in your Terraform config or rename the plugin directory to match
- **Missing Access (403):** Check that your bot has the required permissions and is a member of the server
- **Missing Access (50001):** Enable the `GUILD_MEMBERS` privileged intent in the Discord Developer Portal
- **Connection errors:** Verify your bot token is correct and the bot is online

## Examples

Comprehensive examples are available in the [`examples/`](examples/) directory:

- **Data Sources**: See [`examples/data-sources/`](examples/data-sources/) for examples of querying Discord resources
- **Resources**: See [`examples/resources/`](examples/resources/) for examples of managing Discord resources

Each example includes a `data-source.tf` or `resource.tf` file with working Terraform configuration.

## Limitations

- **Server Creation**: Creating servers (`discord_server` resource) requires a user OAuth2 token, not a bot token. Bot tokens cannot create servers.
- **Channel Types**: Some channel types (news, stage, forum) cannot be created directly by bots due to Discord API limitations. They must be created through the Discord client and then managed via Terraform.
- **Message Editing**: Messages can only be edited by the bot that created them or by users with `MANAGE_MESSAGES` permission.
- **Webhook Tokens**: Webhook tokens are only available at creation time and cannot be retrieved later via the API.
- **Emoji Images**: Emoji images cannot be changed after creation. To change an emoji's image, delete and recreate it.

## Documentation

Full documentation for all data sources and resources is available in the [`docs/`](docs/) directory:

- [Data Sources Documentation](docs/data-sources/)
- [Resources Documentation](docs/resources/)

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for information on developing the provider.

## Support

- **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/tfstack/terraform-provider-discord/issues)
- **Discussions**: Ask questions and share ideas on [GitHub Discussions](https://github.com/tfstack/terraform-provider-discord/discussions)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
