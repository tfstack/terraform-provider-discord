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
var _ resource.Resource = &messageResource{}
var _ resource.ResourceWithConfigure = &messageResource{}
var _ resource.ResourceWithImportState = &messageResource{}

// messageResource defines the resource implementation.
type messageResource struct {
	client *discordgo.Session
}

// messageResourceModel describes the resource data model.
type messageResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ChannelID types.String `tfsdk:"channel_id"`
	Content   types.String `tfsdk:"content"`
	TTS       types.Bool   `tfsdk:"tts"`
	MessageID types.String `tfsdk:"message_id"`
	Timestamp types.String `tfsdk:"timestamp"`
	EditedAt  types.String `tfsdk:"edited_at"`
	Author    types.String `tfsdk:"author"`
}

// NewMessageResource is a helper function to simplify testing.
func NewMessageResource() resource.Resource {
	return &messageResource{}
}

// Metadata returns the resource type name.
func (r *messageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message"
}

// Schema defines the schema for the resource.
func (r *messageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a Discord message in a channel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the message (same as message_id).",
				Computed:    true,
			},
			"channel_id": schema.StringAttribute{
				Description: "The ID of the channel to send the message to.",
				Required:    true,
			},
			"content": schema.StringAttribute{
				Description: "The content of the message. Must be 1-2000 characters. At least one of content or embed must be provided.",
				Optional:    true,
			},
			"tts": schema.BoolAttribute{
				Description: "Whether the message should be sent as text-to-speech. Defaults to false.",
				Optional:    true,
			},
			"message_id": schema.StringAttribute{
				Description: "The ID of the message.",
				Computed:    true,
			},
			"timestamp": schema.StringAttribute{
				Description: "When the message was sent (ISO 8601 timestamp).",
				Computed:    true,
			},
			"edited_at": schema.StringAttribute{
				Description: "When the message was last edited (ISO 8601 timestamp). Null if never edited.",
				Computed:    true,
			},
			"author": schema.StringAttribute{
				Description: "The ID of the user who sent the message.",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the resource with the provider's configured client.
func (r *messageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *messageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data messageResourceModel

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

	// Validate that at least content is provided
	content := ""
	if !data.Content.IsNull() && !data.Content.IsUnknown() {
		content = data.Content.ValueString()
	}

	if content == "" {
		resp.Diagnostics.AddError(
			"Missing Message Content",
			"At least one of content must be provided. The content attribute cannot be empty.",
		)
		return
	}

	// Validate content length (Discord allows 1-2000 characters)
	if len(content) < 1 || len(content) > 2000 {
		resp.Diagnostics.AddError(
			"Invalid Message Content",
			"Message content must be between 1 and 2000 characters.",
		)
		return
	}

	// Set TTS (default to false for API call, but preserve null in state if not set)
	tts := false
	ttsSet := false
	if !data.TTS.IsNull() && !data.TTS.IsUnknown() {
		tts = data.TTS.ValueBool()
		ttsSet = true
	}

	// Send the message using ChannelMessageSendComplex to support TTS
	message, err := r.client.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: content,
		TTS:     tts,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Sending Message",
			fmt.Sprintf("Unable to send message to channel %s: %s", channelID, err.Error()),
		)
		return
	}

	// Verify message was sent successfully
	if message == nil {
		resp.Diagnostics.AddError(
			"Message Send Failed",
			fmt.Sprintf("Message send API call succeeded but returned nil message for channel %s. This may indicate a Discord API issue.", channelID),
		)
		return
	}

	// Verify message ID is set
	if message.ID == "" {
		resp.Diagnostics.AddError(
			"Invalid Message Response",
			fmt.Sprintf("Message was sent but has no ID. Channel ID: %s", channelID),
		)
		return
	}

	// Populate the model with message data from Discord
	data.ID = types.StringValue(message.ID)
	data.MessageID = types.StringValue(message.ID)
	data.ChannelID = types.StringValue(channelID)
	data.Content = types.StringValue(message.Content)

	// TTS is not returned by Discord API, preserve the plan value (null if not set, false/true if set)
	if ttsSet {
		data.TTS = types.BoolValue(tts)
	} else {
		data.TTS = types.BoolNull()
	}

	// Timestamp
	if !message.Timestamp.IsZero() {
		data.Timestamp = types.StringValue(message.Timestamp.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.Timestamp = types.StringNull()
	}

	// Edited timestamp
	if message.EditedTimestamp != nil && !message.EditedTimestamp.IsZero() {
		data.EditedAt = types.StringValue(message.EditedTimestamp.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.EditedAt = types.StringNull()
	}

	// Author
	if message.Author != nil {
		data.Author = types.StringValue(message.Author.ID)
	} else {
		data.Author = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *messageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data messageResourceModel

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

	channelID := data.ChannelID.ValueString()
	messageID := data.MessageID.ValueString()
	if messageID == "" {
		messageID = data.ID.ValueString()
	}

	if channelID == "" || messageID == "" {
		resp.Diagnostics.AddError(
			"Missing Channel or Message ID",
			"The channel_id and message_id are required to read the message.",
		)
		return
	}

	// Fetch the message
	message, err := r.client.ChannelMessage(channelID, messageID)
	if err != nil {
		// If message doesn't exist, mark as removed
		resp.Diagnostics.AddWarning(
			"Message Not Found",
			fmt.Sprintf("Message %s was not found in channel %s. It may have been deleted. Removing from state.", messageID, channelID),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Update model with message data
	data.ID = types.StringValue(message.ID)
	data.MessageID = types.StringValue(message.ID)
	data.ChannelID = types.StringValue(channelID)
	data.Content = types.StringValue(message.Content)

	// TTS - not available in read, preserve from state (keep as null if it was null)
	// Don't change it - keep whatever is in state

	// Timestamp
	if !message.Timestamp.IsZero() {
		data.Timestamp = types.StringValue(message.Timestamp.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.Timestamp = types.StringNull()
	}

	// Edited timestamp
	if message.EditedTimestamp != nil && !message.EditedTimestamp.IsZero() {
		data.EditedAt = types.StringValue(message.EditedTimestamp.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.EditedAt = types.StringNull()
	}

	// Author
	if message.Author != nil {
		data.Author = types.StringValue(message.Author.ID)
	} else {
		data.Author = types.StringNull()
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *messageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state messageResourceModel

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

	channelID := state.ChannelID.ValueString()
	messageID := state.MessageID.ValueString()
	if messageID == "" {
		messageID = state.ID.ValueString()
	}

	if channelID == "" || messageID == "" {
		resp.Diagnostics.AddError(
			"Missing Channel or Message ID",
			"The channel_id and message_id are required to update the message.",
		)
		return
	}

	// Check if channel_id changed - this is not allowed
	if plan.ChannelID.ValueString() != state.ChannelID.ValueString() {
		resp.Diagnostics.AddError(
			"Cannot Change Channel",
			"Discord messages cannot be moved to a different channel. Delete this message and create a new one in the new channel.",
		)
		return
	}

	// Get new content from plan
	newContent := ""
	if !plan.Content.IsNull() && !plan.Content.IsUnknown() {
		newContent = plan.Content.ValueString()
	}

	// Validate content length if provided
	if newContent != "" && (len(newContent) < 1 || len(newContent) > 2000) {
		resp.Diagnostics.AddError(
			"Invalid Message Content",
			"Message content must be between 1 and 2000 characters.",
		)
		return
	}

	// Update the message
	message, err := r.client.ChannelMessageEdit(channelID, messageID, newContent)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Message",
			fmt.Sprintf("Unable to update message %s in channel %s: %s", messageID, channelID, err.Error()),
		)
		return
	}

	// Update state with message data
	data := plan
	data.ID = types.StringValue(message.ID)
	data.MessageID = types.StringValue(message.ID)
	data.ChannelID = types.StringValue(channelID)
	data.Content = types.StringValue(message.Content)

	// TTS - preserve from state (not changeable after creation)
	data.TTS = state.TTS

	// Timestamp
	if !message.Timestamp.IsZero() {
		data.Timestamp = types.StringValue(message.Timestamp.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.Timestamp = types.StringNull()
	}

	// Edited timestamp
	if message.EditedTimestamp != nil && !message.EditedTimestamp.IsZero() {
		data.EditedAt = types.StringValue(message.EditedTimestamp.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.EditedAt = types.StringNull()
	}

	// Author
	if message.Author != nil {
		data.Author = types.StringValue(message.Author.ID)
	} else {
		data.Author = types.StringNull()
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *messageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data messageResourceModel

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

	channelID := data.ChannelID.ValueString()
	messageID := data.MessageID.ValueString()
	if messageID == "" {
		messageID = data.ID.ValueString()
	}

	if channelID == "" || messageID == "" {
		resp.Diagnostics.AddError(
			"Missing Channel or Message ID",
			"The channel_id and message_id are required to delete the message.",
		)
		return
	}

	// Delete the message
	err := r.client.ChannelMessageDelete(channelID, messageID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Message",
			fmt.Sprintf("Unable to delete message %s from channel %s: %s", messageID, channelID, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform.
func (r *messageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: channel_id:message_id
	importID := req.ID
	if importID == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"The import ID must be in the format 'channel_id:message_id'.",
		)
		return
	}

	// Parse the import ID - split by colon
	splitParts := make([]string, 0)
	for i := 0; i < len(importID); i++ {
		if importID[i] == ':' {
			splitParts = append(splitParts, importID[:i], importID[i+1:])
			break
		}
	}

	if len(splitParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			"The import ID must be in the format 'channel_id:message_id' (e.g., '123456789012345678:987654321098765432').",
		)
		return
	}

	channelID := splitParts[0]
	messageID := splitParts[1]

	// Set the IDs in state - Read will populate the rest
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("channel_id"), types.StringValue(channelID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("message_id"), types.StringValue(messageID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(messageID))...)
}
