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
var _ resource.Resource = &channelPermissionResource{}
var _ resource.ResourceWithConfigure = &channelPermissionResource{}
var _ resource.ResourceWithImportState = &channelPermissionResource{}

// channelPermissionResource defines the resource implementation.
type channelPermissionResource struct {
	client *discordgo.Session
}

// channelPermissionResourceModel describes the resource data model.
type channelPermissionResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ChannelID   types.String `tfsdk:"channel_id"`
	Type        types.String `tfsdk:"type"`
	OverwriteID types.String `tfsdk:"overwrite_id"`
	Allow       types.Int64  `tfsdk:"allow"`
	Deny        types.Int64  `tfsdk:"deny"`
}

// NewChannelPermissionResource is a helper function to simplify testing.
func NewChannelPermissionResource() resource.Resource {
	return &channelPermissionResource{}
}

// Metadata returns the resource type name.
func (r *channelPermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_permission"
}

// Schema defines the schema for the resource.
func (r *channelPermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages Discord channel permission overwrites. Permission overwrites allow you to grant or deny specific permissions for a role or member in a channel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A unique identifier for this permission overwrite, composed of channel_id:overwrite_id.",
				Computed:    true,
			},
			"channel_id": schema.StringAttribute{
				Description: "The ID of the channel to set permissions for.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of permission overwrite. Valid values: \"role\" (for a role) or \"member\" (for a user/member).",
				Required:    true,
			},
			"overwrite_id": schema.StringAttribute{
				Description: "The ID of the role or member to set permissions for. Must match the type (role ID for type=\"role\", user ID for type=\"member\").",
				Required:    true,
			},
			"allow": schema.Int64Attribute{
				Description: "The permission bits to allow. Use permission constants or calculate from Discord permission flags.",
				Required:    true,
			},
			"deny": schema.Int64Attribute{
				Description: "The permission bits to deny. Use permission constants or calculate from Discord permission flags. Defaults to 0 if not specified.",
				Optional:    true,
			},
		},
	}
}

// Configure sets up the resource with the provider's configured client.
func (r *channelPermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *channelPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data channelPermissionResourceModel

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

	typeStr := data.Type.ValueString()
	if typeStr == "" {
		resp.Diagnostics.AddError(
			"Missing Type",
			"The type attribute is required. Valid values: \"role\" or \"member\".",
		)
		return
	}

	// Validate type
	var overwriteType discordgo.PermissionOverwriteType
	switch typeStr {
	case "role":
		overwriteType = discordgo.PermissionOverwriteTypeRole
	case "member":
		overwriteType = discordgo.PermissionOverwriteTypeMember
	default:
		resp.Diagnostics.AddError(
			"Invalid Type",
			fmt.Sprintf("Invalid type '%s'. Valid values are: \"role\", \"member\".", typeStr),
		)
		return
	}

	overwriteID := data.OverwriteID.ValueString()
	if overwriteID == "" {
		resp.Diagnostics.AddError(
			"Missing Overwrite ID",
			"The overwrite_id attribute is required.",
		)
		return
	}

	// Validate overwrite ID is not a placeholder
	placeholderIDs := []string{
		"987654321098765432",
		"123456789012345678",
		"111111111111111111",
		"000000000000000000",
	}
	for _, placeholder := range placeholderIDs {
		if overwriteID == placeholder {
			resp.Diagnostics.AddError(
				"Invalid Overwrite ID",
				fmt.Sprintf("The %s ID '%s' is a placeholder. Please provide a valid %s ID from your Discord server. Get the ID by enabling Developer Mode in Discord and right-clicking the %s.", typeStr, overwriteID, typeStr, typeStr),
			)
			return
		}
	}

	allow := int64(0)
	if !data.Allow.IsNull() && !data.Allow.IsUnknown() {
		allow = data.Allow.ValueInt64()
	}

	deny := int64(0)
	if !data.Deny.IsNull() && !data.Deny.IsUnknown() {
		deny = data.Deny.ValueInt64()
	}

	// Create the permission overwrite
	overwrite := &discordgo.PermissionOverwrite{
		ID:    overwriteID,
		Type:  overwriteType,
		Allow: allow,
		Deny:  deny,
	}

	// Get current channel to preserve existing overwrites
	channel, err := r.client.Channel(channelID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Channel",
			fmt.Sprintf("Unable to fetch channel %s: %s", channelID, err.Error()),
		)
		return
	}

	// Build new permission overwrites list
	overwrites := make([]*discordgo.PermissionOverwrite, 0, len(channel.PermissionOverwrites)+1)

	// Copy existing overwrites (excluding the one we're updating)
	for _, existing := range channel.PermissionOverwrites {
		if existing.ID != overwriteID || existing.Type != overwriteType {
			overwrites = append(overwrites, existing)
		}
	}

	// Add our new/updated overwrite
	overwrites = append(overwrites, overwrite)

	// Update channel with new permission overwrites
	edit := &discordgo.ChannelEdit{
		PermissionOverwrites: overwrites,
	}

	updatedChannel, err := r.client.ChannelEditComplex(channelID, edit)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Setting Channel Permission",
			fmt.Sprintf("Unable to set permission for %s %s on channel %s: %s", typeStr, overwriteID, channelID, err.Error()),
		)
		return
	}

	// Find the overwrite we just created to get the final state
	var finalOverwrite *discordgo.PermissionOverwrite
	for _, ow := range updatedChannel.PermissionOverwrites {
		if ow.ID == overwriteID && ow.Type == overwriteType {
			finalOverwrite = ow
			break
		}
	}

	// If not found in immediate response, fetch channel again to verify
	if finalOverwrite == nil {
		// Fetch the channel again to get the latest state
		refreshedChannel, err := r.client.Channel(channelID)
		if err == nil {
			for _, ow := range refreshedChannel.PermissionOverwrites {
				if ow.ID == overwriteID && ow.Type == overwriteType {
					finalOverwrite = ow
					break
				}
			}
		}
	}

	// If still not found, the role/user doesn't exist - fail the operation
	if finalOverwrite == nil {
		resp.Diagnostics.AddError(
			"Permission Overwrite Not Created",
			fmt.Sprintf("The permission overwrite for %s '%s' was not created. This means the %s ID '%s' doesn't exist in your Discord server.\n\nVerify the %s exists in your Discord server and that the bot has permission to view it. Get valid IDs by enabling Developer Mode in Discord and right-clicking the %s.", typeStr, overwriteID, typeStr, overwriteID, typeStr, typeStr),
		)
		return
	}

	// Update model with created permission data from API response
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", channelID, overwriteID))
	data.ChannelID = types.StringValue(channelID)
	data.Type = types.StringValue(typeStr)
	data.OverwriteID = types.StringValue(overwriteID)
	data.Allow = types.Int64Value(int64(finalOverwrite.Allow))
	data.Deny = types.Int64Value(int64(finalOverwrite.Deny))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *channelPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data channelPermissionResourceModel

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
	if channelID == "" {
		resp.Diagnostics.AddError(
			"Missing Channel ID",
			"The channel ID is missing from state.",
		)
		return
	}

	typeStr := data.Type.ValueString()
	overwriteID := data.OverwriteID.ValueString()

	// Validate type
	var overwriteType discordgo.PermissionOverwriteType
	switch typeStr {
	case "role":
		overwriteType = discordgo.PermissionOverwriteTypeRole
	case "member":
		overwriteType = discordgo.PermissionOverwriteTypeMember
	default:
		resp.Diagnostics.AddError(
			"Invalid Type",
			fmt.Sprintf("Invalid type '%s' in state. Valid values are: \"role\", \"member\".", typeStr),
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

	// Find the permission overwrite
	var overwrite *discordgo.PermissionOverwrite
	for _, ow := range channel.PermissionOverwrites {
		if ow.ID == overwriteID && ow.Type == overwriteType {
			overwrite = ow
			break
		}
	}

	if overwrite == nil {
		// Permission overwrite doesn't exist - mark resource for deletion
		resp.State.RemoveResource(ctx)
		return
	}

	// Update model with permission data
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", channelID, overwriteID))
	data.ChannelID = types.StringValue(channelID)
	data.Type = types.StringValue(typeStr)
	data.OverwriteID = types.StringValue(overwriteID)
	data.Allow = types.Int64Value(int64(overwrite.Allow))
	data.Deny = types.Int64Value(int64(overwrite.Deny))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *channelPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state channelPermissionResourceModel

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

	channelID := plan.ChannelID.ValueString()
	typeStr := plan.Type.ValueString()
	overwriteID := plan.OverwriteID.ValueString()

	// Validate type
	var overwriteType discordgo.PermissionOverwriteType
	switch typeStr {
	case "role":
		overwriteType = discordgo.PermissionOverwriteTypeRole
	case "member":
		overwriteType = discordgo.PermissionOverwriteTypeMember
	default:
		resp.Diagnostics.AddError(
			"Invalid Type",
			fmt.Sprintf("Invalid type '%s'. Valid values are: \"role\", \"member\".", typeStr),
		)
		return
	}

	allow := int64(0)
	if !plan.Allow.IsNull() && !plan.Allow.IsUnknown() {
		allow = plan.Allow.ValueInt64()
	}

	deny := int64(0)
	if !plan.Deny.IsNull() && !plan.Deny.IsUnknown() {
		deny = plan.Deny.ValueInt64()
	}

	// Get current channel to preserve existing overwrites
	channel, err := r.client.Channel(channelID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Channel",
			fmt.Sprintf("Unable to fetch channel %s: %s", channelID, err.Error()),
		)
		return
	}

	// Build new permission overwrites list
	overwrites := make([]*discordgo.PermissionOverwrite, 0, len(channel.PermissionOverwrites))

	// Copy existing overwrites (excluding the one we're updating)
	for _, existing := range channel.PermissionOverwrites {
		if existing.ID != overwriteID || existing.Type != overwriteType {
			overwrites = append(overwrites, existing)
		}
	}

	// Add our updated overwrite
	overwrite := &discordgo.PermissionOverwrite{
		ID:    overwriteID,
		Type:  overwriteType,
		Allow: allow,
		Deny:  deny,
	}
	overwrites = append(overwrites, overwrite)

	// Update channel with new permission overwrites
	edit := &discordgo.ChannelEdit{
		PermissionOverwrites: overwrites,
	}

	updatedChannel, err := r.client.ChannelEditComplex(channelID, edit)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Channel Permission",
			fmt.Sprintf("Unable to update permission for %s %s on channel %s: %s", typeStr, overwriteID, channelID, err.Error()),
		)
		return
	}

	// Find the overwrite we just updated to get the final state
	var finalOverwrite *discordgo.PermissionOverwrite
	for _, ow := range updatedChannel.PermissionOverwrites {
		if ow.ID == overwriteID && ow.Type == overwriteType {
			finalOverwrite = ow
			break
		}
	}

	// If not found in immediate response, fetch channel again to verify
	if finalOverwrite == nil {
		// Fetch the channel again to get the latest state
		refreshedChannel, err := r.client.Channel(channelID)
		if err == nil {
			for _, ow := range refreshedChannel.PermissionOverwrites {
				if ow.ID == overwriteID && ow.Type == overwriteType {
					finalOverwrite = ow
					break
				}
			}
		}
	}

	// If still not found, use the values we sent (API accepted them, so they're valid)
	if finalOverwrite == nil {
		resp.Diagnostics.AddWarning(
			"Permission Overwrite Not Found in Response",
			fmt.Sprintf("The permission overwrite for %s %s was updated successfully, but was not found in the API response. Using the values that were sent.", typeStr, overwriteID),
		)
		// Use the values we sent since the API call succeeded
		plan.ID = types.StringValue(fmt.Sprintf("%s:%s", channelID, overwriteID))
		plan.ChannelID = types.StringValue(channelID)
		plan.Type = types.StringValue(typeStr)
		plan.OverwriteID = types.StringValue(overwriteID)
		plan.Allow = types.Int64Value(allow)
		plan.Deny = types.Int64Value(deny)
	} else {
		// Update state with latest permission data from API response
		plan.ID = types.StringValue(fmt.Sprintf("%s:%s", channelID, overwriteID))
		plan.ChannelID = types.StringValue(channelID)
		plan.Type = types.StringValue(typeStr)
		plan.OverwriteID = types.StringValue(overwriteID)
		plan.Allow = types.Int64Value(int64(finalOverwrite.Allow))
		plan.Deny = types.Int64Value(int64(finalOverwrite.Deny))
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *channelPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data channelPermissionResourceModel

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
	typeStr := data.Type.ValueString()
	overwriteID := data.OverwriteID.ValueString()

	// Validate type
	var overwriteType discordgo.PermissionOverwriteType
	switch typeStr {
	case "role":
		overwriteType = discordgo.PermissionOverwriteTypeRole
	case "member":
		overwriteType = discordgo.PermissionOverwriteTypeMember
	default:
		// Invalid type, but we'll try to delete anyway
		overwriteType = discordgo.PermissionOverwriteTypeRole
	}

	// Get current channel to remove the overwrite
	channel, err := r.client.Channel(channelID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Channel",
			fmt.Sprintf("Unable to fetch channel %s: %s", channelID, err.Error()),
		)
		return
	}

	// Build new permission overwrites list without the one we're deleting
	overwrites := make([]*discordgo.PermissionOverwrite, 0, len(channel.PermissionOverwrites))
	for _, existing := range channel.PermissionOverwrites {
		if existing.ID != overwriteID || existing.Type != overwriteType {
			overwrites = append(overwrites, existing)
		}
	}

	// Update channel with permission overwrites (removing the deleted one)
	edit := &discordgo.ChannelEdit{
		PermissionOverwrites: overwrites,
	}

	_, err = r.client.ChannelEditComplex(channelID, edit)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Channel Permission",
			fmt.Sprintf("Unable to delete permission for %s %s on channel %s: %s", typeStr, overwriteID, channelID, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform state.
// Import ID format: channel_id:overwrite_id:type (e.g., "123456789012345678:987654321098765432:role")
func (r *channelPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID: format is "channel_id:overwrite_id:type"
	importID := req.ID

	// Find the last colon (separating overwrite_id and type)
	lastColon := -1
	for i := len(importID) - 1; i >= 0; i-- {
		if importID[i] == ':' {
			lastColon = i
			break
		}
	}

	if lastColon == -1 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: channel_id:overwrite_id:type (e.g., '123456789012345678:987654321098765432:role')",
		)
		return
	}

	prefix := importID[:lastColon]
	typeStr := importID[lastColon+1:]

	// Find the first colon (separating channel_id and overwrite_id)
	firstColon := -1
	for i := 0; i < len(prefix); i++ {
		if prefix[i] == ':' {
			firstColon = i
			break
		}
	}

	if firstColon == -1 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: channel_id:overwrite_id:type (e.g., '123456789012345678:987654321098765432:role')",
		)
		return
	}

	channelID := prefix[:firstColon]
	overwriteID := prefix[firstColon+1:]

	// Validate type
	var overwriteType discordgo.PermissionOverwriteType
	switch typeStr {
	case "role":
		overwriteType = discordgo.PermissionOverwriteTypeRole
	case "member":
		overwriteType = discordgo.PermissionOverwriteTypeMember
	default:
		resp.Diagnostics.AddError(
			"Invalid Type",
			fmt.Sprintf("Invalid type '%s' in import ID. Valid values are: 'role', 'member'", typeStr),
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

	// Fetch the channel to get the permission overwrite
	channel, err := r.client.Channel(channelID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Channel",
			fmt.Sprintf("Unable to fetch channel %s: %s", channelID, err.Error()),
		)
		return
	}

	// Find the permission overwrite
	var overwrite *discordgo.PermissionOverwrite
	for _, ow := range channel.PermissionOverwrites {
		if ow.ID == overwriteID && ow.Type == overwriteType {
			overwrite = ow
			break
		}
	}

	if overwrite == nil {
		resp.Diagnostics.AddError(
			"Permission Overwrite Not Found",
			fmt.Sprintf("Permission overwrite for %s %s on channel %s was not found", typeStr, overwriteID, channelID),
		)
		return
	}

	// Create a model with the permission data
	var data channelPermissionResourceModel
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", channelID, overwriteID))
	data.ChannelID = types.StringValue(channelID)
	data.Type = types.StringValue(typeStr)
	data.OverwriteID = types.StringValue(overwriteID)
	data.Allow = types.Int64Value(int64(overwrite.Allow))
	data.Deny = types.Int64Value(int64(overwrite.Deny))

	// Save the imported state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
