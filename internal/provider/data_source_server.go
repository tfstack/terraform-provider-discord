package provider

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the data source type implements the required interfaces.
var _ datasource.DataSource = &serverDataSource{}

// serverDataSource defines the data source implementation.
type serverDataSource struct {
	client *discordgo.Session
}

// serverDataSourceModel describes the data source data model.
type serverDataSourceModel struct {
	ServerID    types.String `tfsdk:"server_id"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Icon        types.String `tfsdk:"icon"`
	Owner       types.Bool   `tfsdk:"owner"`
	Permissions types.Int64  `tfsdk:"permissions"`
	Features    types.List   `tfsdk:"features"`
}

// NewServerDataSource is a helper function to simplify testing.
func NewServerDataSource() datasource.DataSource {
	return &serverDataSource{}
}

// Metadata returns the data source type name.
func (d *serverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

// Schema defines the schema for the data source.
func (d *serverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a single Discord server (guild) by its ID. The bot must be a member of the server.",
		Attributes: map[string]schema.Attribute{
			"server_id": schema.StringAttribute{
				Description: "The ID of the Discord server (guild) to retrieve.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the server (guild).",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the server (guild).",
				Computed:    true,
			},
			"icon": schema.StringAttribute{
				Description: "The icon hash of the server (guild). Empty string if no icon is set.",
				Computed:    true,
			},
			"owner": schema.BoolAttribute{
				Description: "Whether the bot is the owner of the server (guild).",
				Computed:    true,
			},
			"permissions": schema.Int64Attribute{
				Description: "The permissions integer for the bot in the server (guild).",
				Computed:    true,
			},
			"features": schema.ListAttribute{
				Description: "List of features enabled for the server (guild).",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *serverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*discordgo.Session)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *discordgo.Session, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure client is configured
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Discord Client Not Configured",
			"The Discord client was not properly configured. This is a provider error.",
		)
		return
	}

	serverID := data.ServerID.ValueString()
	if serverID == "" {
		resp.Diagnostics.AddError(
			"Missing Server ID",
			"The server_id attribute is required.",
		)
		return
	}

	// Fetch the guild (server) by ID
	// Note: Guild() requires the bot to be a member of the server
	guild, err := d.client.Guild(serverID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Server",
			fmt.Sprintf("Unable to fetch server %s: %s", serverID, err.Error()),
		)
		return
	}

	// Populate the model with server data
	data.ID = types.StringValue(guild.ID)
	data.Name = types.StringValue(guild.Name)
	data.Owner = types.BoolValue(guild.Owner)
	data.Permissions = types.Int64Value(guild.Permissions)

	// Handle icon (can be empty string)
	if guild.Icon != "" {
		data.Icon = types.StringValue(guild.Icon)
	} else {
		data.Icon = types.StringNull()
	}

	// Convert features slice to Terraform list
	featuresList := make([]types.String, 0, len(guild.Features))
	for _, feature := range guild.Features {
		featuresList = append(featuresList, types.StringValue(string(feature)))
	}

	featuresValue, diags := types.ListValueFrom(ctx, types.StringType, featuresList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Features = featuresValue

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
