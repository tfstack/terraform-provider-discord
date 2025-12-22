package provider

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/stretchr/testify/assert"
)

func TestRoleDataSource_Metadata(t *testing.T) {
	d := NewRoleDataSource()
	req := datasource.MetadataRequest{
		ProviderTypeName: "discord",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	assert.Equal(t, "discord_role", resp.TypeName)
}

func TestRoleDataSource_Schema(t *testing.T) {
	d := NewRoleDataSource()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Retrieves a Discord role")

	// Check optional attributes
	roleIDAttr, ok := resp.Schema.Attributes["role_id"]
	assert.True(t, ok)
	assert.True(t, roleIDAttr.IsOptional())

	nameAttr, ok := resp.Schema.Attributes["name"]
	assert.True(t, ok)
	assert.True(t, nameAttr.IsOptional())

	// Check computed attributes
	idAttr, ok := resp.Schema.Attributes["id"]
	assert.True(t, ok)
	assert.True(t, idAttr.IsComputed())
}

func TestRoleDataSource_Configure(t *testing.T) {
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
			d := &roleDataSource{}
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
