package provider

import (
	"errors"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// IsDiscordNotFound returns true when the Discord API indicates the target
// resource no longer exists, such as 404 responses or Discord error code 10003.
func IsDiscordNotFound(err error) bool {
	if err == nil {
		return false
	}

	if strings.Contains(err.Error(), "HTTP 404") || strings.Contains(err.Error(), "404 Not Found") || strings.Contains(err.Error(), "10003") {
		return true
	}

	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "unknown channel") ||
		strings.Contains(lower, "unknown role") ||
		strings.Contains(lower, "unknown guild") ||
		strings.Contains(lower, "unknown server") ||
		strings.Contains(lower, "unknown emoji") ||
		strings.Contains(lower, "unknown invite") ||
		strings.Contains(lower, "unknown webhook") ||
		strings.Contains(lower, "unknown message") ||
		strings.Contains(lower, "unknown member") {
		return true
	}

	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) {
		if restErr.Response != nil && restErr.Response.StatusCode == http.StatusNotFound {
			return true
		}
		if restErr.Message != nil && restErr.Message.Code == 10003 {
			return true
		}
	}

	return false
}
