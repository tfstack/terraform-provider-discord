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
var _ datasource.DataSource = &channelsDataSource{}

// channelsDataSource defines the data source implementation.
type channelsDataSource struct {
	client *discordgo.Session
}

// channelsDataSourceModel describes the data source data model.
type channelsDataSourceModel struct {
	GuildID      types.String `tfsdk:"guild_id"`
	CategoryName types.String `tfsdk:"category_name"`
	Channels     types.List   `tfsdk:"channels"`
}

// channelModel describes a single channel in the data source.
type channelModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Type       types.Int64  `tfsdk:"type"`
	CategoryID types.String `tfsdk:"category_id"`
	Position   types.Int64  `tfsdk:"position"`
}

// NewChannelsDataSource is a helper function to simplify testing.
func NewChannelsDataSource() datasource.DataSource {
	return &channelsDataSource{}
}

// Metadata returns the data source type name.
func (d *channelsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channels"
}

// Schema defines the schema for the data source.
func (d *channelsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves channels from a Discord guild (server). Optionally filters channels by category name.",
		Attributes: map[string]schema.Attribute{
			"guild_id": schema.StringAttribute{
				Description: "The ID of the Discord guild (server).",
				Required:    true,
			},
			"category_name": schema.StringAttribute{
				Description: "Optional: Filter channels by category name. If provided, only channels within the specified category will be returned.",
				Optional:    true,
			},
			"channels": schema.ListNestedAttribute{
				Description: "List of channels in the guild.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
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
					},
				},
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *channelsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *channelsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data channelsDataSourceModel

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

	guildID := data.GuildID.ValueString()
	if guildID == "" {
		resp.Diagnostics.AddError(
			"Missing Guild ID",
			"The guild_id attribute is required.",
		)
		return
	}

	// Fetch all channels for the guild
	channels, err := d.client.GuildChannels(guildID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Channels",
			fmt.Sprintf("Unable to fetch channels for guild %s: %s", guildID, err.Error()),
		)
		return
	}

	// Filter by category if category_name is provided
	var filteredChannels []*discordgo.Channel
	var categoryID string

	if !data.CategoryName.IsNull() && !data.CategoryName.IsUnknown() {
		categoryName := data.CategoryName.ValueString()

		// Find the category channel by name
		for _, channel := range channels {
			if channel.Type == discordgo.ChannelTypeGuildCategory && channel.Name == categoryName {
				categoryID = channel.ID
				break
			}
		}

		if categoryID == "" {
			resp.Diagnostics.AddError(
				"Category Not Found",
				fmt.Sprintf("No category channel found with name '%s' in guild %s", categoryName, guildID),
			)
			return
		}

		// Filter channels that belong to this category
		for _, channel := range channels {
			if channel.ParentID == categoryID {
				filteredChannels = append(filteredChannels, channel)
			}
		}
	} else {
		// Return all channels if no category filter is specified
		filteredChannels = channels
	}

	// Convert Discord channels to Terraform model
	channelList := make([]channelModel, 0, len(filteredChannels))
	for _, channel := range filteredChannels {
		channelModel := channelModel{
			ID:       types.StringValue(channel.ID),
			Name:     types.StringValue(channel.Name),
			Type:     types.Int64Value(int64(channel.Type)),
			Position: types.Int64Value(int64(channel.Position)),
		}

		if channel.ParentID != "" {
			channelModel.CategoryID = types.StringValue(channel.ParentID)
		} else {
			channelModel.CategoryID = types.StringNull()
		}

		channelList = append(channelList, channelModel)
	}

	// Convert to Terraform list
	channelListValue, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":          types.StringType,
			"name":        types.StringType,
			"type":        types.Int64Type,
			"category_id": types.StringType,
			"position":    types.Int64Type,
		},
	}, channelList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Channels = channelListValue

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
