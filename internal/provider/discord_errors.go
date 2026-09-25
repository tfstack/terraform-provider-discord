package provider

import (
	"errors"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// IsDiscordNotFound returns true when the Discord API indicates the target
// resource no longer exists (HTTP 404 or Discord "unknown resource" codes).
func IsDiscordNotFound(err error) bool {
	if err == nil {
		return false
	}

	// Prefer typed RESTError inspection. discordgo.RESTError.Error() panics when
	// Response is nil, so never call err.Error() before this check.
	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) {
		if restErr.Response != nil && restErr.Response.StatusCode == http.StatusNotFound {
			return true
		}
		if restErr.Message != nil && isDiscordUnknownResourceCode(restErr.Message.Code) {
			return true
		}
		return false
	}

	msg := err.Error()
	if strings.Contains(msg, "HTTP 404") || strings.Contains(msg, "404 Not Found") {
		return true
	}

	lower := strings.ToLower(msg)
	return strings.Contains(lower, "unknown channel") ||
		strings.Contains(lower, "unknown role") ||
		strings.Contains(lower, "unknown guild") ||
		strings.Contains(lower, "unknown server") ||
		strings.Contains(lower, "unknown emoji") ||
		strings.Contains(lower, "unknown invite") ||
		strings.Contains(lower, "unknown webhook") ||
		strings.Contains(lower, "unknown message") ||
		strings.Contains(lower, "unknown member")
}

// Discord API "unknown resource" codes commonly returned after out-of-band deletes.
// See https://discord.com/developers/docs/topics/opcodes-and-status-codes#json
func isDiscordUnknownResourceCode(code int) bool {
	switch code {
	case 10003, // Unknown Channel
		10004, // Unknown Guild
		10008, // Unknown Message
		10011, // Unknown Role
		10013, // Unknown User
		10014, // Unknown Emoji
		10015: // Unknown Webhook
		return true
	default:
		return false
	}
}
