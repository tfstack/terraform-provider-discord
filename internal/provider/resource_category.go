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
var _ resource.Resource = &categoryResource{}
var _ resource.ResourceWithConfigure = &categoryResource{}
var _ resource.ResourceWithImportState = &categoryResource{}

// categoryResource defines the resource implementation.
type categoryResource struct {
	client *discordgo.Session
}

// categoryResourceModel describes the resource data model.
type categoryResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	GuildID  types.String `tfsdk:"guild_id"`
	Position types.Int64  `tfsdk:"position"`
}

// NewCategoryResource is a helper function to simplify testing.
func NewCategoryResource() resource.Resource {
	return &categoryResource{}
}

// Metadata returns the resource type name.
func (r *categoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_category"
}

// Schema defines the schema for the resource.
func (r *categoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a Discord category channel in a guild (server). Category channels are organizational containers that group other channels together.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the category channel.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the category channel. Must be 1-100 characters.",
				Required:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) where the category will be created.",
				Required:    true,
			},
			"position": schema.Int64Attribute{
				Description: "The position of the category in the channel list. Lower numbers appear higher in the list.",
				Optional:    true,
			},
		},
	}
}

// Configure sets up the resource with the provider's configured client.
func (r *categoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *categoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data categoryResourceModel

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
			"Missing Category Name",
			"The name attribute is required.",
		)
		return
	}

	// Prepare category creation data (category channels always have type GuildCategory)
	channelData := discordgo.GuildChannelCreateData{
		Name: name,
		Type: discordgo.ChannelTypeGuildCategory,
	}

	// Create the category channel
	channel, err := r.client.GuildChannelCreateComplex(guildID, channelData)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Category",
			fmt.Sprintf("Unable to create category %s in guild %s: %s", name, guildID, err.Error()),
		)
		return
	}

	// Verify category was created successfully
	if channel == nil {
		resp.Diagnostics.AddError(
			"Category Creation Failed",
			fmt.Sprintf("Category creation API call succeeded but returned nil channel for %s in guild %s. This may indicate a Discord API issue.", name, guildID),
		)
		return
	}

	// Verify channel ID is set
	if channel.ID == "" {
		resp.Diagnostics.AddError(
			"Invalid Category Response",
			fmt.Sprintf("Category was created but has no ID. Category name: %s, Guild ID: %s", name, guildID),
		)
		return
	}

	// Verify it's actually a category
	if channel.Type != discordgo.ChannelTypeGuildCategory {
		resp.Diagnostics.AddError(
			"Invalid Channel Type",
			fmt.Sprintf("Created channel is not a category (type: %d). Expected category type: %d", channel.Type, discordgo.ChannelTypeGuildCategory),
		)
		return
	}

	// Store original plan value for position
	planPosition := data.Position

	// Set position if provided
	if !planPosition.IsNull() && !planPosition.IsUnknown() {
		position := int(planPosition.ValueInt64())
		updatedChannel, err := r.client.ChannelEditComplex(channel.ID, &discordgo.ChannelEdit{
			Position: &position,
		})
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error Setting Category Position",
				fmt.Sprintf("Category was created but position could not be set: %s", err.Error()),
			)
		} else {
			// Use updated channel if position was set successfully
			channel = updatedChannel
		}
	}

	// Update model with created category data
	data.ID = types.StringValue(channel.ID)
	data.Name = types.StringValue(channel.Name)
	data.GuildID = types.StringValue(channel.GuildID)

	// Only set position if it was specified in the plan
	if !planPosition.IsNull() && !planPosition.IsUnknown() {
		data.Position = types.Int64Value(int64(channel.Position))
	} else {
		data.Position = types.Int64Null()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		// If state save fails, we should still report that the category was created
		// but there was an issue saving state. However, this is rare.
		resp.Diagnostics.AddWarning(
			"State Save Warning",
			fmt.Sprintf("Category %s (ID: %s) was created successfully in Discord, but there was an issue saving it to Terraform state. The category exists in Discord but may not be tracked properly.", channel.Name, channel.ID),
		)
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *categoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data categoryResourceModel

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

	categoryID := data.ID.ValueString()
	if categoryID == "" {
		resp.Diagnostics.AddError(
			"Missing Category ID",
			"The category ID is missing from state.",
		)
		return
	}

	// Fetch the channel
	channel, err := r.client.Channel(categoryID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Category",
			fmt.Sprintf("Unable to fetch category %s: %s", categoryID, err.Error()),
		)
		return
	}

	// Verify it's actually a category channel
	if channel.Type != discordgo.ChannelTypeGuildCategory {
		resp.Diagnostics.AddError(
			"Invalid Channel Type",
			fmt.Sprintf("Channel %s is not a category channel (type: %d)", categoryID, channel.Type),
		)
		return
	}

	// Store original state value to preserve nulls
	originalPosition := data.Position

	// Update model with category data
	data.ID = types.StringValue(channel.ID)
	data.Name = types.StringValue(channel.Name)
	data.GuildID = types.StringValue(channel.GuildID)

	// Only update position if it was previously set in state
	if !originalPosition.IsNull() && !originalPosition.IsUnknown() {
		data.Position = types.Int64Value(int64(channel.Position))
	} else {
		data.Position = types.Int64Null()
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *categoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state categoryResourceModel

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

	categoryID := state.ID.ValueString()
	if categoryID == "" {
		resp.Diagnostics.AddError(
			"Missing Category ID",
			"The category ID is missing from state.",
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

	// Update position if changed
	if !plan.Position.Equal(state.Position) {
		if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
			position := int(plan.Position.ValueInt64())
			edit.Position = &position
			hasChanges = true
		}
	}

	// Apply updates if any
	if hasChanges {
		channel, err := r.client.ChannelEditComplex(categoryID, edit)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Category",
				fmt.Sprintf("Unable to update category %s: %s", categoryID, err.Error()),
			)
			return
		}

		// Update state with latest category data
		plan.ID = types.StringValue(channel.ID)
		plan.Name = types.StringValue(channel.Name)
		plan.GuildID = types.StringValue(channel.GuildID)

		// Only set position if it was specified in the plan
		if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
			plan.Position = types.Int64Value(int64(channel.Position))
		} else {
			plan.Position = types.Int64Null()
		}
	} else {
		// No changes, keep plan as is
		plan.ID = state.ID
		plan.GuildID = state.GuildID
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *categoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data categoryResourceModel

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

	categoryID := data.ID.ValueString()
	if categoryID == "" {
		return // Nothing to delete
	}

	// Delete the category channel
	// Note: Deleting a category will also delete all channels within it
	_, err := r.client.ChannelDelete(categoryID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Category",
			fmt.Sprintf("Unable to delete category %s: %s", categoryID, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform state.
func (r *categoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the category channel ID
	categoryID := req.ID

	// Ensure client is configured
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Discord Client Not Configured",
			"The Discord client was not properly configured. This is a provider error.",
		)
		return
	}

	// Fetch the channel to verify it's a category and populate state
	channel, err := r.client.Channel(categoryID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Category",
			fmt.Sprintf("Unable to fetch category %s: %s", categoryID, err.Error()),
		)
		return
	}

	// Verify it's actually a category channel
	if channel.Type != discordgo.ChannelTypeGuildCategory {
		resp.Diagnostics.AddError(
			"Invalid Channel Type",
			fmt.Sprintf("Channel %s is not a category channel (type: %d). Use discord_channel resource instead.", categoryID, channel.Type),
		)
		return
	}

	// Create a model with the category data
	var data categoryResourceModel
	data.ID = types.StringValue(channel.ID)
	data.Name = types.StringValue(channel.Name)
	data.GuildID = types.StringValue(channel.GuildID)
	data.Position = types.Int64Value(int64(channel.Position))

	// Save the imported state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
