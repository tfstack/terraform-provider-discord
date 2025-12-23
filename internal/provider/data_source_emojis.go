package provider

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the data source type implements the required interfaces.
var _ datasource.DataSource = &emojisDataSource{}

// emojisDataSource defines the data source implementation.
type emojisDataSource struct {
	client *discordgo.Session
}

// emojisDataSourceModel describes the data source data model.
type emojisDataSourceModel struct {
	GuildID types.String `tfsdk:"guild_id"`
	Emojis  types.List   `tfsdk:"emojis"`
}

// emojiModel describes a single emoji in the data source.
type emojiModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Animated      types.Bool   `tfsdk:"animated"`
	Managed       types.Bool   `tfsdk:"managed"`
	RequireColons types.Bool   `tfsdk:"require_colons"`
	Roles         types.List   `tfsdk:"roles"`
	User          types.String `tfsdk:"user"`
	Available     types.Bool   `tfsdk:"available"`
}

// NewEmojisDataSource is a helper function to simplify testing.
func NewEmojisDataSource() datasource.DataSource {
	return &emojisDataSource{}
}

// Metadata returns the data source type name.
func (d *emojisDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_emojis"
}

// Schema defines the schema for the data source.
func (d *emojisDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves all custom emojis from a Discord guild (server).",
		Attributes: map[string]schema.Attribute{
			"guild_id": schema.StringAttribute{
				Description: "The ID of the Discord guild (server).",
				Required:    true,
			},
			"emojis": schema.ListNestedAttribute{
				Description: "List of custom emojis in the guild.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the emoji.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the emoji.",
							Computed:    true,
						},
						"animated": schema.BoolAttribute{
							Description: "Whether the emoji is animated.",
							Computed:    true,
						},
						"managed": schema.BoolAttribute{
							Description: "Whether the emoji is managed by an integration.",
							Computed:    true,
						},
						"require_colons": schema.BoolAttribute{
							Description: "Whether the emoji requires colons to be used.",
							Computed:    true,
						},
						"roles": schema.ListAttribute{
							Description: "List of role IDs that can use this emoji. Empty if available to all roles.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"user": schema.StringAttribute{
							Description: "The ID of the user who created the emoji.",
							Computed:    true,
						},
						"available": schema.BoolAttribute{
							Description: "Whether the emoji is available for use.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *emojisDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*discordgo.Session)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *discordgo.Session, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *emojisDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data emojisDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure client is configured
	if d.client == nil {
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

	// Fetch all emojis for the guild
	emojis, err := d.client.GuildEmojis(guildID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Fetching Emojis",
			fmt.Sprintf("Unable to fetch emojis for guild %s: %s", guildID, err.Error()),
		)
		return
	}

	// Convert emojis to the model
	emojiList := make([]attr.Value, 0, len(emojis))
	for _, emoji := range emojis {
		emojiModel := emojiModel{
			ID:            types.StringValue(emoji.ID),
			Name:          types.StringValue(emoji.Name),
			Animated:      types.BoolValue(emoji.Animated),
			Managed:       types.BoolValue(emoji.Managed),
			RequireColons: types.BoolValue(emoji.RequireColons),
			Available:     types.BoolValue(emoji.Available),
		}

		// Convert roles slice to list
		if len(emoji.Roles) > 0 {
			roleList := make([]attr.Value, 0, len(emoji.Roles))
			for _, roleID := range emoji.Roles {
				roleList = append(roleList, types.StringValue(roleID))
			}
			emojiModel.Roles = types.ListValueMust(types.StringType, roleList)
		} else {
			emojiModel.Roles = types.ListNull(types.StringType)
		}

		// User (creator)
		if emoji.User != nil {
			emojiModel.User = types.StringValue(emoji.User.ID)
		} else {
			emojiModel.User = types.StringNull()
		}

		// Convert emojiModel to object
		emojiObj, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"id":             types.StringType,
			"name":           types.StringType,
			"animated":       types.BoolType,
			"managed":        types.BoolType,
			"require_colons": types.BoolType,
			"roles":          types.ListType{ElemType: types.StringType},
			"user":           types.StringType,
			"available":      types.BoolType,
		}, emojiModel)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		emojiList = append(emojiList, emojiObj)
	}

	// Set the emojis list
	data.Emojis = types.ListValueMust(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":             types.StringType,
				"name":           types.StringType,
				"animated":       types.BoolType,
				"managed":        types.BoolType,
				"require_colons": types.BoolType,
				"roles":          types.ListType{ElemType: types.StringType},
				"user":           types.StringType,
				"available":      types.BoolType,
			},
		},
		emojiList,
	)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
