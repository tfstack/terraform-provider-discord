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
var _ datasource.DataSource = &rolesDataSource{}

// rolesDataSource defines the data source implementation.
type rolesDataSource struct {
	client *discordgo.Session
}

// rolesDataSourceModel describes the data source data model.
type rolesDataSourceModel struct {
	GuildID types.String `tfsdk:"guild_id"`
	Roles   types.List   `tfsdk:"roles"`
}

// roleModel describes a single role in the data source.
type roleModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Color       types.Int64  `tfsdk:"color"`
	Position    types.Int64  `tfsdk:"position"`
	Permissions types.Int64  `tfsdk:"permissions"`
	Managed     types.Bool   `tfsdk:"managed"`
	Mentionable types.Bool   `tfsdk:"mentionable"`
	Hoist       types.Bool   `tfsdk:"hoist"`
}

// NewRolesDataSource is a helper function to simplify testing.
func NewRolesDataSource() datasource.DataSource {
	return &rolesDataSource{}
}

// Metadata returns the data source type name.
func (d *rolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

// Schema defines the schema for the data source.
func (d *rolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves all roles from a Discord guild (server).",
		Attributes: map[string]schema.Attribute{
			"guild_id": schema.StringAttribute{
				Description: "The ID of the Discord guild (server).",
				Required:    true,
			},
			"roles": schema.ListNestedAttribute{
				Description: "List of roles in the guild.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the role.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the role.",
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
				},
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *rolesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *rolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data rolesDataSourceModel

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

	// Fetch all roles for the guild
	roles, err := d.client.GuildRoles(guildID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Roles",
			fmt.Sprintf("Unable to fetch roles for guild %s: %s", guildID, err.Error()),
		)
		return
	}

	// Convert Discord roles to Terraform model
	roleList := make([]roleModel, 0, len(roles))
	for _, role := range roles {
		roleModel := roleModel{
			ID:          types.StringValue(role.ID),
			Name:        types.StringValue(role.Name),
			Color:       types.Int64Value(int64(role.Color)),
			Position:    types.Int64Value(int64(role.Position)),
			Permissions: types.Int64Value(role.Permissions),
			Managed:     types.BoolValue(role.Managed),
			Mentionable: types.BoolValue(role.Mentionable),
			Hoist:       types.BoolValue(role.Hoist),
		}
		roleList = append(roleList, roleModel)
	}

	// Convert to Terraform list
	roleObjectType := map[string]attr.Type{
		"id":          types.StringType,
		"name":        types.StringType,
		"color":       types.Int64Type,
		"position":    types.Int64Type,
		"permissions": types.Int64Type,
		"managed":     types.BoolType,
		"mentionable": types.BoolType,
		"hoist":       types.BoolType,
	}

	roleListValue := make([]attr.Value, 0, len(roleList))
	for _, role := range roleList {
		roleMap := map[string]attr.Value{
			"id":          role.ID,
			"name":        role.Name,
			"color":       role.Color,
			"position":    role.Position,
			"permissions": role.Permissions,
			"managed":     role.Managed,
			"mentionable": role.Mentionable,
			"hoist":       role.Hoist,
		}
		roleObj, diags := types.ObjectValue(roleObjectType, roleMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		roleListValue = append(roleListValue, roleObj)
	}

	rolesValue, diags := types.ListValue(types.ObjectType{AttrTypes: roleObjectType}, roleListValue)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Roles = rolesValue

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
