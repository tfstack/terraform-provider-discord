package provider

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stretchr/testify/assert"
)

func TestCategoryDataSource_Metadata(t *testing.T) {
	ds := NewCategoryDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "discord",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	assert.Equal(t, "discord_category", resp.TypeName)
}

func TestCategoryDataSource_Schema(t *testing.T) {
	ds := NewCategoryDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Retrieves a Discord category channel")

	// Check optional attributes (can use category_id OR name+guild_id)
	categoryIDAttr, ok := resp.Schema.Attributes["category_id"]
	assert.True(t, ok)
	assert.True(t, categoryIDAttr.IsOptional())

	nameAttr, ok := resp.Schema.Attributes["name"]
	assert.True(t, ok)
	assert.True(t, nameAttr.IsOptional())

	guildIDAttr, ok := resp.Schema.Attributes["guild_id"]
	assert.True(t, ok)
	assert.True(t, guildIDAttr.IsOptional())

	// Check computed attributes
	computedAttrs := []string{"id", "position"}
	for _, attrName := range computedAttrs {
		attr, ok := resp.Schema.Attributes[attrName]
		assert.True(t, ok, "Attribute %s should exist", attrName)
		assert.True(t, attr.IsComputed(), "Attribute %s should be computed", attrName)
	}
}

func TestCategoryDataSource_Configure(t *testing.T) {
	tests := []struct {
		name          string
		providerData  interface{}
		expectError   bool
		errorContains string
	}{
		{
			name:         "valid discordgo.Session",
			providerData: &discordgo.Session{},
			expectError:  false,
		},
		{
			name:          "invalid provider data type",
			providerData:  "invalid",
			expectError:   true,
			errorContains: "Unexpected Data Source Configure Type",
		},
		{
			name:         "nil provider data",
			providerData: nil,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &categoryDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: tt.providerData,
			}
			resp := &datasource.ConfigureResponse{}

			d.Configure(context.Background(), req, resp)

			if tt.expectError {
				assert.True(t, resp.Diagnostics.HasError())
				if tt.errorContains != "" {
					assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), tt.errorContains)
				}
			} else {
				assert.False(t, resp.Diagnostics.HasError())
			}
		})
	}
}

// Note: Tests for Read method that require Discord API calls
// should be implemented as acceptance tests with TF_ACC=1 environment variable set.
// These unit tests verify the schema, metadata, and configuration validation
// without making API calls.
