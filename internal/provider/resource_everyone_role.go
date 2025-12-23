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
var _ resource.Resource = &everyoneRoleResource{}
var _ resource.ResourceWithConfigure = &everyoneRoleResource{}

// everyoneRoleResource defines the resource implementation.
// Note: This resource does NOT implement ResourceWithImportState because
// the @everyone role always exists and doesn't need importing.
type everyoneRoleResource struct {
	client *discordgo.Session
}

// everyoneRoleResourceModel describes the resource data model.
type everyoneRoleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	GuildID     types.String `tfsdk:"guild_id"`
	Color       types.Int64  `tfsdk:"color"`
	Hoist       types.Bool   `tfsdk:"hoist"`
	Mentionable types.Bool   `tfsdk:"mentionable"`
	Permissions types.Int64  `tfsdk:"permissions"`
	Position    types.Int64  `tfsdk:"position"`
	Managed     types.Bool   `tfsdk:"managed"`
}

// NewEveryoneRoleResource is a helper function to simplify testing.
func NewEveryoneRoleResource() resource.Resource {
	return &everyoneRoleResource{}
}

// Metadata returns the resource type name.
func (r *everyoneRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_everyone_role"
}

// Schema defines the schema for the resource.
func (r *everyoneRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the @everyone role in a Discord guild (server). The @everyone role is a special default role that always exists and cannot be created or deleted, only modified.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the @everyone role (same as guild_id).",
				Computed:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) where the @everyone role will be managed.",
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
				Description: "The position of the role in the guild's role hierarchy. This is always 0 for @everyone role.",
				Computed:    true,
			},
			"managed": schema.BoolAttribute{
				Description: "Whether this role is managed by an integration. This is always false for @everyone role.",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the resource with the provider's configured client.
func (r *everyoneRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create reads the existing @everyone role and sets the initial Terraform state.
// The @everyone role cannot be created as it always exists.
func (r *everyoneRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data everyoneRoleResourceModel

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

	// Fetch the existing @everyone role (it always exists)
	roles, err := r.client.GuildRoles(guildID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching @everyone Role",
			fmt.Sprintf("Unable to fetch roles for guild %s: %s", guildID, err.Error()),
		)
		return
	}

	var everyoneRole *discordgo.Role
	found := false
	for _, role := range roles {
		if role.ID == guildID || role.Name == "@everyone" {
			everyoneRole = role
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"@everyone Role Not Found",
			fmt.Sprintf("The @everyone role was not found in guild %s. This should not happen.", guildID),
		)
		return
	}

	// Check if any attributes need updating
	needsUpdate := false
	roleParams := &discordgo.RoleParams{
		Name: "@everyone",
	}

	// Check if color needs updating
	if !data.Color.IsNull() && !data.Color.IsUnknown() {
		colorValue := int(data.Color.ValueInt64())
		if colorValue < 0 || colorValue > 16777215 {
			resp.Diagnostics.AddError(
				"Invalid Color",
				"Color must be between 0 and 16777215 (0xFFFFFF).",
			)
			return
		}
		if colorValue != everyoneRole.Color {
			roleParams.Color = &colorValue
			needsUpdate = true
		}
	}

	// Check if hoist needs updating
	if !data.Hoist.IsNull() && !data.Hoist.IsUnknown() {
		hoistValue := data.Hoist.ValueBool()
		if hoistValue != everyoneRole.Hoist {
			roleParams.Hoist = &hoistValue
			needsUpdate = true
		}
	}

	// Check if mentionable needs updating
	if !data.Mentionable.IsNull() && !data.Mentionable.IsUnknown() {
		mentionableValue := data.Mentionable.ValueBool()
		if mentionableValue != everyoneRole.Mentionable {
			roleParams.Mentionable = &mentionableValue
			needsUpdate = true
		}
	}

	// Check if permissions need updating
	if !data.Permissions.IsNull() && !data.Permissions.IsUnknown() {
		permissionsValue := data.Permissions.ValueInt64()
		if permissionsValue != everyoneRole.Permissions {
			roleParams.Permissions = &permissionsValue
			needsUpdate = true
		}
	}

	// Update the role if needed
	if needsUpdate {
		// Check if we're trying to update permissions - this requires special permissions
		updatingPermissions := roleParams.Permissions != nil

		updatedRole, err := r.client.GuildRoleEdit(guildID, everyoneRole.ID, roleParams)
		if err != nil {
			// Provide more helpful error message for permission issues
			errorMsg := fmt.Sprintf("Unable to update @everyone role in guild %s: %s", guildID, err.Error())
			if err.Error() != "" && (err.Error() == "HTTP 403 Forbidden" || err.Error() == "Missing Permissions") {
				errorMsg += "\n\nTo manage the @everyone role, the bot needs:\n" +
					"1. 'Manage Roles' permission in the server\n" +
					"2. The bot's role must be higher in the role hierarchy than @everyone\n" +
					"   - Go to Server Settings → Roles and drag the bot's role ABOVE @everyone\n" +
					"   - @everyone is at the bottom, so the bot's role should be higher\n"
				if updatingPermissions {
					errorMsg += "3. When modifying @everyone permissions, the bot must have ALL permissions it's trying to grant\n" +
						"   - Discord restriction: bots can only grant permissions they themselves possess\n" +
						"   - Alternative: Grant the bot 'Administrator' permission (gives all permissions)\n" +
						"   - Example: If setting permissions=104324673, bot needs those same permissions\n"
				}
				errorMsg += "4. Ensure the bot has been granted these permissions in Server Settings → Roles"
			}
			resp.Diagnostics.AddError(
				"Error Updating @everyone Role",
				errorMsg,
			)
			return
		}
		everyoneRole = updatedRole
	}

	// Populate the model with @everyone role data from Discord
	// Important: Always set all attributes from the role, even if they weren't in the plan
	// This ensures Terraform state matches what Discord actually has
	data.ID = types.StringValue(everyoneRole.ID)
	data.GuildID = types.StringValue(guildID)
	data.Color = types.Int64Value(int64(everyoneRole.Color))
	data.Hoist = types.BoolValue(everyoneRole.Hoist)
	data.Mentionable = types.BoolValue(everyoneRole.Mentionable)
	data.Permissions = types.Int64Value(everyoneRole.Permissions)
	data.Position = types.Int64Value(int64(everyoneRole.Position))
	data.Managed = types.BoolValue(everyoneRole.Managed)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *everyoneRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data everyoneRoleResourceModel

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

	guildID := data.GuildID.ValueString()
	if guildID == "" {
		resp.Diagnostics.AddError(
			"Missing Guild ID",
			"The guild ID is missing from the state.",
		)
		return
	}

	// Fetch all roles and find the @everyone role
	roles, err := r.client.GuildRoles(guildID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Roles",
			fmt.Sprintf("Unable to fetch roles for guild %s: %s", guildID, err.Error()),
		)
		return
	}

	var everyoneRole *discordgo.Role
	found := false
	for _, role := range roles {
		if role.ID == guildID || role.Name == "@everyone" {
			everyoneRole = role
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"@everyone Role Not Found",
			fmt.Sprintf("The @everyone role was not found in guild %s. This should not happen.", guildID),
		)
		return
	}

	// Update the model with role data
	data.ID = types.StringValue(everyoneRole.ID)
	data.Color = types.Int64Value(int64(everyoneRole.Color))
	data.Hoist = types.BoolValue(everyoneRole.Hoist)
	data.Mentionable = types.BoolValue(everyoneRole.Mentionable)
	data.Permissions = types.Int64Value(everyoneRole.Permissions)
	data.Position = types.Int64Value(int64(everyoneRole.Position))
	data.Managed = types.BoolValue(everyoneRole.Managed)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *everyoneRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state everyoneRoleResourceModel

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

	guildID := plan.GuildID.ValueString()
	if guildID == "" {
		resp.Diagnostics.AddError(
			"Missing Guild ID",
			"The guild_id attribute is required.",
		)
		return
	}

	// Fetch the @everyone role
	roles, err := r.client.GuildRoles(guildID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching @everyone Role",
			fmt.Sprintf("Unable to fetch roles for guild %s: %s", guildID, err.Error()),
		)
		return
	}

	var everyoneRole *discordgo.Role
	found := false
	for _, role := range roles {
		if role.ID == guildID || role.Name == "@everyone" {
			everyoneRole = role
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"@everyone Role Not Found",
			fmt.Sprintf("The @everyone role was not found in guild %s. This should not happen.", guildID),
		)
		return
	}

	// Prepare role update data
	roleParams := &discordgo.RoleParams{
		Name: "@everyone",
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
	updatingPermissions := false
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		permissionsValue := plan.Permissions.ValueInt64()
		roleParams.Permissions = &permissionsValue
		updatingPermissions = true
	}

	// Check if we actually need to update anything
	needsUpdate := roleParams.Color != nil || roleParams.Hoist != nil || roleParams.Mentionable != nil || roleParams.Permissions != nil
	var role *discordgo.Role
	if !needsUpdate {
		// No changes needed, just use the current role data
		role = everyoneRole
	} else {
		// Update the role
		var err error
		role, err = r.client.GuildRoleEdit(guildID, everyoneRole.ID, roleParams)
		if err != nil {
			// Provide more helpful error message for permission issues
			errorMsg := fmt.Sprintf("Unable to update @everyone role in guild %s: %s", guildID, err.Error())
			if err.Error() != "" && (err.Error() == "HTTP 403 Forbidden" || err.Error() == "Missing Permissions") {
				errorMsg += "\n\nTo manage the @everyone role, the bot needs:\n" +
					"1. 'Manage Roles' permission in the server\n" +
					"2. The bot's role must be higher in the role hierarchy than @everyone\n" +
					"   - Go to Server Settings → Roles and drag the bot's role ABOVE @everyone\n" +
					"   - @everyone is at the bottom, so the bot's role should be higher\n"
				if updatingPermissions {
					errorMsg += "3. When modifying @everyone permissions, the bot must have ALL permissions it's trying to grant\n" +
						"   - Discord restriction: bots can only grant permissions they themselves possess\n" +
						"   - Alternative: Grant the bot 'Administrator' permission (gives all permissions)\n" +
						"   - Example: If setting permissions=104324673, bot needs those same permissions\n"
				}
				errorMsg += "4. Ensure the bot has been granted these permissions in Server Settings → Roles"
			}
			resp.Diagnostics.AddError(
				"Error Updating @everyone Role",
				errorMsg,
			)
			return
		}
	}

	// Update the model with role data from Discord
	// Important: Always set all attributes from the updated role, not just from plan
	data := plan
	data.ID = types.StringValue(role.ID)
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

// Delete removes the resource from Terraform state only.
// The @everyone role cannot be deleted as it always exists.
// Note: The last settings applied by Terraform will remain in Discord since
// we cannot restore the previous state (we don't know what it was before).
func (r *everyoneRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The @everyone role cannot be deleted, just remove from state
	resp.Diagnostics.AddWarning(
		"Cannot Delete @everyone Role",
		"The @everyone role cannot be deleted as it is a default role that always exists. Removing from Terraform state only.\n\n"+
			"Note: The last settings applied by Terraform will remain in Discord. Terraform cannot restore the @everyone role "+
			"to its previous state because it doesn't know what the state was before Terraform managed it. "+
			"If you need to revert changes, manually update the @everyone role in Discord or re-apply Terraform with different values.",
	)
	// No API call needed - just remove from state
}
