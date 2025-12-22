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
var _ datasource.DataSource = &channelDataSource{}

// channelDataSource defines the data source implementation.
type channelDataSource struct {
	client *discordgo.Session
}

// channelDataSourceModel describes the data source data model.
type channelDataSourceModel struct {
	ChannelID  types.String `tfsdk:"channel_id"`
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Type       types.Int64  `tfsdk:"type"`
	CategoryID types.String `tfsdk:"category_id"`
	Position   types.Int64  `tfsdk:"position"`
	GuildID    types.String `tfsdk:"guild_id"`
}

// NewChannelDataSource is a helper function to simplify testing.
func NewChannelDataSource() datasource.DataSource {
	return &channelDataSource{}
}

// Metadata returns the data source type name.
func (d *channelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel"
}

// Schema defines the schema for the data source.
func (d *channelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a single Discord channel by its ID.",
		Attributes: map[string]schema.Attribute{
			"channel_id": schema.StringAttribute{
				Description: "The ID of the Discord channel to retrieve.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the channel.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the channel.",
				Computed:    true,
			},
			"type": schema.Int64Attribute{
				Description: "The type of the channel. 0 = text channel, 2 = voice channel, 4 = category channel, etc.",
				Computed:    true,
			},
			"category_id": schema.StringAttribute{
				Description: "The ID of the parent category channel, if this channel belongs to a category.",
				Computed:    true,
			},
			"position": schema.Int64Attribute{
				Description: "The position of the channel in the channel list.",
				Computed:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) this channel belongs to.",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *channelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *channelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data channelDataSourceModel

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

	channelID := data.ChannelID.ValueString()
	if channelID == "" {
		resp.Diagnostics.AddError(
			"Missing Channel ID",
			"The channel_id attribute is required.",
		)
		return
	}

	// Fetch the channel by ID
	channel, err := d.client.Channel(channelID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Channel",
			fmt.Sprintf("Unable to fetch channel %s: %s", channelID, err.Error()),
		)
		return
	}

	// Populate the model with channel data
	data.ID = types.StringValue(channel.ID)
	data.Name = types.StringValue(channel.Name)
	data.Type = types.Int64Value(int64(channel.Type))
	data.Position = types.Int64Value(int64(channel.Position))
	data.GuildID = types.StringValue(channel.GuildID)

	if channel.ParentID != "" {
		data.CategoryID = types.StringValue(channel.ParentID)
	} else {
		data.CategoryID = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
