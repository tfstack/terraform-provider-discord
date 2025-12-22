package provider

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the data source type implements the required interfaces.
var _ datasource.DataSource = &serversDataSource{}

// serversDataSource defines the data source implementation.
type serversDataSource struct {
	client *discordgo.Session
}

// serversDataSourceModel describes the data source data model.
type serversDataSourceModel struct {
	Servers types.List `tfsdk:"servers"`
}

// serverModel describes a single server (guild) in the data source.
type serverModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Icon        types.String `tfsdk:"icon"`
	Owner       types.Bool   `tfsdk:"owner"`
	Permissions types.Int64  `tfsdk:"permissions"`
	Features    types.List   `tfsdk:"features"`
}

// NewServersDataSource is a helper function to simplify testing.
func NewServersDataSource() datasource.DataSource {
	return &serversDataSource{}
}

// Metadata returns the data source type name.
func (d *serversDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_servers"
}

// Schema defines the schema for the data source.
func (d *serversDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of Discord servers (guilds) that the bot is a member of.",
		Attributes: map[string]schema.Attribute{
			"servers": schema.ListNestedAttribute{
				Description: "List of servers (guilds) the bot is a member of.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
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
				},
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *serversDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *serversDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serversDataSourceModel

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

	// Fetch all guilds (servers) the bot is a member of
	// Using limit 200 (max) to get as many guilds as possible
	// Empty strings for beforeID and afterID to get all guilds
	// withCounts = false to avoid extra API calls
	guilds, err := d.client.UserGuilds(200, "", "", false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Servers",
			fmt.Sprintf("Unable to fetch servers: %s", err.Error()),
		)
		return
	}

	// Convert Discord guilds to Terraform model
	serverList := make([]serverModel, 0, len(guilds))
	for _, guild := range guilds {
		serverModel := serverModel{
			ID:          types.StringValue(guild.ID),
			Name:        types.StringValue(guild.Name),
			Owner:       types.BoolValue(guild.Owner),
			Permissions: types.Int64Value(guild.Permissions),
		}

		// Handle icon (can be empty string)
		if guild.Icon != "" {
			serverModel.Icon = types.StringValue(guild.Icon)
		} else {
			serverModel.Icon = types.StringNull()
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
		serverModel.Features = featuresValue

		serverList = append(serverList, serverModel)
	}

	// Convert to Terraform list
	serverListValue, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":          types.StringType,
			"name":        types.StringType,
			"icon":        types.StringType,
			"owner":       types.BoolType,
			"permissions": types.Int64Type,
			"features":    types.ListType{ElemType: types.StringType},
		},
	}, serverList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Servers = serverListValue

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
