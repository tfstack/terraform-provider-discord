package provider

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the resource type implements the required interfaces.
var _ resource.Resource = &roleResource{}
var _ resource.ResourceWithConfigure = &roleResource{}
var _ resource.ResourceWithImportState = &roleResource{}

// roleResource defines the resource implementation.
type roleResource struct {
	client *discordgo.Session
}

// roleResourceModel describes the resource data model.
type roleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	GuildID     types.String `tfsdk:"guild_id"`
	Color       types.Int64  `tfsdk:"color"`
	Hoist       types.Bool   `tfsdk:"hoist"`
	Mentionable types.Bool   `tfsdk:"mentionable"`
	Permissions types.Int64  `tfsdk:"permissions"`
	Position    types.Int64  `tfsdk:"position"`
	Managed     types.Bool   `tfsdk:"managed"`
}

// NewRoleResource is a helper function to simplify testing.
func NewRoleResource() resource.Resource {
	return &roleResource{}
}

// Metadata returns the resource type name.
func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the schema for the resource.
func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a Discord role in a guild (server).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the role.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the role. Must be 1-100 characters.",
				Required:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) where the role will be created.",
				Required:    true,
			},
			"color": schema.Int64Attribute{
				Description: "The color of the role as a decimal integer (0-16777215). 0 means no color.",
				Optional:    true,
				Computed:    true,
			},
			"hoist": schema.BoolAttribute{
				Description: "Whether to display the role's users separately in the member list.",
				Optional:    true,
				Computed:    true,
			},
			"mentionable": schema.BoolAttribute{
				Description: "Whether this role is mentionable.",
				Optional:    true,
				Computed:    true,
			},
			"permissions": schema.Int64Attribute{
				Description: "The permissions integer for the role on the guild. This is a combination of bit masks.",
				Optional:    true,
				Computed:    true,
			},
			"position": schema.Int64Attribute{
				Description: "The position of the role in the guild's role hierarchy. Lower numbers appear higher in the list.",
				Computed:    true,
			},
			"managed": schema.BoolAttribute{
				Description: "Whether this role is managed by an integration. This is read-only and set by Discord.",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the resource with the provider's configured client.
func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*discordgo.Session)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *discordgo.Session, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data roleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure client is configured
	if r.client == nil {
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

	name := data.Name.ValueString()
	if name == "" {
		resp.Diagnostics.AddError(
			"Missing Role Name",
			"The name attribute is required.",
		)
		return
	}

	// Validate name length (Discord requires 1-100 characters)
	if len(name) < 1 || len(name) > 100 {
		resp.Diagnostics.AddError(
			"Invalid Role Name",
			"Role name must be between 1 and 100 characters.",
		)
		return
	}

	// Note: The @everyone role should be managed using discord_everyone_role resource
	if name == "@everyone" {
		resp.Diagnostics.AddError(
			"Invalid Role Name",
			"The @everyone role cannot be managed with discord_role. Use discord_everyone_role resource instead.",
		)
		return
	}

	// Prepare role creation data
	roleParams := &discordgo.RoleParams{
		Name: name,
	}

	// Set color if provided
	if !data.Color.IsNull() && !data.Color.IsUnknown() {
		colorValue := int(data.Color.ValueInt64())
		// Validate color range (0-16777215)
		if colorValue < 0 || colorValue > 16777215 {
			resp.Diagnostics.AddError(
				"Invalid Color",
				"Color must be between 0 and 16777215 (0xFFFFFF).",
			)
			return
		}
		roleParams.Color = &colorValue
	}

	// Set hoist if provided
	if !data.Hoist.IsNull() && !data.Hoist.IsUnknown() {
		hoistValue := data.Hoist.ValueBool()
		roleParams.Hoist = &hoistValue
	}

	// Set mentionable if provided
	if !data.Mentionable.IsNull() && !data.Mentionable.IsUnknown() {
		mentionableValue := data.Mentionable.ValueBool()
		roleParams.Mentionable = &mentionableValue
	}

	// Set permissions if provided
	if !data.Permissions.IsNull() && !data.Permissions.IsUnknown() {
		permissionsValue := data.Permissions.ValueInt64()
		roleParams.Permissions = &permissionsValue
	}

	// Create the role
	role, err := r.client.GuildRoleCreate(guildID, roleParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Role",
			fmt.Sprintf("Unable to create role %s in guild %s: %s", name, guildID, err.Error()),
		)
		return
	}

	// Verify role was created successfully
	if role == nil {
		resp.Diagnostics.AddError(
			"Role Creation Failed",
			fmt.Sprintf("Role creation API call succeeded but returned nil role for %s in guild %s. This may indicate a Discord API issue.", name, guildID),
		)
		return
	}

	// Verify role ID is set
	if role.ID == "" {
		resp.Diagnostics.AddError(
			"Invalid Role Response",
			fmt.Sprintf("Role was created but has no ID. Role name: %s, Guild ID: %s", name, guildID),
		)
		return
	}

	// Populate the model with role data from Discord
	// Important: Always set all attributes from the created role, even if they weren't in the plan
	// This ensures Terraform state matches what Discord actually has
	data.ID = types.StringValue(role.ID)
	data.Name = types.StringValue(role.Name)
	data.GuildID = types.StringValue(guildID)
	data.Color = types.Int64Value(int64(role.Color))
	data.Hoist = types.BoolValue(role.Hoist)
	data.Mentionable = types.BoolValue(role.Mentionable)
	data.Permissions = types.Int64Value(role.Permissions)
	data.Position = types.Int64Value(int64(role.Position))
	data.Managed = types.BoolValue(role.Managed)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data roleResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure client is configured
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Discord Client Not Configured",
			"The Discord client was not properly configured. This is a provider error.",
		)
		return
	}

	roleID := data.ID.ValueString()
	if roleID == "" {
		resp.Diagnostics.AddError(
			"Missing Role ID",
			"The role ID is missing from the state.",
		)
		return
	}

	guildID := data.GuildID.ValueString()
	if guildID == "" {
		resp.Diagnostics.AddError(
			"Missing Guild ID",
			"The guild ID is missing from the state.",
		)
		return
	}

	// Fetch all roles and find the one with matching ID
	roles, err := r.client.GuildRoles(guildID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Roles",
			fmt.Sprintf("Unable to fetch roles for guild %s: %s", guildID, err.Error()),
		)
		return
	}

	var role *discordgo.Role
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
			fmt.Sprintf("Role with ID %s not found in guild %s. The role may have been deleted.", roleID, guildID),
		)
		return
	}

	// Update the model with role data
	data.ID = types.StringValue(role.ID)
	data.Name = types.StringValue(role.Name)
	data.Color = types.Int64Value(int64(role.Color))
	data.Hoist = types.BoolValue(role.Hoist)
	data.Mentionable = types.BoolValue(role.Mentionable)
	data.Permissions = types.Int64Value(role.Permissions)
	data.Position = types.Int64Value(int64(role.Position))
	data.Managed = types.BoolValue(role.Managed)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state roleResourceModel

	// Read Terraform plan and state data into the models
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure client is configured
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Discord Client Not Configured",
			"The Discord client was not properly configured. This is a provider error.",
		)
		return
	}

	roleID := state.ID.ValueString()
	if roleID == "" {
		resp.Diagnostics.AddError(
			"Missing Role ID",
			"The role ID is missing from state.",
		)
		return
	}

	guildID := plan.GuildID.ValueString()
	if guildID == "" {
		resp.Diagnostics.AddError(
			"Missing Guild ID",
			"The guild_id attribute is required.",
		)
		return
	}

	name := plan.Name.ValueString()
	if name == "" {
		resp.Diagnostics.AddError(
			"Missing Role Name",
			"The name attribute is required.",
		)
		return
	}

	// Validate name length
	if len(name) < 1 || len(name) > 100 {
		resp.Diagnostics.AddError(
			"Invalid Role Name",
			"Role name must be between 1 and 100 characters.",
		)
		return
	}

	// Prepare role update data
	roleParams := &discordgo.RoleParams{
		Name: name,
	}

	// Set color if provided or changed
	if !plan.Color.IsNull() && !plan.Color.IsUnknown() {
		colorValue := int(plan.Color.ValueInt64())
		// Validate color range
		if colorValue < 0 || colorValue > 16777215 {
			resp.Diagnostics.AddError(
				"Invalid Color",
				"Color must be between 0 and 16777215 (0xFFFFFF).",
			)
			return
		}
		roleParams.Color = &colorValue
	}

	// Set hoist if provided or changed
	if !plan.Hoist.IsNull() && !plan.Hoist.IsUnknown() {
		hoistValue := plan.Hoist.ValueBool()
		roleParams.Hoist = &hoistValue
	}

	// Set mentionable if provided or changed
	if !plan.Mentionable.IsNull() && !plan.Mentionable.IsUnknown() {
		mentionableValue := plan.Mentionable.ValueBool()
		roleParams.Mentionable = &mentionableValue
	}

	// Set permissions if provided or changed
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		permissionsValue := plan.Permissions.ValueInt64()
		roleParams.Permissions = &permissionsValue
	}

	// Update the role
	role, err := r.client.GuildRoleEdit(guildID, roleID, roleParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Role",
			fmt.Sprintf("Unable to update role %s in guild %s: %s", roleID, guildID, err.Error()),
		)
		return
	}

	// Update the model with role data from Discord
	// Important: Always set all attributes from the updated role, not just from plan
	data := plan
	data.ID = types.StringValue(role.ID)
	data.Name = types.StringValue(role.Name)
	data.GuildID = types.StringValue(guildID)
	data.Color = types.Int64Value(int64(role.Color))
	data.Hoist = types.BoolValue(role.Hoist)
	data.Mentionable = types.BoolValue(role.Mentionable)
	data.Permissions = types.Int64Value(role.Permissions)
	data.Position = types.Int64Value(int64(role.Position))
	data.Managed = types.BoolValue(role.Managed)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data roleResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure client is configured
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Discord Client Not Configured",
			"The Discord client was not properly configured. This is a provider error.",
		)
		return
	}

	roleID := data.ID.ValueString()
	if roleID == "" {
		resp.Diagnostics.AddError(
			"Missing Role ID",
			"The role ID is missing from the state.",
		)
		return
	}

	guildID := data.GuildID.ValueString()
	if guildID == "" {
		resp.Diagnostics.AddError(
			"Missing Guild ID",
			"The guild ID is missing from the state.",
		)
		return
	}

	// Delete the role
	err := r.client.GuildRoleDelete(guildID, roleID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Role",
			fmt.Sprintf("Unable to delete role %s in guild %s: %s", roleID, guildID, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform state.
func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: guild_id:role_id
	// Parse the import ID
	importID := req.ID
	if importID == "" {
		resp.Diagnostics.AddError(
			"Missing Import ID",
			"The import ID is required. Format: guild_id:role_id",
		)
		return
	}

	// Ensure client is configured
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Discord Client Not Configured",
			"The Discord client was not properly configured. This is a provider error.",
		)
		return
	}

	// Try to parse as guild_id:role_id format
	var guildID, roleID string
	if len(importID) > 18 && importID[18] == ':' {
		// Format: guild_id:role_id
		guildID = importID[:18]
		roleID = importID[19:]
	} else {
		// Fallback: assume it's just role_id, but we need guild_id
		// For now, require the format guild_id:role_id
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			"Import ID must be in the format 'guild_id:role_id'. Example: '123456789012345678:987654321098765432'. Note: Use discord_everyone_role resource for @everyone role.",
		)
		return
	}

	// Prevent importing @everyone role (use discord_everyone_role instead)
	if roleID == guildID {
		resp.Diagnostics.AddError(
			"Cannot Import @everyone Role",
			"The @everyone role cannot be imported with discord_role. Use discord_everyone_role resource instead.",
		)
		return
	}

	// Fetch the role to populate state
	roles, err := r.client.GuildRoles(guildID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Role",
			fmt.Sprintf("Unable to fetch roles for guild %s: %s", guildID, err.Error()),
		)
		return
	}

	var role *discordgo.Role
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

	// Create a model with the role data
	var data roleResourceModel
	data.ID = types.StringValue(role.ID)
	data.Name = types.StringValue(role.Name)
	data.GuildID = types.StringValue(guildID)
	data.Color = types.Int64Value(int64(role.Color))
	data.Hoist = types.BoolValue(role.Hoist)
	data.Mentionable = types.BoolValue(role.Mentionable)
	data.Permissions = types.Int64Value(role.Permissions)
	data.Position = types.Int64Value(int64(role.Position))
	data.Managed = types.BoolValue(role.Managed)

	// Save the imported state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
