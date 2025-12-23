package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNewColorDataSource(t *testing.T) {
	ds := NewColorDataSource()
	if ds == nil {
		t.Fatal("NewColorDataSource() returned nil")
	}
	// The function signature already ensures it returns datasource.DataSource
	_ = ds
}

func TestHexToDecimal(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		expected int64
		wantErr  bool
	}{
		{
			name:     "6-digit hex with #",
			hex:      "#4287f5",
			expected: 4360181,
			wantErr:  false,
		},
		{
			name:     "6-digit hex without #",
			hex:      "4287f5",
			expected: 4360181,
			wantErr:  false,
		},
		{
			name:     "3-digit hex with #",
			hex:      "#f5a",
			expected: 0xff55aa,
			wantErr:  false,
		},
		{
			name:     "3-digit hex without #",
			hex:      "f5a",
			expected: 0xff55aa,
			wantErr:  false,
		},
		{
			name:     "black",
			hex:      "#000000",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "white",
			hex:      "#FFFFFF",
			expected: 16777215,
			wantErr:  false,
		},
		{
			name:    "invalid hex characters",
			hex:     "#GGGGGG",
			wantErr: true,
		},
		{
			name:    "invalid length",
			hex:     "#12345",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hexToDecimal(tt.hex)
			if (err != nil) != tt.wantErr {
				t.Errorf("hexToDecimal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("hexToDecimal() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRgbToDecimal(t *testing.T) {
	tests := []struct {
		name     string
		rgb      string
		expected int64
		wantErr  bool
	}{
		{
			name:     "valid rgb with spaces",
			rgb:      "rgb(46, 204, 113)",
			expected: 46*65536 + 204*256 + 113, // 3033457
			wantErr:  false,
		},
		{
			name:     "valid rgb without spaces",
			rgb:      "rgb(46,204,113)",
			expected: 46*65536 + 204*256 + 113,
			wantErr:  false,
		},
		{
			name:     "red",
			rgb:      "rgb(255, 0, 0)",
			expected: 16711680,
			wantErr:  false,
		},
		{
			name:     "green",
			rgb:      "rgb(0, 255, 0)",
			expected: 65280,
			wantErr:  false,
		},
		{
			name:     "blue",
			rgb:      "rgb(0, 0, 255)",
			expected: 255,
			wantErr:  false,
		},
		{
			name:    "invalid format",
			rgb:     "46, 204, 113",
			wantErr: true,
		},
		{
			name:    "invalid red value",
			rgb:     "rgb(256, 0, 0)",
			wantErr: true,
		},
		{
			name:    "negative value",
			rgb:     "rgb(-1, 0, 0)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rgbToDecimal(tt.rgb)
			if (err != nil) != tt.wantErr {
				t.Errorf("rgbToDecimal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("rgbToDecimal() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestColorDataSource_Read(t *testing.T) {
	ds := &colorDataSource{}

	// Test cases would require setting up the full Terraform framework context
	// This is a placeholder for more comprehensive integration tests
	_ = ds
}

// Helper function to create a test model.
func createColorModel(hex, rgb string) colorDataSourceModel {
	model := colorDataSourceModel{}
	if hex != "" {
		model.Hex = types.StringValue(hex)
	} else {
		model.Hex = types.StringNull()
	}
	if rgb != "" {
		model.RGB = types.StringValue(rgb)
	} else {
		model.RGB = types.StringNull()
	}
	return model
}

func TestColorDataSource_Validation(t *testing.T) {
	// Test that validation works correctly
	// This would be tested in integration tests with the full framework
	_ = createColorModel
}
