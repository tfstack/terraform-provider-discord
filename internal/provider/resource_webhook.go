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
var _ resource.Resource = &webhookResource{}
var _ resource.ResourceWithConfigure = &webhookResource{}
var _ resource.ResourceWithImportState = &webhookResource{}

// webhookResource defines the resource implementation.
type webhookResource struct {
	client *discordgo.Session
}

// webhookResourceModel describes the resource data model.
type webhookResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ChannelID types.String `tfsdk:"channel_id"`
	Name      types.String `tfsdk:"name"`
	Avatar    types.String `tfsdk:"avatar"`
	Token     types.String `tfsdk:"token"`
	URL       types.String `tfsdk:"url"`
	GuildID   types.String `tfsdk:"guild_id"`
	User      types.String `tfsdk:"user"`
	Type      types.Int64  `tfsdk:"type"`
}

// NewWebhookResource is a helper function to simplify testing.
func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

// Metadata returns the resource type name.
func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

// Schema defines the schema for the resource.
func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a Discord webhook for a channel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the webhook.",
				Computed:    true,
			},
			"channel_id": schema.StringAttribute{
				Description: "The ID of the channel to create the webhook for.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the webhook. Must be 1-80 characters.",
				Required:    true,
			},
			"avatar": schema.StringAttribute{
				Description: "The avatar hash of the webhook. Can be null if no avatar is set.",
				Optional:    true,
				Computed:    true,
			},
			"token": schema.StringAttribute{
				Description: "The token of the webhook (used for sending messages). This is sensitive and should be kept secret.",
				Computed:    true,
				Sensitive:   true,
			},
			"url": schema.StringAttribute{
				Description: "The full webhook URL (https://discord.com/api/webhooks/{id}/{token}).",
				Computed:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) this webhook belongs to.",
				Computed:    true,
			},
			"user": schema.StringAttribute{
				Description: "The ID of the user who created the webhook.",
				Computed:    true,
			},
			"type": schema.Int64Attribute{
				Description: "The type of the webhook (1 = Incoming, 2 = Channel Follower).",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the resource with the provider's configured client.
func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data webhookResourceModel

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

	name := data.Name.ValueString()
	if name == "" {
		resp.Diagnostics.AddError(
			"Missing Webhook Name",
			"The name attribute is required.",
		)
		return
	}

	// Validate name length (Discord requires 1-80 characters)
	if len(name) < 1 || len(name) > 80 {
		resp.Diagnostics.AddError(
			"Invalid Webhook Name",
			"Webhook name must be between 1 and 80 characters.",
		)
		return
	}

	// Prepare avatar (can be empty string)
	avatar := ""
	if !data.Avatar.IsNull() && !data.Avatar.IsUnknown() {
		avatar = data.Avatar.ValueString()
	}

	// Create the webhook
	webhook, err := r.client.WebhookCreate(channelID, name, avatar)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Webhook",
			fmt.Sprintf("Unable to create webhook %s in channel %s: %s", name, channelID, err.Error()),
		)
		return
	}

	// Verify webhook was created successfully
	if webhook == nil {
		resp.Diagnostics.AddError(
			"Webhook Creation Failed",
			fmt.Sprintf("Webhook creation API call succeeded but returned nil webhook for %s in channel %s. This may indicate a Discord API issue.", name, channelID),
		)
		return
	}

	// Verify webhook ID is set
	if webhook.ID == "" {
		resp.Diagnostics.AddError(
			"Invalid Webhook Response",
			fmt.Sprintf("Webhook was created but has no ID. Webhook name: %s, Channel ID: %s", name, channelID),
		)
		return
	}

	// Populate the model with webhook data from Discord
	data.ID = types.StringValue(webhook.ID)
	data.ChannelID = types.StringValue(webhook.ChannelID)
	data.Name = types.StringValue(webhook.Name)

	// Avatar - use Discord response if available, otherwise preserve plan value
	// Discord may not return the avatar in the response even if we set it
	if webhook.Avatar != "" {
		data.Avatar = types.StringValue(webhook.Avatar)
	} else if !data.Avatar.IsNull() && !data.Avatar.IsUnknown() && data.Avatar.ValueString() != "" {
		// Preserve plan value if Discord didn't return avatar (may not be in response immediately)
		// The avatar was set in the plan, so keep it
	} else {
		// No avatar was set in plan and Discord didn't return one
		data.Avatar = types.StringNull()
	}

	// Token (sensitive)
	if webhook.Token != "" {
		data.Token = types.StringValue(webhook.Token)
	} else {
		data.Token = types.StringNull()
	}

	// Webhook URL
	if webhook.ID != "" && webhook.Token != "" {
		data.URL = types.StringValue(fmt.Sprintf("https://discord.com/api/webhooks/%s/%s", webhook.ID, webhook.Token))
	} else {
		data.URL = types.StringNull()
	}

	// Guild ID
	if webhook.GuildID != "" {
		data.GuildID = types.StringValue(webhook.GuildID)
	} else {
		data.GuildID = types.StringNull()
	}

	// User (creator)
	if webhook.User != nil {
		data.User = types.StringValue(webhook.User.ID)
	} else {
		data.User = types.StringNull()
	}

	// Type
	data.Type = types.Int64Value(int64(webhook.Type))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data webhookResourceModel

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

	webhookID := data.ID.ValueString()
	if webhookID == "" {
		resp.Diagnostics.AddError(
			"Missing Webhook ID",
			"The webhook ID is missing from state.",
		)
		return
	}

	// Fetch the webhook
	webhook, err := r.client.Webhook(webhookID)
	if err != nil {
		// If webhook doesn't exist, mark as removed
		resp.Diagnostics.AddWarning(
			"Webhook Not Found",
			fmt.Sprintf("Webhook %s was not found. It may have been deleted. Removing from state.", webhookID),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Update model with webhook data
	data.ID = types.StringValue(webhook.ID)
	data.ChannelID = types.StringValue(webhook.ChannelID)
	data.Name = types.StringValue(webhook.Name)

	// Avatar - use Discord response if available, otherwise preserve existing state value
	// Discord may not return the avatar in the response even if it's set
	if webhook.Avatar != "" {
		data.Avatar = types.StringValue(webhook.Avatar)
	} else if !data.Avatar.IsNull() && !data.Avatar.IsUnknown() && data.Avatar.ValueString() != "" {
		// Preserve existing state value if Discord didn't return avatar
		// Keep the existing value from state
	} else {
		// No avatar in state and Discord didn't return one
		data.Avatar = types.StringNull()
	}

	// Token - only available when creating, not when reading
	// Keep existing token from state if available, otherwise null
	if data.Token.IsNull() || data.Token.IsUnknown() {
		data.Token = types.StringNull()
	}

	// Webhook URL - can only be constructed if we have token
	if !data.Token.IsNull() && !data.Token.IsUnknown() && data.Token.ValueString() != "" {
		data.URL = types.StringValue(fmt.Sprintf("https://discord.com/api/webhooks/%s/%s", webhook.ID, data.Token.ValueString()))
	} else {
		data.URL = types.StringNull()
	}

	// Guild ID
	if webhook.GuildID != "" {
		data.GuildID = types.StringValue(webhook.GuildID)
	} else {
		data.GuildID = types.StringNull()
	}

	// User (creator)
	if webhook.User != nil {
		data.User = types.StringValue(webhook.User.ID)
	} else {
		data.User = types.StringNull()
	}

	// Type
	data.Type = types.Int64Value(int64(webhook.Type))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state webhookResourceModel

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

	webhookID := state.ID.ValueString()
	if webhookID == "" {
		resp.Diagnostics.AddError(
			"Missing Webhook ID",
			"The webhook ID is missing from state.",
		)
		return
	}

	// Check if channel_id changed - this is not allowed
	if plan.ChannelID.ValueString() != state.ChannelID.ValueString() {
		resp.Diagnostics.AddError(
			"Cannot Change Channel",
			"Discord webhooks cannot be moved to a different channel. Delete this webhook and create a new one for the new channel.",
		)
		return
	}

	// Prepare webhook update data
	name := plan.Name.ValueString()

	// Set avatar (use plan value if provided, otherwise keep existing)
	avatar := ""
	if !plan.Avatar.IsNull() && !plan.Avatar.IsUnknown() {
		avatar = plan.Avatar.ValueString()
	} else if !state.Avatar.IsNull() && !state.Avatar.IsUnknown() {
		// Keep existing avatar if not specified in plan
		avatar = state.Avatar.ValueString()
	}

	channelID := state.ChannelID.ValueString()

	// Update the webhook
	webhook, err := r.client.WebhookEdit(webhookID, name, avatar, channelID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Webhook",
			fmt.Sprintf("Unable to update webhook %s: %s", webhookID, err.Error()),
		)
		return
	}

	// Update state with webhook data
	data := plan
	data.ID = types.StringValue(webhook.ID)
	data.ChannelID = types.StringValue(webhook.ChannelID)
	data.Name = types.StringValue(webhook.Name)

	// Avatar - use Discord response if available, otherwise preserve plan value
	// Discord may not return the avatar in the response even if we set it
	if webhook.Avatar != "" {
		data.Avatar = types.StringValue(webhook.Avatar)
	} else if !plan.Avatar.IsNull() && !plan.Avatar.IsUnknown() && plan.Avatar.ValueString() != "" {
		// Preserve plan value if Discord didn't return avatar (may not be in response)
		// The avatar was set in the plan, so keep it
	} else {
		// No avatar was set in plan and Discord didn't return one
		data.Avatar = types.StringNull()
	}

	// Token - preserve from state (not returned by API)
	data.Token = state.Token

	// Webhook URL
	if !data.Token.IsNull() && !data.Token.IsUnknown() && data.Token.ValueString() != "" {
		data.URL = types.StringValue(fmt.Sprintf("https://discord.com/api/webhooks/%s/%s", webhook.ID, data.Token.ValueString()))
	} else {
		data.URL = types.StringNull()
	}

	// Guild ID
	if webhook.GuildID != "" {
		data.GuildID = types.StringValue(webhook.GuildID)
	} else {
		data.GuildID = types.StringNull()
	}

	// User (creator)
	if webhook.User != nil {
		data.User = types.StringValue(webhook.User.ID)
	} else {
		data.User = types.StringNull()
	}

	// Type
	data.Type = types.Int64Value(int64(webhook.Type))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data webhookResourceModel

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

	webhookID := data.ID.ValueString()
	if webhookID == "" {
		resp.Diagnostics.AddError(
			"Missing Webhook ID",
			"The webhook ID is missing from state.",
		)
		return
	}

	// Delete the webhook
	err := r.client.WebhookDelete(webhookID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Webhook",
			fmt.Sprintf("Unable to delete webhook %s: %s", webhookID, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform.
func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by webhook ID
	webhookID := req.ID
	if webhookID == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"The import ID must be the webhook ID.",
		)
		return
	}

	// Set the ID in state - Read will populate the rest
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(webhookID))...)
}
