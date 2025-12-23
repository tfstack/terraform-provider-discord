package provider

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stretchr/testify/assert"
)

func TestRolesDataSource_Metadata(t *testing.T) {
	d := NewRolesDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "discord",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(t.Context(), req, resp)

	assert.Equal(t, "discord_roles", resp.TypeName)
}

func TestRolesDataSource_Schema(t *testing.T) {
	d := NewRolesDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(t.Context(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Retrieves all roles")

	// Check required attribute
	guildIDAttr, ok := resp.Schema.Attributes["guild_id"]
	assert.True(t, ok)
	assert.True(t, guildIDAttr.IsRequired())

	// Check computed attribute
	rolesAttr, ok := resp.Schema.Attributes["roles"]
	assert.True(t, ok)
	assert.True(t, rolesAttr.IsComputed())
}

func TestRolesDataSource_Configure(t *testing.T) {
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
			d := &rolesDataSource{}
			req := datasource.ConfigureRequest{
				ProviderData: tt.providerData,
			}
			resp := &datasource.ConfigureResponse{}

			d.Configure(t.Context(), req, resp)

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
