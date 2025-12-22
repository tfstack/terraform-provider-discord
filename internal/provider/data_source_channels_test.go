package provider

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stretchr/testify/assert"
)

func TestChannelsDataSource_Metadata(t *testing.T) {
	ds := NewChannelsDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "discord",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	assert.Equal(t, "discord_channels", resp.TypeName)
}

func TestChannelsDataSource_Schema(t *testing.T) {
	ds := NewChannelsDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Retrieves channels from a Discord guild")

	// Check required attribute
	guildIDAttr, ok := resp.Schema.Attributes["guild_id"]
	assert.True(t, ok)
	assert.True(t, guildIDAttr.IsRequired())

	// Check optional attribute
	categoryNameAttr, ok := resp.Schema.Attributes["category_name"]
	assert.True(t, ok)
	assert.True(t, categoryNameAttr.IsOptional())

	// Check computed attribute
	channelsAttr, ok := resp.Schema.Attributes["channels"]
	assert.True(t, ok)
	assert.True(t, channelsAttr.IsComputed())
}

func TestChannelsDataSource_Configure(t *testing.T) {
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
			ds := &channelsDataSource{}
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
