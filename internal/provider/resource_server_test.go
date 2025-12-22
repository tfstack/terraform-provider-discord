package provider

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

func TestServerResource_Metadata(t *testing.T) {
	r := NewServerResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "discord",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	assert.Equal(t, "discord_server", resp.TypeName)
}

func TestServerResource_Schema(t *testing.T) {
	r := NewServerResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Creates and manages a Discord server")

	// Check required attribute
	nameAttr, ok := resp.Schema.Attributes["name"]
	assert.True(t, ok)
	assert.True(t, nameAttr.IsRequired())

	// Check computed attribute
	idAttr, ok := resp.Schema.Attributes["id"]
	assert.True(t, ok)
	assert.True(t, idAttr.IsComputed())
}

func TestServerResource_Configure(t *testing.T) {
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
			errorContains: "Unexpected Resource Configure Type",
		},
		{
			name:         "nil provider data",
			providerData: nil,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &serverResource{}
			req := resource.ConfigureRequest{
				ProviderData: tt.providerData,
			}
			resp := &resource.ConfigureResponse{}

			r.Configure(context.Background(), req, resp)

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

// Note: Tests for Create, Read, Update, and Delete methods that require Discord API calls
// should be implemented as acceptance tests with TF_ACC=1 environment variable set.
// These unit tests verify the schema, metadata, and configuration validation
// without making API calls.
