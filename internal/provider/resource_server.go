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
var _ resource.Resource = &serverResource{}
var _ resource.ResourceWithConfigure = &serverResource{}
var _ resource.ResourceWithImportState = &serverResource{}

// serverResource defines the resource implementation.
type serverResource struct {
	client *discordgo.Session
}

// serverResourceModel describes the resource data model.
type serverResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// NewServerResource is a helper function to simplify testing.
func NewServerResource() resource.Resource {
	return &serverResource{}
}

// Metadata returns the resource type name.
func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

// Schema defines the schema for the resource.
func (r *serverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a Discord server (guild).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the server (guild).",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the server (guild). Must be 2-100 characters.",
				Required:    true,
			},
		},
	}
}

// Configure sets up the resource with the provider's configured client.
func (r *serverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data serverResourceModel

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

	name := data.Name.ValueString()
	if name == "" {
		resp.Diagnostics.AddError(
			"Missing Server Name",
			"The name attribute is required.",
		)
		return
	}

	// Validate name length (Discord requires 2-100 characters)
	if len(name) < 2 || len(name) > 100 {
		resp.Diagnostics.AddError(
			"Invalid Server Name",
			"Server name must be between 2 and 100 characters.",
		)
		return
	}

	// Create the guild (server)
	// Note: This endpoint requires a user OAuth2 token, not a bot token
	// Bot tokens will receive error 20001: "Bots cannot use this endpoint"
	guild, err := r.client.GuildCreate(name)
	if err != nil {
		// Provide more helpful error message for bot token limitation
		if err.Error() != "" {
			resp.Diagnostics.AddError(
				"Error Creating Server",
				fmt.Sprintf("Unable to create server %s: %s\n\nNote: Creating servers requires a user OAuth2 token, not a bot token. Bot tokens cannot create servers.", name, err.Error()),
			)
		} else {
			resp.Diagnostics.AddError(
				"Error Creating Server",
				fmt.Sprintf("Unable to create server %s. Note: Creating servers requires a user OAuth2 token, not a bot token. Bot tokens cannot create servers.", name),
			)
		}
		return
	}

	// Verify server was created successfully
	if guild == nil {
		resp.Diagnostics.AddError(
			"Server Creation Failed",
			fmt.Sprintf("Server creation API call succeeded but returned nil guild for %s. This may indicate a Discord API issue.", name),
		)
		return
	}

	// Verify server ID is set
	if guild.ID == "" {
		resp.Diagnostics.AddError(
			"Invalid Server Response",
			fmt.Sprintf("Server was created but has no ID. Server name: %s", name),
		)
		return
	}

	// Populate the model with server data
	data.ID = types.StringValue(guild.ID)
	data.Name = types.StringValue(guild.Name)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data serverResourceModel

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

	serverID := data.ID.ValueString()
	if serverID == "" {
		resp.Diagnostics.AddError(
			"Missing Server ID",
			"The server ID is missing from the state.",
		)
		return
	}

	// Fetch the server by ID
	guild, err := r.client.Guild(serverID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Server",
			fmt.Sprintf("Unable to fetch server %s: %s", serverID, err.Error()),
		)
		return
	}

	// Update the model with server data
	data.ID = types.StringValue(guild.ID)
	data.Name = types.StringValue(guild.Name)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data serverResourceModel

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

	serverID := data.ID.ValueString()
	if serverID == "" {
		resp.Diagnostics.AddError(
			"Missing Server ID",
			"The server ID is missing from the state.",
		)
		return
	}

	name := data.Name.ValueString()
	if name == "" {
		resp.Diagnostics.AddError(
			"Missing Server Name",
			"The name attribute is required.",
		)
		return
	}

	// Validate name length
	if len(name) < 2 || len(name) > 100 {
		resp.Diagnostics.AddError(
			"Invalid Server Name",
			"Server name must be between 2 and 100 characters.",
		)
		return
	}

	// Update the guild (server)
	guildParams := &discordgo.GuildParams{
		Name: name,
	}

	guild, err := r.client.GuildEdit(serverID, guildParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Server",
			fmt.Sprintf("Unable to update server %s: %s", serverID, err.Error()),
		)
		return
	}

	// Update the model with server data
	data.ID = types.StringValue(guild.ID)
	data.Name = types.StringValue(guild.Name)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data serverResourceModel

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

	serverID := data.ID.ValueString()
	if serverID == "" {
		resp.Diagnostics.AddError(
			"Missing Server ID",
			"The server ID is missing from the state.",
		)
		return
	}

	// Delete the guild (server)
	err := r.client.GuildDelete(serverID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Server",
			fmt.Sprintf("Unable to delete server %s: %s", serverID, err.Error()),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform state.
func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the server ID
	serverID := req.ID

	// Ensure client is configured
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Discord Client Not Configured",
			"The Discord client was not properly configured. This is a provider error.",
		)
		return
	}

	// Fetch the server to populate state
	guild, err := r.client.Guild(serverID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Server",
			fmt.Sprintf("Unable to fetch server %s: %s", serverID, err.Error()),
		)
		return
	}

	// Create a model with the server data
	var data serverResourceModel
	data.ID = types.StringValue(guild.ID)
	data.Name = types.StringValue(guild.Name)

	// Save the imported state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
