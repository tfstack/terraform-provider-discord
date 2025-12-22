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
var _ datasource.DataSource = &emojiDataSource{}

// emojiDataSource defines the data source implementation.
type emojiDataSource struct {
	client *discordgo.Session
}

// emojiDataSourceModel describes the data source data model.
type emojiDataSourceModel struct {
	EmojiID       types.String `tfsdk:"emoji_id"`
	Name          types.String `tfsdk:"name"`
	GuildID       types.String `tfsdk:"guild_id"`
	ID            types.String `tfsdk:"id"`
	Animated      types.Bool   `tfsdk:"animated"`
	Managed       types.Bool   `tfsdk:"managed"`
	RequireColons types.Bool   `tfsdk:"require_colons"`
	Roles         types.List   `tfsdk:"roles"`
	User          types.String `tfsdk:"user"`
	Available     types.Bool   `tfsdk:"available"`
}

// NewEmojiDataSource is a helper function to simplify testing.
func NewEmojiDataSource() datasource.DataSource {
	return &emojiDataSource{}
}

// Metadata returns the data source type name.
func (d *emojiDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_emoji"
}

// Schema defines the schema for the data source.
func (d *emojiDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a Discord custom emoji. Can be looked up by ID or by name and guild ID.",
		Attributes: map[string]schema.Attribute{
			"emoji_id": schema.StringAttribute{
				Description: "The ID of the emoji to retrieve. Either this or (name + guild_id) must be provided.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the emoji. Must be provided along with guild_id if emoji_id is not specified.",
				Optional:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) where the emoji is located. Must be provided along with name if emoji_id is not specified.",
				Optional:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the emoji.",
				Computed:    true,
			},
			"animated": schema.BoolAttribute{
				Description: "Whether the emoji is animated.",
				Computed:    true,
			},
			"managed": schema.BoolAttribute{
				Description: "Whether the emoji is managed by an integration.",
				Computed:    true,
			},
			"require_colons": schema.BoolAttribute{
				Description: "Whether the emoji requires colons to be used.",
				Computed:    true,
			},
			"roles": schema.ListAttribute{
				Description: "List of role IDs that can use this emoji. Empty if available to all roles.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"user": schema.StringAttribute{
				Description: "The ID of the user who created the emoji.",
				Computed:    true,
			},
			"available": schema.BoolAttribute{
				Description: "Whether the emoji is available for use.",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *emojiDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *emojiDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data emojiDataSourceModel

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

	emojiID := data.EmojiID.ValueString()
	name := data.Name.ValueString()
	guildID := data.GuildID.ValueString()

	// Validate that either emoji_id or (name + guild_id) is provided
	if emojiID == "" && (name == "" || guildID == "") {
		resp.Diagnostics.AddError(
			"Missing Required Attributes",
			"Either emoji_id or both name and guild_id must be provided.",
		)
		return
	}

	var emoji *discordgo.Emoji

	if emojiID != "" {
		// Look up by emoji ID - we need guild_id to fetch it
		if guildID == "" {
			resp.Diagnostics.AddError(
				"Missing Guild ID",
				"guild_id is required when using emoji_id.",
			)
			return
		}

		fetchedEmoji, err := d.client.GuildEmoji(guildID, emojiID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Fetching Emoji",
				fmt.Sprintf("Unable to fetch emoji %s from guild %s: %s", emojiID, guildID, err.Error()),
			)
			return
		}
		emoji = fetchedEmoji
	} else {
		// Look up by name - fetch all emojis and find matching name
		emojis, err := d.client.GuildEmojis(guildID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Fetching Emojis",
				fmt.Sprintf("Unable to fetch emojis for guild %s: %s", guildID, err.Error()),
			)
			return
		}

		// Find emoji by name
		found := false
		for _, e := range emojis {
			if e.Name == name {
				emoji = e
				found = true
				break
			}
		}

		if !found {
			resp.Diagnostics.AddError(
				"Emoji Not Found",
				fmt.Sprintf("Emoji with name '%s' was not found in guild %s.", name, guildID),
			)
			return
		}
	}

	// Populate the model with emoji data
	data.ID = types.StringValue(emoji.ID)
	data.EmojiID = types.StringValue(emoji.ID)
	data.Name = types.StringValue(emoji.Name)
	data.GuildID = types.StringValue(guildID)
	data.Animated = types.BoolValue(emoji.Animated)
	data.Managed = types.BoolValue(emoji.Managed)
	data.RequireColons = types.BoolValue(emoji.RequireColons)
	data.Available = types.BoolValue(emoji.Available)

	// Convert roles slice to list
	if len(emoji.Roles) > 0 {
		roleList := make([]attr.Value, 0, len(emoji.Roles))
		for _, roleID := range emoji.Roles {
			roleList = append(roleList, types.StringValue(roleID))
		}
		data.Roles = types.ListValueMust(types.StringType, roleList)
	} else {
		data.Roles = types.ListNull(types.StringType)
	}

	// User (creator)
	if emoji.User != nil {
		data.User = types.StringValue(emoji.User.ID)
	} else {
		data.User = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
