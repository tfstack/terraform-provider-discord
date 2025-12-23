package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the data source type implements the required interfaces.
var _ datasource.DataSource = &colorDataSource{}

// colorDataSource defines the data source implementation.
type colorDataSource struct{}

// colorDataSourceModel describes the data source data model.
type colorDataSourceModel struct {
	Hex types.String `tfsdk:"hex"`
	RGB types.String `tfsdk:"rgb"`
	Dec types.Int64  `tfsdk:"dec"`
}

// NewColorDataSource is a helper function to simplify testing.
func NewColorDataSource() datasource.DataSource {
	return &colorDataSource{}
}

// Metadata returns the data source type name.
func (d *colorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_color"
}

// Schema defines the schema for the data source.
func (d *colorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A simple helper to get the integer representation of a hex or RGB color for use in Discord role colors.",
		Attributes: map[string]schema.Attribute{
			"hex": schema.StringAttribute{
				Description: "A hex color value (e.g., \"#4287f5\" or \"4287f5\"). Either this or `rgb` must be provided.",
				Optional:    true,
			},
			"rgb": schema.StringAttribute{
				Description: "An RGB color value (e.g., \"rgb(46, 204, 113)\"). Either this or `hex` must be provided.",
				Optional:    true,
			},
			"dec": schema.Int64Attribute{
				Description: "The decimal integer representation of the color (0-16777215) for use in Discord role colors.",
				Computed:    true,
			},
		},
	}
}

// Configure sets up the data source with the provider's configured client.
func (d *colorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// This data source doesn't need a Discord client - it's a pure computation
}

// Read reads the data source.
func (d *colorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data colorDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that exactly one of hex or rgb is provided
	hasHex := !data.Hex.IsNull() && !data.Hex.IsUnknown() && data.Hex.ValueString() != ""
	hasRGB := !data.RGB.IsNull() && !data.RGB.IsUnknown() && data.RGB.ValueString() != ""

	if !hasHex && !hasRGB {
		resp.Diagnostics.AddError(
			"Missing Color Input",
			"Either `hex` or `rgb` must be provided.",
		)
		return
	}

	if hasHex && hasRGB {
		resp.Diagnostics.AddError(
			"Conflicting Color Input",
			"Only one of `hex` or `rgb` can be provided, not both.",
		)
		return
	}

	var decimalValue int64
	var err error

	if hasHex {
		decimalValue, err = hexToDecimal(data.Hex.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Hex Color",
				fmt.Sprintf("Unable to parse hex color: %s", err.Error()),
			)
			return
		}
	} else {
		decimalValue, err = rgbToDecimal(data.RGB.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid RGB Color",
				fmt.Sprintf("Unable to parse RGB color: %s", err.Error()),
			)
			return
		}
	}

	// Validate color range (0-16777215)
	if decimalValue < 0 || decimalValue > 16777215 {
		resp.Diagnostics.AddError(
			"Invalid Color Range",
			"Color value must be between 0 and 16777215 (0xFFFFFF).",
		)
		return
	}

	// Set the computed decimal value
	data.Dec = types.Int64Value(decimalValue)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// hexToDecimal converts a hex color string to decimal integer.
// Supports formats: "#RRGGBB", "RRGGBB", "#RGB", "RGB".
func hexToDecimal(hex string) (int64, error) {
	// Remove leading # if present
	hex = strings.TrimPrefix(hex, "#")
	hex = strings.ToUpper(hex)

	// Validate hex string contains only valid hex characters
	matched, err := regexp.MatchString("^[0-9A-F]+$", hex)
	if err != nil {
		return 0, fmt.Errorf("error validating hex string: %w", err)
	}
	if !matched {
		return 0, fmt.Errorf("invalid hex color format: %s (must contain only 0-9 and A-F)", hex)
	}

	// Handle short format (RGB -> RRGGBB)
	if len(hex) == 3 {
		hex = string(hex[0]) + string(hex[0]) + string(hex[1]) + string(hex[1]) + string(hex[2]) + string(hex[2])
	}

	// Validate length
	if len(hex) != 6 {
		return 0, fmt.Errorf("invalid hex color length: %s (must be 3 or 6 characters)", hex)
	}

	// Parse hex to decimal
	decimal, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing hex to decimal: %w", err)
	}

	return decimal, nil
}

// rgbToDecimal converts an RGB color string to decimal integer.
// Supports format: "rgb(R, G, B)" or "rgb(R,G,B)".
func rgbToDecimal(rgb string) (int64, error) {
	// Match RGB format: rgb(R, G, B) or rgb(R,G,B)
	rgbRegex := regexp.MustCompile(`^rgb\s*\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)$`)
	matches := rgbRegex.FindStringSubmatch(rgb)
	if matches == nil || len(matches) != 4 {
		return 0, fmt.Errorf("invalid RGB format: %s (expected format: rgb(R, G, B))", rgb)
	}

	// Parse R, G, B values
	r, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing red value: %w", err)
	}
	if r < 0 || r > 255 {
		return 0, fmt.Errorf("red value must be between 0 and 255, got: %d", r)
	}

	g, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing green value: %w", err)
	}
	if g < 0 || g > 255 {
		return 0, fmt.Errorf("green value must be between 0 and 255, got: %d", g)
	}

	b, err := strconv.ParseInt(matches[3], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing blue value: %w", err)
	}
	if b < 0 || b > 255 {
		return 0, fmt.Errorf("blue value must be between 0 and 255, got: %d", b)
	}

	// Convert RGB to decimal: R*65536 + G*256 + B
	decimal := r*65536 + g*256 + b

	return decimal, nil
}
