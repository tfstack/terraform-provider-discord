package provider

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the resource type implements the required interfaces.
var _ resource.Resource = &inviteResource{}
var _ resource.ResourceWithConfigure = &inviteResource{}
var _ resource.ResourceWithImportState = &inviteResource{}

// inviteResource defines the resource implementation.
type inviteResource struct {
	client *discordgo.Session
}

// inviteResourceModel describes the resource data model.
type inviteResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Code      types.String `tfsdk:"code"`
	ChannelID types.String `tfsdk:"channel_id"`
	MaxAge    types.Int64  `tfsdk:"max_age"`
	MaxUses   types.Int64  `tfsdk:"max_uses"`
	Temporary types.Bool   `tfsdk:"temporary"`
	Unique    types.Bool   `tfsdk:"unique"`
	URL       types.String `tfsdk:"url"`
	CreatedAt types.String `tfsdk:"created_at"`
	ExpiresAt types.String `tfsdk:"expires_at"`
	Uses      types.Int64  `tfsdk:"uses"`
}

// NewInviteResource is a helper function to simplify testing.
func NewInviteResource() resource.Resource {
	return &inviteResource{}
}

// Metadata returns the resource type name.
func (r *inviteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_invite"
}

// Schema defines the schema for the resource.
func (r *inviteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a Discord invite for a channel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the invite (same as code).",
				Computed:    true,
			},
			"code": schema.StringAttribute{
				Description: "The invite code (unique identifier for the invite).",
				Computed:    true,
			},
			"channel_id": schema.StringAttribute{
				Description: "The ID of the channel to create an invite for.",
				Required:    true,
			},
			"max_age": schema.Int64Attribute{
				Description: "Duration (in seconds) after which the invite expires. 0 means the invite never expires. Defaults to 86400 (24 hours).",
				Optional:    true,
				Computed:    true,
			},
			"max_uses": schema.Int64Attribute{
				Description: "Maximum number of times the invite can be used. 0 means unlimited. Defaults to 0.",
				Optional:    true,
				Computed:    true,
			},
			"temporary": schema.BoolAttribute{
				Description: "Whether the invite grants temporary membership. If true, members will be kicked when they disconnect unless they're assigned a role. Defaults to false.",
				Optional:    true,
				Computed:    true,
			},
			"unique": schema.BoolAttribute{
				Description: "Whether the invite should be unique. If true, Discord will try to reuse a similar invite. Defaults to false.",
				Optional:    true,
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The full invite URL (https://discord.gg/{code}).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "When the invite was created (ISO 8601 timestamp).",
				Computed:    true,
			},
			"expires_at": schema.StringAttribute{
				Description: "When the invite expires (ISO 8601 timestamp). Null if max_age is 0.",
				Computed:    true,
			},
			"uses": schema.Int64Attribute{
				Description: "Number of times the invite has been used.",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the resource with the provider's configured client.
func (r *inviteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *inviteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data inviteResourceModel

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

	channelID := data.ChannelID.ValueString()
	if channelID == "" {
		resp.Diagnostics.AddError(
			"Missing Channel ID",
			"The channel_id attribute is required.",
		)
		return
	}

	// Prepare invite creation data
	inviteData := discordgo.Invite{
		Channel: &discordgo.Channel{ID: channelID},
	}

	// Set max_age (default to 86400 seconds = 24 hours if not specified)
	maxAge := int64(86400)
	if !data.MaxAge.IsNull() && !data.MaxAge.IsUnknown() {
		maxAge = data.MaxAge.ValueInt64()
		// Validate max_age (0-604800 seconds = 0-7 days)
		if maxAge < 0 || maxAge > 604800 {
			resp.Diagnostics.AddError(
				"Invalid Max Age",
				"max_age must be between 0 and 604800 seconds (0-7 days).",
			)
			return
		}
	}
	inviteData.MaxAge = int(maxAge)

	// Set max_uses (default to 0 = unlimited if not specified)
	maxUses := int64(0)
	if !data.MaxUses.IsNull() && !data.MaxUses.IsUnknown() {
		maxUses = data.MaxUses.ValueInt64()
		// Validate max_uses (0-100)
		if maxUses < 0 || maxUses > 100 {
			resp.Diagnostics.AddError(
				"Invalid Max Uses",
				"max_uses must be between 0 and 100.",
			)
			return
		}
	}
	inviteData.MaxUses = int(maxUses)

	// Set temporary (default to false if not specified)
	temporary := false
	if !data.Temporary.IsNull() && !data.Temporary.IsUnknown() {
		temporary = data.Temporary.ValueBool()
	}
	inviteData.Temporary = temporary

	// Set unique (default to false if not specified)
	unique := false
	if !data.Unique.IsNull() && !data.Unique.IsUnknown() {
		unique = data.Unique.ValueBool()
	}
	inviteData.Unique = unique

	// Create the invite
	invite, err := r.client.ChannelInviteCreate(channelID, discordgo.Invite{
		MaxAge:    inviteData.MaxAge,
		MaxUses:   inviteData.MaxUses,
		Temporary: inviteData.Temporary,
		Unique:    inviteData.Unique,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Invite",
			fmt.Sprintf("Unable to create invite for channel %s: %s", channelID, err.Error()),
		)
		return
	}

	// Verify invite was created successfully
	if invite == nil {
		resp.Diagnostics.AddError(
			"Invite Creation Failed",
			fmt.Sprintf("Invite creation API call succeeded but returned nil invite for channel %s. This may indicate a Discord API issue.", channelID),
		)
		return
	}

	// Verify invite code is set
	if invite.Code == "" {
		resp.Diagnostics.AddError(
			"Invalid Invite Response",
			fmt.Sprintf("Invite was created but has no code. Channel ID: %s", channelID),
		)
		return
	}

	// Populate the model with invite data from Discord
	data.ID = types.StringValue(invite.Code)
	data.Code = types.StringValue(invite.Code)
	data.ChannelID = types.StringValue(channelID)
	data.MaxAge = types.Int64Value(int64(invite.MaxAge))
	data.MaxUses = types.Int64Value(int64(invite.MaxUses))
	data.Temporary = types.BoolValue(invite.Temporary)
	data.Unique = types.BoolValue(invite.Unique)
	data.URL = types.StringValue(fmt.Sprintf("https://discord.gg/%s", invite.Code))

	// Created at timestamp
	if !invite.CreatedAt.IsZero() {
		data.CreatedAt = types.StringValue(invite.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.CreatedAt = types.StringNull()
	}

	// Expires at timestamp
	if invite.ExpiresAt != nil && !invite.ExpiresAt.IsZero() {
		data.ExpiresAt = types.StringValue(invite.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.ExpiresAt = types.StringNull()
	}

	// Uses count
	data.Uses = types.Int64Value(int64(invite.Uses))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *inviteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data inviteResourceModel

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

	code := data.Code.ValueString()
	if code == "" {
		// Fallback to ID if code is not set
		code = data.ID.ValueString()
	}

	if code == "" {
		resp.Diagnostics.AddError(
			"Missing Invite Code",
			"The invite code is missing from state.",
		)
		return
	}

	// Fetch the invite with additional metadata
	invite, err := r.client.InviteWithCounts(code)
	if err != nil {
		// If invite doesn't exist or was deleted, mark as removed
		resp.Diagnostics.AddWarning(
			"Invite Not Found",
			fmt.Sprintf("Invite %s was not found. It may have been deleted or expired. Removing from state.", code),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Update model with invite data
	data.ID = types.StringValue(invite.Code)
	data.Code = types.StringValue(invite.Code)

	if invite.Channel != nil {
		data.ChannelID = types.StringValue(invite.Channel.ID)
	} else {
		// Keep existing channel_id from state if not in response
		if data.ChannelID.IsNull() || data.ChannelID.IsUnknown() {
			resp.Diagnostics.AddWarning(
				"Missing Channel ID",
				"Invite response did not include channel information.",
			)
		}
	}

	data.MaxAge = types.Int64Value(int64(invite.MaxAge))
	data.MaxUses = types.Int64Value(int64(invite.MaxUses))
	data.Temporary = types.BoolValue(invite.Temporary)
	data.Unique = types.BoolValue(invite.Unique)
	data.URL = types.StringValue(fmt.Sprintf("https://discord.gg/%s", invite.Code))

	// Created at timestamp
	if !invite.CreatedAt.IsZero() {
		data.CreatedAt = types.StringValue(invite.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.CreatedAt = types.StringNull()
	}

	// Expires at timestamp
	if invite.ExpiresAt != nil && !invite.ExpiresAt.IsZero() {
		data.ExpiresAt = types.StringValue(invite.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.ExpiresAt = types.StringNull()
	}

	// Uses count
	data.Uses = types.Int64Value(int64(invite.Uses))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *inviteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state inviteResourceModel

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

	// Check if channel_id changed - this is not allowed
	if plan.ChannelID.ValueString() != state.ChannelID.ValueString() {
		resp.Diagnostics.AddError(
			"Cannot Change Channel",
			"Discord invites cannot be moved to a different channel. Delete this invite and create a new one for the new channel.",
		)
		return
	}

	// Get the old invite code
	code := state.Code.ValueString()
	if code == "" {
		code = state.ID.ValueString()
	}

	if code == "" {
		resp.Diagnostics.AddError(
			"Missing Invite Code",
			"The invite code is missing from state.",
		)
		return
	}

	// Check if any immutable fields changed
	oldMaxAge := state.MaxAge.ValueInt64()
	newMaxAge := oldMaxAge
	if !plan.MaxAge.IsNull() && !plan.MaxAge.IsUnknown() {
		newMaxAge = plan.MaxAge.ValueInt64()
	}

	oldMaxUses := state.MaxUses.ValueInt64()
	newMaxUses := oldMaxUses
	if !plan.MaxUses.IsNull() && !plan.MaxUses.IsUnknown() {
		newMaxUses = plan.MaxUses.ValueInt64()
	}

	oldTemporary := state.Temporary.ValueBool()
	newTemporary := oldTemporary
	if !plan.Temporary.IsNull() && !plan.Temporary.IsUnknown() {
		newTemporary = plan.Temporary.ValueBool()
	}

	oldUnique := state.Unique.ValueBool()
	newUnique := oldUnique
	if !plan.Unique.IsNull() && !plan.Unique.IsUnknown() {
		newUnique = plan.Unique.ValueBool()
	}

	// If any immutable fields changed, delete old invite and create new one
	needsRecreate := (newMaxAge != oldMaxAge) || (newMaxUses != oldMaxUses) || (newTemporary != oldTemporary) || (newUnique != oldUnique)

	if needsRecreate {
		// Delete the old invite
		_, err := r.client.InviteDelete(code)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Could Not Delete Old Invite",
				fmt.Sprintf("Unable to delete old invite %s: %s. Continuing with creation of new invite.", code, err.Error()),
			)
		}

		// Create new invite with new settings (reuse Create logic)
		channelID := plan.ChannelID.ValueString()

		// Set max_age
		maxAge := int64(86400)
		if !plan.MaxAge.IsNull() && !plan.MaxAge.IsUnknown() {
			maxAge = plan.MaxAge.ValueInt64()
		}

		// Set max_uses
		maxUses := int64(0)
		if !plan.MaxUses.IsNull() && !plan.MaxUses.IsUnknown() {
			maxUses = plan.MaxUses.ValueInt64()
		}

		// Set temporary
		temporary := false
		if !plan.Temporary.IsNull() && !plan.Temporary.IsUnknown() {
			temporary = plan.Temporary.ValueBool()
		}

		// Set unique
		unique := false
		if !plan.Unique.IsNull() && !plan.Unique.IsUnknown() {
			unique = plan.Unique.ValueBool()
		}

		// Create the new invite
		invite, err := r.client.ChannelInviteCreate(channelID, discordgo.Invite{
			MaxAge:    int(maxAge),
			MaxUses:   int(maxUses),
			Temporary: temporary,
			Unique:    unique,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating New Invite",
				fmt.Sprintf("Unable to create new invite for channel %s: %s", channelID, err.Error()),
			)
			return
		}

		// Populate state with new invite data
		data := plan
		data.ID = types.StringValue(invite.Code)
		data.Code = types.StringValue(invite.Code)
		data.ChannelID = types.StringValue(channelID)
		data.MaxAge = types.Int64Value(int64(invite.MaxAge))
		data.MaxUses = types.Int64Value(int64(invite.MaxUses))
		data.Temporary = types.BoolValue(invite.Temporary)
		data.Unique = types.BoolValue(invite.Unique)
		data.URL = types.StringValue(fmt.Sprintf("https://discord.gg/%s", invite.Code))

		if !invite.CreatedAt.IsZero() {
			data.CreatedAt = types.StringValue(invite.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
		} else {
			data.CreatedAt = types.StringNull()
		}

		if invite.ExpiresAt != nil && !invite.ExpiresAt.IsZero() {
			data.ExpiresAt = types.StringValue(invite.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
		} else {
			data.ExpiresAt = types.StringNull()
		}

		data.Uses = types.Int64Value(int64(invite.Uses))

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// No immutable fields changed, just read current state
	invite, err := r.client.InviteWithCounts(code)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invite Not Found",
			fmt.Sprintf("Invite %s was not found. It may have been deleted or expired.", code),
		)
		return
	}

	// Update state with current invite data
	data := plan
	data.ID = types.StringValue(invite.Code)
	data.Code = types.StringValue(invite.Code)
	data.ChannelID = types.StringValue(plan.ChannelID.ValueString())
	data.URL = types.StringValue(fmt.Sprintf("https://discord.gg/%s", invite.Code))
	data.MaxAge = types.Int64Value(int64(invite.MaxAge))
	data.MaxUses = types.Int64Value(int64(invite.MaxUses))
	data.Temporary = types.BoolValue(invite.Temporary)
	data.Unique = types.BoolValue(invite.Unique)

	if !invite.CreatedAt.IsZero() {
		data.CreatedAt = types.StringValue(invite.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.CreatedAt = types.StringNull()
	}

	if invite.ExpiresAt != nil && !invite.ExpiresAt.IsZero() {
		data.ExpiresAt = types.StringValue(invite.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.ExpiresAt = types.StringNull()
	}

	data.Uses = types.Int64Value(int64(invite.Uses))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *inviteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data inviteResourceModel

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

	code := data.Code.ValueString()
	if code == "" {
		// Fallback to ID if code is not set
		code = data.ID.ValueString()
	}

	if code == "" {
		resp.Diagnostics.AddError(
			"Missing Invite Code",
			"The invite code is missing from state.",
		)
		return
	}

	// Delete the invite
	_, err := r.client.InviteDelete(code)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Invite",
			fmt.Sprintf("Unable to delete invite %s: %s", code, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform.
func (r *inviteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by invite code
	code := req.ID
	if code == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"The import ID must be the invite code.",
		)
		return
	}

	// Set the code in state - Read will populate the rest
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("code"), types.StringValue(code))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(code))...)
}
