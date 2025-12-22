package provider

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stretchr/testify/assert"
)

func TestChannelDataSource_Metadata(t *testing.T) {
	ds := NewChannelDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "discord",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	assert.Equal(t, "discord_channel", resp.TypeName)
}

func TestChannelDataSource_Schema(t *testing.T) {
	ds := NewChannelDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Equal(t, "Retrieves a single Discord channel by its ID.", resp.Schema.Description)

	// Check required attribute
	channelIDAttr, ok := resp.Schema.Attributes["channel_id"]
	assert.True(t, ok)
	assert.True(t, channelIDAttr.IsRequired())

	// Check computed attributes
	computedAttrs := []string{"id", "name", "type", "category_id", "position", "guild_id"}
	for _, attrName := range computedAttrs {
		attr, ok := resp.Schema.Attributes[attrName]
		assert.True(t, ok, "Attribute %s should exist", attrName)
		assert.True(t, attr.IsComputed(), "Attribute %s should be computed", attrName)
	}
}

func TestChannelDataSource_Configure(t *testing.T) {
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
			ds := &channelDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: tt.providerData,
			}
			resp := &datasource.ConfigureResponse{}

			ds.Configure(context.Background(), req, resp)

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

// Note: Tests for Read() method that require Discord API calls should be
// implemented as acceptance tests with TF_ACC=1 environment variable set.
// These unit tests verify the schema, metadata, and configuration validation
// without making API calls.
