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
var _ datasource.DataSource = &memberDataSource{}

// memberDataSource defines the data source implementation.
type memberDataSource struct {
	client *discordgo.Session
}

// memberDataSourceModel describes the data source data model.
type memberDataSourceModel struct {
	UserID                     types.String `tfsdk:"user_id"`
	GuildID                    types.String `tfsdk:"guild_id"`
	ID                         types.String `tfsdk:"id"`
	Username                   types.String `tfsdk:"username"`
	Discriminator              types.String `tfsdk:"discriminator"`
	GlobalName                 types.String `tfsdk:"global_name"`
	Nickname                   types.String `tfsdk:"nickname"`
	Avatar                     types.String `tfsdk:"avatar"`
	Bot                        types.Bool   `tfsdk:"bot"`
	Roles                      types.List   `tfsdk:"roles"`
	JoinedAt                   types.String `tfsdk:"joined_at"`
	PremiumSince               types.String `tfsdk:"premium_since"`
	Deaf                       types.Bool   `tfsdk:"deaf"`
	Mute                       types.Bool   `tfsdk:"mute"`
	Pending                    types.Bool   `tfsdk:"pending"`
	Permissions                types.Int64  `tfsdk:"permissions"`
	CommunicationDisabledUntil types.String `tfsdk:"communication_disabled_until"`
}

// NewMemberDataSource is a helper function to simplify testing.
func NewMemberDataSource() datasource.DataSource {
	return &memberDataSource{}
}

// Metadata returns the data source type name.
func (d *memberDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_member"
}

// Schema defines the schema for the data source.
func (d *memberDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a Discord member from a guild (server).",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Description: "The ID of the user (member) to retrieve.",
				Required:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) where the member is located.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the member (same as user_id).",
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username of the member.",
				Computed:    true,
			},
			"discriminator": schema.StringAttribute{
				Description: "The discriminator of the member (4-digit number after #).",
				Computed:    true,
			},
			"global_name": schema.StringAttribute{
				Description: "The global display name of the member.",
				Computed:    true,
			},
			"nickname": schema.StringAttribute{
				Description: "The guild-specific nickname of the member.",
				Computed:    true,
			},
			"avatar": schema.StringAttribute{
				Description: "The avatar hash of the member.",
				Computed:    true,
			},
			"bot": schema.BoolAttribute{
				Description: "Whether the member is a bot.",
				Computed:    true,
			},
			"roles": schema.ListAttribute{
				Description: "List of role IDs the member has.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"joined_at": schema.StringAttribute{
				Description: "When the member joined the guild (ISO 8601 timestamp).",
				Computed:    true,
			},
			"premium_since": schema.StringAttribute{
				Description: "When the member started boosting the guild (ISO 8601 timestamp).",
				Computed:    true,
			},
			"deaf": schema.BoolAttribute{
				Description: "Whether the member is deafened in voice channels.",
				Computed:    true,
			},
			"mute": schema.BoolAttribute{
				Description: "Whether the member is muted in voice channels.",
				Computed:    true,
			},
			"pending": schema.BoolAttribute{
				Description: "Whether the member has not yet passed the guild's Membership Screening requirements.",
				Computed:    true,
			},
			"permissions": schema.Int64Attribute{
				Description: "Total permissions of the member in the channel (only present when fetched via channel endpoint).",
				Computed:    true,
			},
			"communication_disabled_until": schema.StringAttribute{
				Description: "When the member's timeout will expire (ISO 8601 timestamp).",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *memberDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *memberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data memberDataSourceModel

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

	userID := data.UserID.ValueString()
	guildID := data.GuildID.ValueString()

	if userID == "" {
		resp.Diagnostics.AddError(
			"Missing User ID",
			"The user_id attribute is required.",
		)
		return
	}

	if guildID == "" {
		resp.Diagnostics.AddError(
			"Missing Guild ID",
			"The guild_id attribute is required.",
		)
		return
	}

	// Fetch the member from the guild
	member, err := d.client.GuildMember(guildID, userID)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to fetch member %s from guild %s: %s", userID, guildID, err.Error())

		// Check for common permission/intent errors
		if err.Error() == "HTTP 403 Forbidden, {\"message\": \"Missing Access\", \"code\": 50001}" ||
			err.Error() == "HTTP 403 Forbidden" {
			errorMsg += "\n\nThis error typically indicates that the bot is missing the GUILD_MEMBERS privileged intent." +
				"\n\nTo fix this:" +
				"\n1. Go to https://discord.com/developers/applications" +
				"\n2. Select your bot application" +
				"\n3. Go to the 'Bot' section" +
				"\n4. Scroll down to 'Privileged Gateway Intents'" +
				"\n5. Enable 'SERVER MEMBERS INTENT' (GUILD_MEMBERS)" +
				"\n6. Save changes" +
				"\n7. Restart your bot/application" +
				"\n\nNote: This is a Discord API requirement, not just a permission issue. Even with Administrator permissions," +
				" the privileged intent must be enabled in the Developer Portal."
		}

		resp.Diagnostics.AddError(
			"Error Fetching Member",
			errorMsg,
		)
		return
	}

	// Validate member has User data
	if member.User == nil {
		resp.Diagnostics.AddError(
			"Invalid Member Data",
			fmt.Sprintf("Member %s in guild %s has no User data. This may indicate a Discord API issue.", userID, guildID),
		)
		return
	}

	// Populate the model with member data
	data.ID = types.StringValue(member.User.ID)
	data.Username = types.StringValue(member.User.Username)

	// Discriminator (may be empty for new users)
	if member.User.Discriminator != "" {
		data.Discriminator = types.StringValue(member.User.Discriminator)
	} else {
		data.Discriminator = types.StringNull()
	}

	// Global name (display name)
	if member.User.GlobalName != "" {
		data.GlobalName = types.StringValue(member.User.GlobalName)
	} else {
		data.GlobalName = types.StringNull()
	}

	// Nickname (guild-specific)
	if member.Nick != "" {
		data.Nickname = types.StringValue(member.Nick)
	} else {
		data.Nickname = types.StringNull()
	}

	// Avatar
	if member.Avatar != "" {
		data.Avatar = types.StringValue(member.Avatar)
	} else {
		data.Avatar = types.StringNull()
	}

	data.Bot = types.BoolValue(member.User.Bot)

	// Convert roles slice to Terraform list
	rolesList := make([]attr.Value, 0, len(member.Roles))
	for _, roleID := range member.Roles {
		rolesList = append(rolesList, types.StringValue(roleID))
	}

	rolesValue, diags := types.ListValue(types.StringType, rolesList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Roles = rolesValue

	// Joined at timestamp
	if !member.JoinedAt.IsZero() {
		data.JoinedAt = types.StringValue(member.JoinedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.JoinedAt = types.StringNull()
	}

	// Premium since timestamp
	// PremiumSince is a *time.Time (pointer), so check for nil first
	if member.PremiumSince != nil && !member.PremiumSince.IsZero() {
		data.PremiumSince = types.StringValue(member.PremiumSince.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.PremiumSince = types.StringNull()
	}

	data.Deaf = types.BoolValue(member.Deaf)
	data.Mute = types.BoolValue(member.Mute)
	data.Pending = types.BoolValue(member.Pending)

	// Permissions (may not always be present)
	if member.Permissions != 0 {
		data.Permissions = types.Int64Value(member.Permissions)
	} else {
		data.Permissions = types.Int64Null()
	}

	// Communication disabled until (timeout)
	// CommunicationDisabledUntil is a *time.Time (pointer), so check for nil first
	if member.CommunicationDisabledUntil != nil && !member.CommunicationDisabledUntil.IsZero() {
		data.CommunicationDisabledUntil = types.StringValue(member.CommunicationDisabledUntil.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.CommunicationDisabledUntil = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
