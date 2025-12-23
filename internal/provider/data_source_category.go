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
var _ datasource.DataSource = &categoryDataSource{}

// categoryDataSource defines the data source implementation.
type categoryDataSource struct {
	client *discordgo.Session
}

// categoryDataSourceModel describes the data source data model.
type categoryDataSourceModel struct {
	CategoryID types.String `tfsdk:"category_id"`
	Name       types.String `tfsdk:"name"`
	GuildID    types.String `tfsdk:"guild_id"`
	ID         types.String `tfsdk:"id"`
	Position   types.Int64  `tfsdk:"position"`
}

// NewCategoryDataSource is a helper function to simplify testing.
func NewCategoryDataSource() datasource.DataSource {
	return &categoryDataSource{}
}

// Metadata returns the data source type name.
func (d *categoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_category"
}

// Schema defines the schema for the data source.
func (d *categoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a Discord category channel. Can be looked up by ID or by name and guild ID.",
		Attributes: map[string]schema.Attribute{
			"category_id": schema.StringAttribute{
				Description: "The ID of the category channel to retrieve. Either this or (name + guild_id) must be provided.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the category channel. Must be provided along with guild_id if category_id is not specified.",
				Optional:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) where the category is located. Must be provided along with name if category_id is not specified.",
				Optional:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the category channel.",
				Computed:    true,
			},
			"position": schema.Int64Attribute{
				Description: "The position of the category in the channel list.",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *categoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *categoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data categoryDataSourceModel

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

	var channel *discordgo.Channel
	var err error

	categoryID := data.CategoryID.ValueString()
	name := data.Name.ValueString()
	guildID := data.GuildID.ValueString()

	// Determine lookup method
	if categoryID != "" {
		// Lookup by ID
		channel, err = d.client.Channel(categoryID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Fetching Category",
				fmt.Sprintf("Unable to fetch category %s: %s", categoryID, err.Error()),
			)
			return
		}

		// Verify it's actually a category channel
		if channel.Type != discordgo.ChannelTypeGuildCategory {
			resp.Diagnostics.AddError(
				"Invalid Channel Type",
				fmt.Sprintf("Channel %s is not a category channel (type: %d)", categoryID, channel.Type),
			)
			return
		}
	} else if name != "" && guildID != "" {
		// Lookup by name and guild ID
		channels, err := d.client.GuildChannels(guildID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Fetching Guild Channels",
				fmt.Sprintf("Unable to fetch channels from guild %s: %s", guildID, err.Error()),
			)
			return
		}

		// Find the category channel by name
		found := false
		for _, ch := range channels {
			if ch.Type == discordgo.ChannelTypeGuildCategory && ch.Name == name {
				channel = ch
				found = true
				break
			}
		}

		if !found {
			resp.Diagnostics.AddError(
				"Category Not Found",
				fmt.Sprintf("No category channel found with name '%s' in guild %s", name, guildID),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Missing Required Attributes",
			"Either category_id must be provided, or both name and guild_id must be provided.",
		)
		return
	}

	// Populate the model with category data
	data.ID = types.StringValue(channel.ID)
	data.Name = types.StringValue(channel.Name)
	data.GuildID = types.StringValue(channel.GuildID)
	data.Position = types.Int64Value(int64(channel.Position))
	data.CategoryID = types.StringValue(channel.ID) // For consistency, set category_id to the ID

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
