package provider

import "testing"

func TestParseGuildRoleImportID(t *testing.T) {
	cases := []struct {
		name      string
		importID  string
		wantGuild string
		wantRole  string
		wantOK    bool
	}{
		{
			name:      "18-digit snowflakes",
			importID:  "123456789012345678:987654321098765432",
			wantGuild: "123456789012345678",
			wantRole:  "987654321098765432",
			wantOK:    true,
		},
		{
			name:      "19-digit snowflakes",
			importID:  "1234567890123456789:9876543210987654321",
			wantGuild: "1234567890123456789",
			wantRole:  "9876543210987654321",
			wantOK:    true,
		},
		{
			name:      "mixed lengths",
			importID:  "1234567890123456789:987654321098765432",
			wantGuild: "1234567890123456789",
			wantRole:  "987654321098765432",
			wantOK:    true,
		},
		{
			name:     "missing colon",
			importID: "1234567890123456789",
			wantOK:   false,
		},
		{
			name:     "empty guild",
			importID: ":9876543210987654321",
			wantOK:   false,
		},
		{
			name:     "empty role",
			importID: "1234567890123456789:",
			wantOK:   false,
		},
		{
			name:     "empty",
			importID: "",
			wantOK:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			guildID, roleID, ok := parseGuildRoleImportID(tc.importID)
			if ok != tc.wantOK {
				t.Fatalf("ok: got %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if guildID != tc.wantGuild || roleID != tc.wantRole {
				t.Fatalf("got guild=%q role=%q, want guild=%q role=%q", guildID, roleID, tc.wantGuild, tc.wantRole)
			}
		})
	}
}
