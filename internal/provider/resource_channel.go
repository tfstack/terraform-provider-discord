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
var _ resource.Resource = &channelResource{}
var _ resource.ResourceWithConfigure = &channelResource{}
var _ resource.ResourceWithImportState = &channelResource{}

// channelResource defines the resource implementation.
type channelResource struct {
	client *discordgo.Session
}

// channelResourceModel describes the resource data model.
type channelResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	GuildID    types.String `tfsdk:"guild_id"`
	CategoryID types.String `tfsdk:"category_id"`
	Position   types.Int64  `tfsdk:"position"`
}

// NewChannelResource is a helper function to simplify testing.
func NewChannelResource() resource.Resource {
	return &channelResource{}
}

// channelTypeFromString converts a string channel type to discordgo.ChannelType.
func channelTypeFromString(typeStr string) (discordgo.ChannelType, error) {
	switch typeStr {
	case "text", "":
		return discordgo.ChannelTypeGuildText, nil
	case "voice":
		return discordgo.ChannelTypeGuildVoice, nil
	case "category":
		return discordgo.ChannelTypeGuildCategory, nil
	// Note: News channels (type 5) cannot be created directly - they must be converted from text channels
	// The Discord API only accepts {0, 2, 4, 6, 13, 14, 15, 16} for channel creation
	// Note: Stage channels (type 13) cannot be created by bots - Discord API limitation
	// case "stage":
	//	return discordgo.ChannelTypeGuildStageVoice, nil
	// Note: Forum channels (type 15) cannot be created by bots - Discord API limitation
	// case "forum":
	//	return discordgo.ChannelTypeGuildForum, nil
	case "media":
		return discordgo.ChannelTypeGuildMedia, nil
	case "directory":
		return discordgo.ChannelTypeGuildDirectory, nil
	default:
		return discordgo.ChannelTypeGuildText, fmt.Errorf("invalid channel type: %s. Valid values are: text, voice, category, media, directory. Note: News, stage, and forum channels cannot be created by bots - they must be created manually or via user OAuth2 tokens", typeStr)
	}
}

// channelTypeToString converts discordgo.ChannelType to a string.
func channelTypeToString(channelType discordgo.ChannelType) string {
	switch channelType {
	case discordgo.ChannelTypeGuildText:
		return "text"
	case discordgo.ChannelTypeGuildVoice:
		return "voice"
	case discordgo.ChannelTypeGuildCategory:
		return "category"
	case discordgo.ChannelTypeGuildNews:
		return "news" // Can be read but not created directly
	case discordgo.ChannelTypeGuildStageVoice:
		return "stage"
	case discordgo.ChannelTypeGuildForum:
		return "forum"
	case discordgo.ChannelTypeGuildMedia:
		return "media"
	case discordgo.ChannelTypeGuildDirectory:
		return "directory"
	default:
		return "text" // Default fallback
	}
}

// Metadata returns the resource type name.
func (r *channelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel"
}

// Schema defines the schema for the resource.
func (r *channelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a Discord channel in a guild (server).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the channel.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the channel. Must be 1-100 characters.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the channel. Valid values: \"text\" (text chat channel), \"voice\" (voice channel), \"category\" (organizational container), \"media\" (media channel), \"directory\" (directory channel). Defaults to \"text\". Note: News, stage, and forum channels cannot be created by bots - they must be created manually in Discord or via user OAuth2 tokens.",
				Optional:    true,
				Computed:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) where the channel will be created.",
				Required:    true,
			},
			"category_id": schema.StringAttribute{
				Description: "The ID of the parent category channel. If provided, the channel will be created under this category. Note: Category channels (type=\"category\") cannot have a parent category.",
				Optional:    true,
			},
			"position": schema.Int64Attribute{
				Description: "The position of the channel in the channel list. Lower numbers appear higher in the list.",
				Optional:    true,
			},
		},
	}
}

// Configure sets up the resource with the provider's configured client.
func (r *channelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *channelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data channelResourceModel

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
			"Missing Channel Name",
			"The name attribute is required.",
		)
		return
	}

	// Determine channel type (default to text channel)
	channelType := discordgo.ChannelTypeGuildText
	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		typeStr := data.Type.ValueString()
		var err error
		channelType, err = channelTypeFromString(typeStr)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Channel Type",
				err.Error(),
			)
			return
		}
	}

	// Validate: category channels cannot have a parent category
	if channelType == discordgo.ChannelTypeGuildCategory {
		if !data.CategoryID.IsNull() && !data.CategoryID.IsUnknown() {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"Category channels cannot be placed under another category. Remove the category_id attribute when creating a category channel.",
			)
			return
		}
	}

	// Validate: News channels cannot be created directly - they must be converted from text channels
	if channelType == discordgo.ChannelTypeGuildNews {
		resp.Diagnostics.AddError(
			"Invalid Channel Type",
			"News channels cannot be created directly. Create a text channel first, then convert it to a news channel in Discord or via the API.",
		)
		return
	}

	// Validate: Stage and forum channels cannot be created by bots (Discord API limitation)
	if channelType == discordgo.ChannelTypeGuildStageVoice {
		resp.Diagnostics.AddError(
			"Invalid Channel Type",
			"Stage channels cannot be created by bots via the Discord API. This is a Discord API limitation (error code 50024). Stage channels must be created manually in Discord or through user OAuth2 tokens.",
		)
		return
	}

	if channelType == discordgo.ChannelTypeGuildForum {
		resp.Diagnostics.AddError(
			"Invalid Channel Type",
			"Forum channels cannot be created by bots via the Discord API. This is a Discord API limitation (error code 50024). Forum channels must be created manually in Discord or through user OAuth2 tokens.",
		)
		return
	}

	// Prepare channel creation data
	channelData := discordgo.GuildChannelCreateData{
		Name: name,
		Type: channelType,
	}

	// Set parent category if provided (only for non-category channels)
	if !data.CategoryID.IsNull() && !data.CategoryID.IsUnknown() {
		channelData.ParentID = data.CategoryID.ValueString()
	}

	// Create the channel
	channel, err := r.client.GuildChannelCreateComplex(guildID, channelData)
	if err != nil {
		// Provide more helpful error messages for specific channel types
		errorMsg := fmt.Sprintf("Unable to create channel %s in guild %s: %s", name, guildID, err.Error())

		// Check for specific channel type errors
		if channelType == discordgo.ChannelTypeGuildDirectory {
			errorMsg += "\n\nNote: Directory channels are only available in Community servers."
		}

		resp.Diagnostics.AddError(
			"Error Creating Channel",
			errorMsg,
		)
		return
	}

	// Verify channel was created successfully
	if channel == nil {
		resp.Diagnostics.AddError(
			"Channel Creation Failed",
			fmt.Sprintf("Channel creation API call succeeded but returned nil channel for %s in guild %s. This may indicate a Discord API issue.", name, guildID),
		)
		return
	}

	// Verify channel ID is set
	if channel.ID == "" {
		resp.Diagnostics.AddError(
			"Invalid Channel Response",
			fmt.Sprintf("Channel was created but has no ID. Channel name: %s, Guild ID: %s", name, guildID),
		)
		return
	}

	// Store original plan values before we modify data
	planPosition := data.Position
	planCategoryID := data.CategoryID

	// Set position if provided
	if !planPosition.IsNull() && !planPosition.IsUnknown() {
		position := int(planPosition.ValueInt64())
		updatedChannel, err := r.client.ChannelEditComplex(channel.ID, &discordgo.ChannelEdit{
			Position: &position,
		})
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error Setting Channel Position",
				fmt.Sprintf("Channel was created but position could not be set: %s", err.Error()),
			)
		} else {
			// Use updated channel if position was set successfully
			channel = updatedChannel
		}
	}

	// Update model with created channel data
	data.ID = types.StringValue(channel.ID)
	data.Name = types.StringValue(channel.Name)
	data.Type = types.StringValue(channelTypeToString(channel.Type))
	data.GuildID = types.StringValue(channel.GuildID)

	// Only set position if it was specified in the plan
	if !planPosition.IsNull() && !planPosition.IsUnknown() {
		data.Position = types.Int64Value(int64(channel.Position))
	} else {
		data.Position = types.Int64Null()
	}

	// Set category_id based on plan and actual channel state
	if !planCategoryID.IsNull() && !planCategoryID.IsUnknown() {
		// User specified a category, use what Discord returned
		if channel.ParentID != "" {
			data.CategoryID = types.StringValue(channel.ParentID)
		} else {
			data.CategoryID = types.StringNull()
		}
	} else {
		// User didn't specify category, keep it null
		data.CategoryID = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		// If state save fails, we should still report that the channel was created
		// but there was an issue saving state. However, this is rare.
		resp.Diagnostics.AddWarning(
			"State Save Warning",
			fmt.Sprintf("Channel %s (ID: %s) was created successfully in Discord, but there was an issue saving it to Terraform state. The channel exists in Discord but may not be tracked properly.", channel.Name, channel.ID),
		)
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *channelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data channelResourceModel

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

	channelID := data.ID.ValueString()
	if channelID == "" {
		resp.Diagnostics.AddError(
			"Missing Channel ID",
			"The channel ID is missing from state.",
		)
		return
	}

	// Fetch the channel
	channel, err := r.client.Channel(channelID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Channel",
			fmt.Sprintf("Unable to fetch channel %s: %s", channelID, err.Error()),
		)
		return
	}

	// Store original state values to preserve nulls
	originalPosition := data.Position
	originalCategoryID := data.CategoryID

	// Update model with channel data
	data.ID = types.StringValue(channel.ID)
	data.Name = types.StringValue(channel.Name)
	data.Type = types.StringValue(channelTypeToString(channel.Type))
	data.GuildID = types.StringValue(channel.GuildID)

	// Only update position if it was previously set in state
	if !originalPosition.IsNull() && !originalPosition.IsUnknown() {
		data.Position = types.Int64Value(int64(channel.Position))
	} else {
		data.Position = types.Int64Null()
	}

	// Only update category_id if it was previously set in state
	if !originalCategoryID.IsNull() && !originalCategoryID.IsUnknown() {
		if channel.ParentID != "" {
			data.CategoryID = types.StringValue(channel.ParentID)
		} else {
			data.CategoryID = types.StringNull()
		}
	} else {
		data.CategoryID = types.StringNull()
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *channelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state channelResourceModel

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

	channelID := state.ID.ValueString()
	if channelID == "" {
		resp.Diagnostics.AddError(
			"Missing Channel ID",
			"The channel ID is missing from state.",
		)
		return
	}

	// Prepare channel edit data
	edit := &discordgo.ChannelEdit{}
	hasChanges := false

	// Update name if changed
	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		edit.Name = name
		hasChanges = true
	}

	// Update category if changed
	if !plan.CategoryID.Equal(state.CategoryID) {
		if !plan.CategoryID.IsNull() && !plan.CategoryID.IsUnknown() {
			parentID := plan.CategoryID.ValueString()
			edit.ParentID = parentID
		} else {
			// Setting to empty string removes from category
			edit.ParentID = ""
		}
		hasChanges = true
	}

	// Update position if changed
	if !plan.Position.Equal(state.Position) {
		if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
			position := int(plan.Position.ValueInt64())
			edit.Position = &position
			hasChanges = true
		}
	}

	// Update type if changed (note: Discord doesn't allow changing channel type, but we'll try)
	if !plan.Type.Equal(state.Type) {
		resp.Diagnostics.AddWarning(
			"Channel Type Change",
			"Discord does not support changing channel type. The type change will be ignored.",
		)
	}

	// Apply updates if any
	if hasChanges {
		channel, err := r.client.ChannelEditComplex(channelID, edit)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Channel",
				fmt.Sprintf("Unable to update channel %s: %s", channelID, err.Error()),
			)
			return
		}

		// Update state with latest channel data
		plan.ID = types.StringValue(channel.ID)
		plan.Name = types.StringValue(channel.Name)
		plan.Type = types.StringValue(channelTypeToString(channel.Type))
		plan.GuildID = types.StringValue(channel.GuildID)
		plan.Position = types.Int64Value(int64(channel.Position))

		if channel.ParentID != "" {
			plan.CategoryID = types.StringValue(channel.ParentID)
		} else {
			plan.CategoryID = types.StringNull()
		}
	} else {
		// No changes, keep plan as is but ensure type is set correctly
		plan.ID = state.ID
		plan.GuildID = state.GuildID
		// Type should already be set from plan, but ensure it's correct
		if plan.Type.IsNull() || plan.Type.IsUnknown() {
			plan.Type = state.Type
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *channelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data channelResourceModel

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

	channelID := data.ID.ValueString()
	if channelID == "" {
		return // Nothing to delete
	}

	// Delete the channel
	_, err := r.client.ChannelDelete(channelID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Channel",
			fmt.Sprintf("Unable to delete channel %s: %s", channelID, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform state.
func (r *channelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the channel ID
	channelID := req.ID

	// Ensure client is configured
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Discord Client Not Configured",
			"The Discord client was not properly configured. This is a provider error.",
		)
		return
	}

	// Fetch the channel to populate state
	channel, err := r.client.Channel(channelID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Channel",
			fmt.Sprintf("Unable to fetch channel %s: %s", channelID, err.Error()),
		)
		return
	}

	// Create a model with the channel data
	var data channelResourceModel
	data.ID = types.StringValue(channel.ID)
	data.Name = types.StringValue(channel.Name)
	data.Type = types.StringValue(channelTypeToString(channel.Type))
	data.GuildID = types.StringValue(channel.GuildID)
	data.Position = types.Int64Value(int64(channel.Position))

	if channel.ParentID != "" {
		data.CategoryID = types.StringValue(channel.ParentID)
	} else {
		data.CategoryID = types.StringNull()
	}

	// Save the imported state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
