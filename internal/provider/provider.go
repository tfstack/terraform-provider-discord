package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure discordProvider satisfies various provider interfaces.
var _ provider.Provider = &discordProvider{}

// discordProvider defines the provider implementation.
type discordProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// discordProviderModel describes the provider data model.
type discordProviderModel struct {
	Token types.String `tfsdk:"token"`
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &discordProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *discordProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "discord"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *discordProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Discord provider allows you to manage Discord servers, channels, roles, members, and other resources using Terraform. " +
			"Authentication is done via a Discord bot token, which can be provided in the provider configuration block or via the DISCORD_BOT_TOKEN environment variable. " +
			"To get a bot token, create a bot application in the Discord Developer Portal and copy the token from the Bot section.",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Description: "Discord bot token for authentication. " +
					"This token is required to authenticate with the Discord API. " +
					"You can obtain a bot token from the Discord Developer Portal (https://discord.com/developers/applications). " +
					"Alternatively, you can set the DISCORD_BOT_TOKEN environment variable instead of providing it here. " +
					"This attribute is sensitive and will not be displayed in logs or output.",
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

// Configure prepares a Discord API client for data sources and resources.
func (p *discordProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config discordProviderModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get token from config or environment variable
	token := config.Token.ValueString()
	if token == "" {
		token = os.Getenv("DISCORD_BOT_TOKEN")
	}

	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Discord Bot Token",
			"The provider cannot create the Discord API client as there is no token configured. "+
				"Set the token value in the provider configuration block or set the DISCORD_BOT_TOKEN environment variable.",
		)
		return
	}

	// Create Discord session
	// Discord bot tokens should be prefixed with "Bot "
	tokenPrefix := "Bot "
	if len(token) > 4 && token[:4] == "Bot " {
		tokenPrefix = ""
	}

	dg, err := discordgo.New(tokenPrefix + token)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Discord API Client",
			fmt.Sprintf("An unexpected error occurred when creating the Discord API client: %s", err.Error()),
		)
		return
	}

	// Open the websocket and begin listening
	err = dg.Open()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Open Discord Connection",
			fmt.Sprintf("An unexpected error occurred when opening the Discord connection: %s", err.Error()),
		)
		return
	}

	// Make the Discord session available during DataSource and Resource
	// Configure methods.
	resp.ResourceData = dg
	resp.DataSourceData = dg
}

// Resources defines the resources implemented by the provider.
func (p *discordProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewChannelResource,
		NewCategoryResource,
		NewChannelPermissionResource,
		NewServerResource,
		NewRoleResource,
		NewEveryoneRoleResource,
		NewInviteResource,
		NewWebhookResource,
		NewMessageResource,
		NewRoleMemberResource,
		NewEmojiResource,
	}
}

// DataSources defines the data sources implemented by the provider.
func (p *discordProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewChannelsDataSource,
		NewChannelDataSource,
		NewCategoryDataSource,
		NewServersDataSource,
		NewServerDataSource,
		NewRolesDataSource,
		NewRoleDataSource,
		NewColorDataSource,
		NewMemberDataSource,
		NewMembersDataSource,
		NewEmojisDataSource,
		NewEmojiDataSource,
	}
}
