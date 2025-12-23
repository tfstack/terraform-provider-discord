package provider

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
)

func TestChannelResource_Metadata(t *testing.T) {
	r := NewChannelResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "discord",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(t.Context(), req, resp)

	assert.Equal(t, "discord_channel", resp.TypeName)
}

func TestChannelResource_Schema(t *testing.T) {
	r := NewChannelResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(t.Context(), req, resp)

	assert.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Description, "Creates and manages a Discord channel")

	// Check required attributes
	nameAttr, ok := resp.Schema.Attributes["name"]
	assert.True(t, ok)
	assert.True(t, nameAttr.IsRequired())

	guildIDAttr, ok := resp.Schema.Attributes["guild_id"]
	assert.True(t, ok)
	assert.True(t, guildIDAttr.IsRequired())

	// Check optional attributes
	optionalAttrs := []string{"type", "category_id", "position"}
	for _, attrName := range optionalAttrs {
		attr, ok := resp.Schema.Attributes[attrName]
		assert.True(t, ok, "Attribute %s should exist", attrName)
		assert.True(t, attr.IsOptional(), "Attribute %s should be optional", attrName)
	}

	// Check computed attributes
	computedAttrs := []string{"id", "type"}
	for _, attrName := range computedAttrs {
		attr, ok := resp.Schema.Attributes[attrName]
		assert.True(t, ok, "Attribute %s should exist", attrName)
		assert.True(t, attr.IsComputed(), "Attribute %s should be computed", attrName)
	}
}

func TestChannelResource_Configure(t *testing.T) {
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
			r := &channelResource{}
			req := resource.ConfigureRequest{
				ProviderData: tt.providerData,
			}
			resp := &resource.ConfigureResponse{}

			r.Configure(t.Context(), req, resp)

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

func TestChannelTypeFromString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    discordgo.ChannelType
		expectError bool
	}{
		{
			name:        "text channel",
			input:       "text",
			expected:    discordgo.ChannelTypeGuildText,
			expectError: false,
		},
		{
			name:        "empty string defaults to text",
			input:       "",
			expected:    discordgo.ChannelTypeGuildText,
			expectError: false,
		},
		{
			name:        "voice channel",
			input:       "voice",
			expected:    discordgo.ChannelTypeGuildVoice,
			expectError: false,
		},
		{
			name:        "category channel",
			input:       "category",
			expected:    discordgo.ChannelTypeGuildCategory,
			expectError: false,
		},
		{
			name:        "invalid type",
			input:       "invalid",
			expected:    discordgo.ChannelTypeGuildText,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := channelTypeFromString(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid channel type")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestChannelTypeToString(t *testing.T) {
	tests := []struct {
		name     string
		input    discordgo.ChannelType
		expected string
	}{
		{
			name:     "text channel",
			input:    discordgo.ChannelTypeGuildText,
			expected: "text",
		},
		{
			name:     "voice channel",
			input:    discordgo.ChannelTypeGuildVoice,
			expected: "voice",
		},
		{
			name:     "category channel",
			input:    discordgo.ChannelTypeGuildCategory,
			expected: "category",
		},
		{
			name:     "unknown type defaults to text",
			input:    discordgo.ChannelType(999),
			expected: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := channelTypeToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Note: Tests for Create, Read, Update, and Delete methods that require Discord API calls
// should be implemented as acceptance tests with TF_ACC=1 environment variable set.
// These unit tests verify the schema, metadata, configuration validation, and helper functions
// without making API calls.
