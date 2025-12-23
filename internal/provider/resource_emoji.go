package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the resource type implements the required interfaces.
var _ resource.Resource = &emojiResource{}
var _ resource.ResourceWithConfigure = &emojiResource{}
var _ resource.ResourceWithImportState = &emojiResource{}

// emojiResource defines the resource implementation.
type emojiResource struct {
	client *discordgo.Session
}

// emojiResourceModel describes the resource data model.
type emojiResourceModel struct {
	ID            types.String `tfsdk:"id"`
	GuildID       types.String `tfsdk:"guild_id"`
	Name          types.String `tfsdk:"name"`
	Image         types.String `tfsdk:"image"`
	ImagePath     types.String `tfsdk:"image_path"`
	ImageURL      types.String `tfsdk:"image_url"`
	Roles         types.List   `tfsdk:"roles"`
	Animated      types.Bool   `tfsdk:"animated"`
	Managed       types.Bool   `tfsdk:"managed"`
	RequireColons types.Bool   `tfsdk:"require_colons"`
	Available     types.Bool   `tfsdk:"available"`
	User          types.String `tfsdk:"user"`
}

// NewEmojiResource is a helper function to simplify testing.
func NewEmojiResource() resource.Resource {
	return &emojiResource{}
}

// Metadata returns the resource type name.
func (r *emojiResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_emoji"
}

// Schema defines the schema for the resource.
func (r *emojiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a Discord custom emoji in a guild (server).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the emoji.",
				Computed:    true,
			},
			"guild_id": schema.StringAttribute{
				Description: "The ID of the guild (server) where the emoji will be created.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the emoji. Must be 2-32 characters and contain only alphanumeric characters and underscores.",
				Required:    true,
			},
			"image": schema.StringAttribute{
				Description: "Base64-encoded image data for the emoji. Must be a valid PNG, JPG, or GIF image. Either image, image_path, or image_url must be provided.",
				Optional:    true,
				Sensitive:   false,
			},
			"image_path": schema.StringAttribute{
				Description: "Path to a local image file for the emoji. Must be a valid PNG, JPG, or GIF image. Either image, image_path, or image_url must be provided.",
				Optional:    true,
			},
			"image_url": schema.StringAttribute{
				Description: "URL to an image for the emoji. Must be a valid PNG, JPG, or GIF image. Either image, image_path, or image_url must be provided.",
				Optional:    true,
			},
			"roles": schema.ListAttribute{
				Description: "List of role IDs that can use this emoji. If empty, all roles can use it.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"animated": schema.BoolAttribute{
				Description: "Whether the emoji is animated (read-only, determined by image format).",
				Computed:    true,
			},
			"managed": schema.BoolAttribute{
				Description: "Whether the emoji is managed by an integration (read-only).",
				Computed:    true,
			},
			"require_colons": schema.BoolAttribute{
				Description: "Whether the emoji requires colons to be used (read-only).",
				Computed:    true,
			},
			"available": schema.BoolAttribute{
				Description: "Whether the emoji is available for use (read-only).",
				Computed:    true,
			},
			"user": schema.StringAttribute{
				Description: "The ID of the user who created the emoji (read-only).",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the resource with the provider's configured client.
func (r *emojiResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// readImageData reads image data from various sources (base64, file path, or URL)
func (r *emojiResource) readImageData(data *emojiResourceModel) ([]byte, string, error) {
	// Check image (base64)
	if !data.Image.IsNull() && !data.Image.IsUnknown() {
		imageData := data.Image.ValueString()
		// Remove data URL prefix if present (data:image/png;base64,...)
		if strings.HasPrefix(imageData, "data:") {
			parts := strings.Split(imageData, ",")
			if len(parts) == 2 {
				imageData = parts[1]
			}
		}
		decoded, err := base64.StdEncoding.DecodeString(imageData)
		if err != nil {
			return nil, "", fmt.Errorf("invalid base64 image data: %w", err)
		}
		return decoded, "image/png", nil
	}

	// Check image_path (local file)
	if !data.ImagePath.IsNull() && !data.ImagePath.IsUnknown() {
		path := data.ImagePath.ValueString()
		fileData, err := os.ReadFile(path)
		if err != nil {
			return nil, "", fmt.Errorf("unable to read image file %s: %w", path, err)
		}
		// Determine content type from file extension
		contentType := "image/png"
		if strings.HasSuffix(strings.ToLower(path), ".jpg") || strings.HasSuffix(strings.ToLower(path), ".jpeg") {
			contentType = "image/jpeg"
		} else if strings.HasSuffix(strings.ToLower(path), ".gif") {
			contentType = "image/gif"
		}
		return fileData, contentType, nil
	}

	// Check image_url (remote URL)
	if !data.ImageURL.IsNull() && !data.ImageURL.IsUnknown() {
		url := data.ImageURL.ValueString()
		resp, err := http.Get(url)
		if err != nil {
			return nil, "", fmt.Errorf("unable to fetch image from URL %s: %w", url, err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("unable to fetch image from URL %s: HTTP %d", url, resp.StatusCode)
		}

		imageData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, "", fmt.Errorf("unable to read image data from URL %s: %w", url, err)
		}

		// Determine content type from response header
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "image/png"
		}
		return imageData, contentType, nil
	}

	return nil, "", fmt.Errorf("one of image, image_path, or image_url must be provided")
}

// Create creates the resource and sets the initial Terraform state.
func (r *emojiResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data emojiResourceModel

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
			"Missing Emoji Name",
			"The name attribute is required.",
		)
		return
	}

	// Validate name (Discord requires 2-32 characters, alphanumeric and underscores)
	if len(name) < 2 || len(name) > 32 {
		resp.Diagnostics.AddError(
			"Invalid Emoji Name",
			"Emoji name must be between 2 and 32 characters.",
		)
		return
	}

	// Read image data
	imageData, contentType, err := r.readImageData(&data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Image Data",
			err.Error(),
		)
		return
	}

	// Prepare emoji creation data
	emojiParams := &discordgo.EmojiParams{
		Name:  name,
		Image: fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(imageData)),
	}

	// Set roles if provided
	if !data.Roles.IsNull() && !data.Roles.IsUnknown() {
		roles := make([]string, 0)
		for _, roleValue := range data.Roles.Elements() {
			roleStr := roleValue.(types.String)
			roles = append(roles, roleStr.ValueString())
		}
		emojiParams.Roles = roles
	}

	// Create the emoji
	emoji, err := r.client.GuildEmojiCreate(guildID, emojiParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Emoji",
			fmt.Sprintf("Unable to create emoji %s in guild %s: %s", name, guildID, err.Error()),
		)
		return
	}

	// Verify emoji was created successfully
	if emoji == nil {
		resp.Diagnostics.AddError(
			"Emoji Creation Failed",
			fmt.Sprintf("Emoji creation API call succeeded but returned nil emoji for %s in guild %s. This may indicate a Discord API issue.", name, guildID),
		)
		return
	}

	// Verify emoji ID is set
	if emoji.ID == "" {
		resp.Diagnostics.AddError(
			"Invalid Emoji Response",
			fmt.Sprintf("Emoji was created but has no ID. Emoji name: %s, Guild ID: %s", name, guildID),
		)
		return
	}

	// Populate the model with emoji data from Discord
	data.ID = types.StringValue(emoji.ID)
	data.GuildID = types.StringValue(guildID)
	data.Name = types.StringValue(emoji.Name)
	data.Animated = types.BoolValue(emoji.Animated)
	data.Managed = types.BoolValue(emoji.Managed)
	data.RequireColons = types.BoolValue(emoji.RequireColons)
	data.Available = types.BoolValue(emoji.Available)

	// Convert roles slice to list
	if len(emoji.Roles) > 0 {
		roleList := make([]attr.Value, 0, len(emoji.Roles))
		for _, roleID := range emoji.Roles {
			roleList = append(roleList, types.StringValue(roleID))
		}
		data.Roles = types.ListValueMust(types.StringType, roleList)
	} else {
		data.Roles = types.ListNull(types.StringType)
	}

	// User (creator)
	if emoji.User != nil {
		data.User = types.StringValue(emoji.User.ID)
	} else {
		data.User = types.StringNull()
	}

	// Preserve image source attributes from plan if they were set (for consistency)
	// These are only used during creation/update, but we keep them in state to match the plan
	if !data.Image.IsNull() && !data.Image.IsUnknown() {
		// Keep the plan value
	} else {
		data.Image = types.StringNull()
	}

	if !data.ImagePath.IsNull() && !data.ImagePath.IsUnknown() {
		// Keep the plan value
	} else {
		data.ImagePath = types.StringNull()
	}

	if !data.ImageURL.IsNull() && !data.ImageURL.IsUnknown() {
		// Keep the plan value
	} else {
		data.ImageURL = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *emojiResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data emojiResourceModel

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
	emojiID := data.ID.ValueString()

	if guildID == "" || emojiID == "" {
		resp.Diagnostics.AddError(
			"Missing Guild or Emoji ID",
			"The guild_id and id are required to read the emoji.",
		)
		return
	}

	// Fetch the emoji
	emoji, err := r.client.GuildEmoji(guildID, emojiID)
	if err != nil {
		// If emoji doesn't exist, mark as removed
		resp.Diagnostics.AddWarning(
			"Emoji Not Found",
			fmt.Sprintf("Emoji %s was not found in guild %s. It may have been deleted. Removing from state.", emojiID, guildID),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Update model with emoji data
	data.ID = types.StringValue(emoji.ID)
	data.GuildID = types.StringValue(guildID)
	data.Name = types.StringValue(emoji.Name)
	data.Animated = types.BoolValue(emoji.Animated)
	data.Managed = types.BoolValue(emoji.Managed)
	data.RequireColons = types.BoolValue(emoji.RequireColons)
	data.Available = types.BoolValue(emoji.Available)

	// Convert roles slice to list
	if len(emoji.Roles) > 0 {
		roleList := make([]attr.Value, 0, len(emoji.Roles))
		for _, roleID := range emoji.Roles {
			roleList = append(roleList, types.StringValue(roleID))
		}
		data.Roles = types.ListValueMust(types.StringType, roleList)
	} else {
		data.Roles = types.ListNull(types.StringType)
	}

	// User (creator)
	if emoji.User != nil {
		data.User = types.StringValue(emoji.User.ID)
	} else {
		data.User = types.StringNull()
	}

	// Preserve image source attributes from existing state (they're only needed for creation/update)
	// The values are already in data from reading state, so we keep them as-is
	// Don't change them - keep whatever is in state

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *emojiResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state emojiResourceModel

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

	guildID := state.GuildID.ValueString()
	emojiID := state.ID.ValueString()

	if guildID == "" || emojiID == "" {
		resp.Diagnostics.AddError(
			"Missing Guild or Emoji ID",
			"The guild_id and id are required to update the emoji.",
		)
		return
	}

	// Check if guild_id changed - this is not allowed
	if plan.GuildID.ValueString() != state.GuildID.ValueString() {
		resp.Diagnostics.AddError(
			"Cannot Change Guild",
			"Discord emojis cannot be moved to a different guild. Delete this emoji and create a new one in the new guild.",
		)
		return
	}

	// Prepare emoji update data
	emojiParams := &discordgo.EmojiParams{
		Name: plan.Name.ValueString(),
	}

	// If image is being updated, read new image data
	if (!plan.Image.IsNull() && !plan.Image.IsUnknown()) ||
		(!plan.ImagePath.IsNull() && !plan.ImagePath.IsUnknown()) ||
		(!plan.ImageURL.IsNull() && !plan.ImageURL.IsUnknown()) {
		imageData, contentType, err := r.readImageData(&plan)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Image Data",
				err.Error(),
			)
			return
		}
		emojiParams.Image = fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(imageData))
	}

	// Set roles if provided or changed
	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		roles := make([]string, 0)
		for _, roleValue := range plan.Roles.Elements() {
			roleStr := roleValue.(types.String)
			roles = append(roles, roleStr.ValueString())
		}
		emojiParams.Roles = roles
	}

	// Update the emoji
	emoji, err := r.client.GuildEmojiEdit(guildID, emojiID, emojiParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Emoji",
			fmt.Sprintf("Unable to update emoji %s in guild %s: %s", emojiID, guildID, err.Error()),
		)
		return
	}

	// Update state with emoji data
	data := plan
	data.ID = types.StringValue(emoji.ID)
	data.GuildID = types.StringValue(guildID)
	data.Name = types.StringValue(emoji.Name)
	data.Animated = types.BoolValue(emoji.Animated)
	data.Managed = types.BoolValue(emoji.Managed)
	data.RequireColons = types.BoolValue(emoji.RequireColons)
	data.Available = types.BoolValue(emoji.Available)

	// Convert roles slice to list
	if len(emoji.Roles) > 0 {
		roleList := make([]attr.Value, 0, len(emoji.Roles))
		for _, roleID := range emoji.Roles {
			roleList = append(roleList, types.StringValue(roleID))
		}
		data.Roles = types.ListValueMust(types.StringType, roleList)
	} else {
		data.Roles = types.ListNull(types.StringType)
	}

	// User (creator)
	if emoji.User != nil {
		data.User = types.StringValue(emoji.User.ID)
	} else {
		data.User = types.StringNull()
	}

	// Preserve image source attributes from plan if they were set, otherwise from state
	// These are only used during creation/update, but we keep them in state to match the plan
	if !plan.Image.IsNull() && !plan.Image.IsUnknown() {
		data.Image = plan.Image
	} else if !state.Image.IsNull() && !state.Image.IsUnknown() {
		data.Image = state.Image
	} else {
		data.Image = types.StringNull()
	}

	if !plan.ImagePath.IsNull() && !plan.ImagePath.IsUnknown() {
		data.ImagePath = plan.ImagePath
	} else if !state.ImagePath.IsNull() && !state.ImagePath.IsUnknown() {
		data.ImagePath = state.ImagePath
	} else {
		data.ImagePath = types.StringNull()
	}

	if !plan.ImageURL.IsNull() && !plan.ImageURL.IsUnknown() {
		data.ImageURL = plan.ImageURL
	} else if !state.ImageURL.IsNull() && !state.ImageURL.IsUnknown() {
		data.ImageURL = state.ImageURL
	} else {
		data.ImageURL = types.StringNull()
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *emojiResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data emojiResourceModel

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
	emojiID := data.ID.ValueString()

	if guildID == "" || emojiID == "" {
		resp.Diagnostics.AddError(
			"Missing Guild or Emoji ID",
			"The guild_id and id are required to delete the emoji.",
		)
		return
	}

	// Delete the emoji
	err := r.client.GuildEmojiDelete(guildID, emojiID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Emoji",
			fmt.Sprintf("Unable to delete emoji %s from guild %s: %s", emojiID, guildID, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform.
func (r *emojiResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: guild_id:emoji_id
	importID := req.ID
	if importID == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"The import ID must be in the format 'guild_id:emoji_id'.",
		)
		return
	}

	// Parse the import ID
	parts := strings.Split(importID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			"The import ID must be in the format 'guild_id:emoji_id' (e.g., '123456789012345678:987654321098765432').",
		)
		return
	}

	guildID := parts[0]
	emojiID := parts[1]

	// Set the IDs in state - Read will populate the rest
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("guild_id"), types.StringValue(guildID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(emojiID))...)
}
