package provider

import (
	"errors"
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestIsDiscordNotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"HTTP 404", errors.New("HTTP 404 Not Found"), true},
		{"Unknown Channel", errors.New("Unknown Channel"), true},
		{"Unknown Role", errors.New("Unknown Role"), true},
		{"Unknown Invite", errors.New("Unknown Invite"), true},
		{"Discord REST 404", &discordgo.RESTError{Response: &http.Response{StatusCode: http.StatusNotFound}}, true},
		{"Discord REST 10003", &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Code: 10003}}, true},
		{"Discord REST 10011", &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Code: 10011}}, true},
		{"Discord REST other code", &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Code: 50001}}, false},
		{"Permission Error", errors.New("HTTP 403 Forbidden"), false},
		{"Other Error", errors.New("some other error"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsDiscordNotFound(tc.err)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}
