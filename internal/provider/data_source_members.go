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
var _ datasource.DataSource = &membersDataSource{}

// membersDataSource defines the data source implementation.
type membersDataSource struct {
	client *discordgo.Session
}

// membersDataSourceModel describes the data source data model.
type membersDataSourceModel struct {
	GuildID types.String `tfsdk:"guild_id"`
	Members types.List   `tfsdk:"members"`
}

// memberModel describes a single member in the data source.
type memberModel struct {
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

// NewMembersDataSource is a helper function to simplify testing.
func NewMembersDataSource() datasource.DataSource {
	return &membersDataSource{}
}

// Metadata returns the data source type name.
func (d *membersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_members"
}

// Schema defines the schema for the data source.
func (d *membersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves all members from a Discord guild (server). Note: This may take time for large servers as it fetches members in batches.",
		Attributes: map[string]schema.Attribute{
			"guild_id": schema.StringAttribute{
				Description: "The ID of the Discord guild (server).",
				Required:    true,
			},
			"members": schema.ListNestedAttribute{
				Description: "List of members in the guild.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the member (user ID).",
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
				},
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *membersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *membersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data membersDataSourceModel

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

	// Fetch all members for the guild
	// Note: GuildMembers() fetches members in batches and may take time for large servers
	// The limit parameter controls how many members to fetch per request (max 1000)
	// After parameter is used for pagination
	members, err := d.client.GuildMembers(guildID, "", 1000)
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to fetch members for guild %s: %s", guildID, err.Error())

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
			"Error Fetching Members",
			errorMsg,
		)
		return
	}

	// Convert Discord members to Terraform model
	memberList := make([]memberModel, 0, len(members))
	for _, member := range members {
		// Skip members with nil User (shouldn't happen, but be defensive)
		if member.User == nil {
			resp.Diagnostics.AddWarning(
				"Skipping Member with Nil User",
				fmt.Sprintf("Skipping a member in guild %s because the User field is nil. This may indicate a Discord API issue.", guildID),
			)
			continue
		}

		memberModel := memberModel{
			ID:       types.StringValue(member.User.ID),
			Username: types.StringValue(member.User.Username),
			Bot:      types.BoolValue(member.User.Bot),
			Deaf:     types.BoolValue(member.Deaf),
			Mute:     types.BoolValue(member.Mute),
			Pending:  types.BoolValue(member.Pending),
		}

		// Discriminator (may be empty for new users)
		if member.User.Discriminator != "" {
			memberModel.Discriminator = types.StringValue(member.User.Discriminator)
		} else {
			memberModel.Discriminator = types.StringNull()
		}

		// Global name (display name)
		if member.User.GlobalName != "" {
			memberModel.GlobalName = types.StringValue(member.User.GlobalName)
		} else {
			memberModel.GlobalName = types.StringNull()
		}

		// Nickname (guild-specific)
		if member.Nick != "" {
			memberModel.Nickname = types.StringValue(member.Nick)
		} else {
			memberModel.Nickname = types.StringNull()
		}

		// Avatar
		if member.Avatar != "" {
			memberModel.Avatar = types.StringValue(member.Avatar)
		} else {
			memberModel.Avatar = types.StringNull()
		}

		// Convert roles slice to Terraform list
		rolesList := make([]attr.Value, 0, len(member.Roles))
		for _, roleID := range member.Roles {
			rolesList = append(rolesList, types.StringValue(roleID))
		}

		rolesValue, diags := types.ListValue(types.StringType, rolesList)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			continue
		}
		memberModel.Roles = rolesValue

		// Joined at timestamp
		// JoinedAt is a time.Time (not a pointer), so we can safely check IsZero()
		if !member.JoinedAt.IsZero() {
			memberModel.JoinedAt = types.StringValue(member.JoinedAt.Format("2006-01-02T15:04:05Z07:00"))
		} else {
			memberModel.JoinedAt = types.StringNull()
		}

		// Premium since timestamp
		// PremiumSince is a *time.Time (pointer), so check for nil first
		if member.PremiumSince != nil && !member.PremiumSince.IsZero() {
			memberModel.PremiumSince = types.StringValue(member.PremiumSince.Format("2006-01-02T15:04:05Z07:00"))
		} else {
			memberModel.PremiumSince = types.StringNull()
		}

		// Permissions (may not always be present)
		if member.Permissions != 0 {
			memberModel.Permissions = types.Int64Value(member.Permissions)
		} else {
			memberModel.Permissions = types.Int64Null()
		}

		// Communication disabled until (timeout)
		if member.CommunicationDisabledUntil != nil && !member.CommunicationDisabledUntil.IsZero() {
			memberModel.CommunicationDisabledUntil = types.StringValue(member.CommunicationDisabledUntil.Format("2006-01-02T15:04:05Z07:00"))
		} else {
			memberModel.CommunicationDisabledUntil = types.StringNull()
		}

		memberList = append(memberList, memberModel)
	}

	// Convert member list to Terraform list
	memberObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":                           types.StringType,
			"username":                     types.StringType,
			"discriminator":                types.StringType,
			"global_name":                  types.StringType,
			"nickname":                     types.StringType,
			"avatar":                       types.StringType,
			"bot":                          types.BoolType,
			"roles":                        types.ListType{ElemType: types.StringType},
			"joined_at":                    types.StringType,
			"premium_since":                types.StringType,
			"deaf":                         types.BoolType,
			"mute":                         types.BoolType,
			"pending":                      types.BoolType,
			"permissions":                  types.Int64Type,
			"communication_disabled_until": types.StringType,
		},
	}

	memberObjects := make([]attr.Value, 0, len(memberList))
	for _, m := range memberList {
		obj, diags := types.ObjectValue(memberObjectType.AttrTypes, map[string]attr.Value{
			"id":                           m.ID,
			"username":                     m.Username,
			"discriminator":                m.Discriminator,
			"global_name":                  m.GlobalName,
			"nickname":                     m.Nickname,
			"avatar":                       m.Avatar,
			"bot":                          m.Bot,
			"roles":                        m.Roles,
			"joined_at":                    m.JoinedAt,
			"premium_since":                m.PremiumSince,
			"deaf":                         m.Deaf,
			"mute":                         m.Mute,
			"pending":                      m.Pending,
			"permissions":                  m.Permissions,
			"communication_disabled_until": m.CommunicationDisabledUntil,
		})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			continue
		}
		memberObjects = append(memberObjects, obj)
	}

	membersValue, diags := types.ListValue(memberObjectType, memberObjects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Members = membersValue

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
