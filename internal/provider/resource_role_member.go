package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the resource type implements the required interfaces.
var _ resource.Resource = &roleMemberResource{}
var _ resource.ResourceWithConfigure = &roleMemberResource{}
var _ resource.ResourceWithImportState = &roleMemberResource{}

// roleMemberResource defines the resource implementation.
type roleMemberResource struct {
	client *discordgo.Session
}

// roleMemberResourceModel describes the resource data model.
type roleMemberResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GuildID types.String `tfsdk:"guild_id"`
	RoleID  types.String `tfsdk:"role_id"`
	UserID  types.String `tfsdk:"user_id"`
}

// NewRoleMemberResource is a helper function to simplify testing.
func NewRoleMemberResource() resource.Resource {
	return &roleMemberResource{}
}

// Metadata returns the resource type name.
func (r *roleMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_member"
}

// Schema defines the schema for the resource.
func (r *roleMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the membership of a user in a Discord role. This resource adds or removes a user from a role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the role membership (format: guild_id:role_id:user_id).",
				Computed:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) where the role exists.",
				Required:    true,
			},
			"role_id": schema.StringAttribute{
				Description: "The ID of the role to add the user to.",
				Required:    true,
			},
			"user_id": schema.StringAttribute{
				Description: "The ID of the user to add to the role.",
				Required:    true,
			},
		},
	}
}

// Configure sets up the resource with the provider's configured client.
func (r *roleMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *roleMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data roleMemberResourceModel

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

	roleID := data.RoleID.ValueString()
	if roleID == "" {
		resp.Diagnostics.AddError(
			"Missing Role ID",
			"The role_id attribute is required.",
		)
		return
	}

	userID := data.UserID.ValueString()
	if userID == "" {
		resp.Diagnostics.AddError(
			"Missing User ID",
			"The user_id attribute is required.",
		)
		return
	}

	// Add the user to the role
	err := r.client.GuildMemberRoleAdd(guildID, userID, roleID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Adding User to Role",
			fmt.Sprintf("Unable to add user %s to role %s in guild %s: %s", userID, roleID, guildID, err.Error()),
		)
		return
	}

	// Set the ID (composite key)
	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s", guildID, roleID, userID))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *roleMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data roleMemberResourceModel

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
	roleID := data.RoleID.ValueString()
	userID := data.UserID.ValueString()

	if guildID == "" || roleID == "" || userID == "" {
		resp.Diagnostics.AddError(
			"Missing Required Fields",
			"The guild_id, role_id, and user_id are required to read the role membership.",
		)
		return
	}

	// Fetch the guild member to check if they have the role
	member, err := r.client.GuildMember(guildID, userID)
	if err != nil {
		// If member doesn't exist or is not in the guild, mark as removed
		resp.Diagnostics.AddWarning(
			"Member Not Found",
			fmt.Sprintf("Member %s was not found in guild %s. They may have left the server. Removing from state.", userID, guildID),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Check if the member has the role
	hasRole := false
	for _, memberRoleID := range member.Roles {
		if memberRoleID == roleID {
			hasRole = true
			break
		}
	}

	if !hasRole {
		// Member doesn't have the role, mark as removed
		resp.Diagnostics.AddWarning(
			"Role Membership Not Found",
			fmt.Sprintf("User %s does not have role %s in guild %s. Removing from state.", userID, roleID, guildID),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Update the ID (composite key)
	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s", guildID, roleID, userID))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *roleMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state roleMemberResourceModel

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

	oldGuildID := state.GuildID.ValueString()
	oldRoleID := state.RoleID.ValueString()
	oldUserID := state.UserID.ValueString()

	newGuildID := plan.GuildID.ValueString()
	newRoleID := plan.RoleID.ValueString()
	newUserID := plan.UserID.ValueString()

	// If any of the IDs changed, we need to remove from old and add to new
	// This is essentially a replacement
	if oldGuildID != newGuildID || oldRoleID != newRoleID || oldUserID != newUserID {
		// Remove from old role
		if oldGuildID != "" && oldRoleID != "" && oldUserID != "" {
			err := r.client.GuildMemberRoleRemove(oldGuildID, oldUserID, oldRoleID)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Error Removing Old Role Membership",
					fmt.Sprintf("Unable to remove user %s from role %s in guild %s: %s. Continuing with new role assignment.", oldUserID, oldRoleID, oldGuildID, err.Error()),
				)
			}
		}

		// Add to new role
		if newGuildID != "" && newRoleID != "" && newUserID != "" {
			err := r.client.GuildMemberRoleAdd(newGuildID, newUserID, newRoleID)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Adding User to New Role",
					fmt.Sprintf("Unable to add user %s to role %s in guild %s: %s", newUserID, newRoleID, newGuildID, err.Error()),
				)
				return
			}
		}
	}

	// Update state
	data := plan
	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s", newGuildID, newRoleID, newUserID))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *roleMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data roleMemberResourceModel

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
	roleID := data.RoleID.ValueString()
	userID := data.UserID.ValueString()

	if guildID == "" || roleID == "" || userID == "" {
		resp.Diagnostics.AddError(
			"Missing Required Fields",
			"The guild_id, role_id, and user_id are required to delete the role membership.",
		)
		return
	}

	// Remove the user from the role
	err := r.client.GuildMemberRoleRemove(guildID, userID, roleID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Removing User from Role",
			fmt.Sprintf("Unable to remove user %s from role %s in guild %s: %s", userID, roleID, guildID, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform.
func (r *roleMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: guild_id:role_id:user_id
	importID := req.ID
	if importID == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"The import ID must be in the format 'guild_id:role_id:user_id'.",
		)
		return
	}

	// Parse the import ID
	parts := strings.Split(importID, ":")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			"The import ID must be in the format 'guild_id:role_id:user_id' (e.g., '123456789012345678:987654321098765432:111111111111111111').",
		)
		return
	}

	guildID := parts[0]
	roleID := parts[1]
	userID := parts[2]

	// Set the IDs in state - Read will verify the membership exists
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("guild_id"), types.StringValue(guildID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), types.StringValue(roleID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), types.StringValue(userID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(importID))...)
}
