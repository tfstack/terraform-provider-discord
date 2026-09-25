package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the data source type implements the required interfaces.
var _ datasource.DataSource = &permissionsDataSource{}

// permissionsDataSource computes Discord permission bitmasks from named flags.
type permissionsDataSource struct{}

// NewPermissionsDataSource is a helper function to simplify testing.
func NewPermissionsDataSource() datasource.DataSource {
	return &permissionsDataSource{}
}

// Metadata returns the data source type name.
func (d *permissionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permissions"
}

// Schema defines the schema for the data source.
func (d *permissionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"allow_extends": schema.Int64Attribute{
			Description: "Base allow permission bits to extend (OR'd with flags set to allow).",
			Optional:    true,
		},
		"deny_extends": schema.Int64Attribute{
			Description: "Base deny permission bits to extend (OR'd with flags set to deny).",
			Optional:    true,
		},
		"allow_bits": schema.Int64Attribute{
			Description: "Computed allow permission bits for use with role permissions or channel overwrite allow.",
			Computed:    true,
		},
		"deny_bits": schema.Int64Attribute{
			Description: "Computed deny permission bits for use with channel overwrite deny.",
			Computed:    true,
		},
		"permissions": schema.Int64Attribute{
			Description: "Alias of allow_bits for convenient use with discord_role.permissions and discord_everyone_role.permissions.",
			Computed:    true,
		},
	}

	for _, name := range permissionFlagNames() {
		attrs[name] = schema.StringAttribute{
			Description: fmt.Sprintf("Set the `%s` permission bit to `allow`, `deny`, or `unset` (default).", name),
			Optional:    true,
		}
	}

	resp.Schema = schema.Schema{
		Description: "Computes Discord permission integers from named permission flags. " +
			"Use allow_bits/permissions with roles, and allow_bits plus deny_bits with channel permission overwrites.",
		Attributes: attrs,
	}
}

// Configure is a no-op; this data source is pure computation.
func (d *permissionsDataSource) Configure(_ context.Context, _ datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
}

// Read computes permission bitmasks from configuration.
func (d *permissionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var allowExtends, denyExtends types.Int64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("allow_extends"), &allowExtends)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("deny_extends"), &denyExtends)...)
	if resp.Diagnostics.HasError() {
		return
	}

	flags := make(map[string]string, len(discordPermissionBits))
	for _, name := range permissionFlagNames() {
		var value types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(name), &value)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if value.IsNull() || value.IsUnknown() {
			flags[name] = "unset"
			continue
		}
		flags[name] = value.ValueString()
	}

	var allowBase, denyBase int64
	if !allowExtends.IsNull() && !allowExtends.IsUnknown() {
		allowBase = allowExtends.ValueInt64()
	}
	if !denyExtends.IsNull() && !denyExtends.IsUnknown() {
		denyBase = denyExtends.ValueInt64()
	}

	allowBits, denyBits, err := computePermissionBits(flags, allowBase, denyBase)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Permission Configuration", err.Error())
		return
	}

	// Persist inputs and computed outputs into state.
	if allowExtends.IsNull() || allowExtends.IsUnknown() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_extends"), types.Int64Null())...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_extends"), allowExtends)...)
	}
	if denyExtends.IsNull() || denyExtends.IsUnknown() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deny_extends"), types.Int64Null())...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deny_extends"), denyExtends)...)
	}

	for _, name := range permissionFlagNames() {
		raw := flags[name]
		if raw == "unset" {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(name), types.StringNull())...)
			continue
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(name), types.StringValue(raw))...)
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_bits"), types.Int64Value(allowBits))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deny_bits"), types.Int64Value(denyBits))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permissions"), types.Int64Value(allowBits))...)
}
