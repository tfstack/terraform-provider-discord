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
var _ datasource.DataSource = &roleDataSource{}

// roleDataSource defines the data source implementation.
type roleDataSource struct {
	client *discordgo.Session
}

// roleDataSourceModel describes the data source data model.
type roleDataSourceModel struct {
	RoleID      types.String `tfsdk:"role_id"`
	Name        types.String `tfsdk:"name"`
	GuildID     types.String `tfsdk:"guild_id"`
	ID          types.String `tfsdk:"id"`
	Color       types.Int64  `tfsdk:"color"`
	Position    types.Int64  `tfsdk:"position"`
	Permissions types.Int64  `tfsdk:"permissions"`
	Managed     types.Bool   `tfsdk:"managed"`
	Mentionable types.Bool   `tfsdk:"mentionable"`
	Hoist       types.Bool   `tfsdk:"hoist"`
}

// NewRoleDataSource is a helper function to simplify testing.
func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

// Metadata returns the data source type name.
func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the schema for the data source.
func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a Discord role. Can be looked up by ID or by name and guild ID.",
		Attributes: map[string]schema.Attribute{
			"role_id": schema.StringAttribute{
				Description: "The ID of the role to retrieve. Either this or (name + guild_id) must be provided.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the role. Must be provided along with guild_id if role_id is not specified.",
				Optional:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) where the role is located. Must be provided along with name if role_id is not specified.",
				Optional:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the role.",
				Computed:    true,
			},
			"color": schema.Int64Attribute{
				Description: "The hex color of the role (as an integer).",
				Computed:    true,
			},
			"position": schema.Int64Attribute{
				Description: "The position of the role in the guild's role hierarchy.",
				Computed:    true,
			},
			"permissions": schema.Int64Attribute{
				Description: "The permissions integer for the role on the guild.",
				Computed:    true,
			},
			"managed": schema.BoolAttribute{
				Description: "Whether this role is managed by an integration.",
				Computed:    true,
			},
			"mentionable": schema.BoolAttribute{
				Description: "Whether this role is mentionable.",
				Computed:    true,
			},
			"hoist": schema.BoolAttribute{
				Description: "Whether this role is hoisted (shows up separately in member list).",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data roleDataSourceModel

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

	roleID := data.RoleID.ValueString()
	name := data.Name.ValueString()
	guildID := data.GuildID.ValueString()

	// Validate that either role_id or (name + guild_id) is provided
	if roleID == "" && (name == "" || guildID == "") {
		resp.Diagnostics.AddError(
			"Missing Required Attributes",
			"Either role_id or both name and guild_id must be provided.",
		)
		return
	}

	var role *discordgo.Role

	if roleID != "" {
		// Look up by role ID - we need to get the guild first to find the role
		// Since we don't have a direct API to get role by ID, we'll need guild_id
		// But if only role_id is provided, we can try to find it by searching guilds
		// For now, let's require guild_id when using role_id for simplicity
		if guildID == "" {
			resp.Diagnostics.AddError(
				"Missing Guild ID",
				"When using role_id, guild_id is also required.",
			)
			return
		}

		// Fetch all roles and find the one with matching ID
		roles, err := d.client.GuildRoles(guildID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Fetching Roles",
				fmt.Sprintf("Unable to fetch roles for guild %s: %s", guildID, err.Error()),
			)
			return
		}

		found := false
		for _, r := range roles {
			if r.ID == roleID {
				role = r
				found = true
				break
			}
		}

		if !found {
			resp.Diagnostics.AddError(
				"Role Not Found",
				fmt.Sprintf("Role with ID %s not found in guild %s", roleID, guildID),
			)
			return
		}
	} else {
		// Look up by name and guild_id
		roles, err := d.client.GuildRoles(guildID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Fetching Roles",
				fmt.Sprintf("Unable to fetch roles for guild %s: %s", guildID, err.Error()),
			)
			return
		}

		found := false
		for _, r := range roles {
			if r.Name == name {
				role = r
				found = true
				break
			}
		}

		if !found {
			resp.Diagnostics.AddError(
				"Role Not Found",
				fmt.Sprintf("Role with name '%s' not found in guild %s", name, guildID),
			)
			return
		}
	}

	// Populate the model with role data
	data.ID = types.StringValue(role.ID)
	data.Name = types.StringValue(role.Name)
	data.Color = types.Int64Value(int64(role.Color))
	data.Position = types.Int64Value(int64(role.Position))
	data.Permissions = types.Int64Value(role.Permissions)
	data.Managed = types.BoolValue(role.Managed)
	data.Mentionable = types.BoolValue(role.Mentionable)
	data.Hoist = types.BoolValue(role.Hoist)
	data.GuildID = types.StringValue(guildID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
