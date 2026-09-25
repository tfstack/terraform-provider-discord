package provider

import (
	"fmt"
	"sort"
)

// discordPermissionBits maps Terraform attribute names to Discord permission
// bit values. Names follow common Discord provider conventions (snake_case).
// Reference: https://discord.com/developers/docs/topics/permissions
var discordPermissionBits = map[string]int64{
	"create_instant_invite":       0x1,
	"kick_members":                0x2,
	"ban_members":                 0x4,
	"administrator":               0x8,
	"manage_channels":             0x10,
	"manage_guild":                0x20,
	"add_reactions":               0x40,
	"view_audit_log":              0x80,
	"priority_speaker":            0x100,
	"stream":                      0x200,
	"view_channel":                0x400,
	"send_messages":               0x800,
	"send_tts_messages":           0x1000,
	"manage_messages":             0x2000,
	"embed_links":                 0x4000,
	"attach_files":                0x8000,
	"read_message_history":        0x10000,
	"mention_everyone":            0x20000,
	"use_external_emojis":         0x40000,
	"view_guild_insights":         0x80000,
	"connect":                     0x100000,
	"speak":                       0x200000,
	"mute_members":                0x400000,
	"deafen_members":              0x800000,
	"move_members":                0x1000000,
	"use_vad":                     0x2000000,
	"change_nickname":             0x4000000,
	"manage_nicknames":            0x8000000,
	"manage_roles":                0x10000000,
	"manage_webhooks":             0x20000000,
	"manage_emojis":               0x40000000,
	"use_application_commands":    0x80000000,
	"request_to_speak":            0x100000000,
	"manage_events":               0x200000000,
	"manage_threads":              0x400000000,
	"create_public_threads":       0x800000000,
	"create_private_threads":      0x1000000000,
	"use_external_stickers":       0x2000000000,
	"send_thread_messages":        0x4000000000,
	"start_embedded_activities":   0x8000000000,
	"moderate_members":            0x10000000000,
	"view_monetization_analytics": 0x20000000000,
	"use_soundboard":              0x40000000000,
	"create_expressions":          0x80000000000,
	"create_events":               0x100000000000,
	"use_external_sounds":         0x200000000000,
	"send_voice_messages":         0x400000000000,
	"set_voice_channel_status":    0x1000000000000,
	"send_polls":                  0x2000000000000,
	"use_external_apps":           0x4000000000000,
	"pin_messages":                0x8000000000000,
	"bypass_slowmode":             0x10000000000000,
}

// permissionFlagNames returns sorted permission attribute names for stable schemas/tests.
func permissionFlagNames() []string {
	names := make([]string, 0, len(discordPermissionBits))
	for name := range discordPermissionBits {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// computePermissionBits builds allow/deny bitmasks from per-flag values.
// Each value must be "allow", "deny", or "unset" (empty/"unset" are equivalent).
func computePermissionBits(flags map[string]string, allowExtends, denyExtends int64) (allowBits, denyBits int64, err error) {
	allowBits = allowExtends
	denyBits = denyExtends

	for name, raw := range flags {
		bit, ok := discordPermissionBits[name]
		if !ok {
			return 0, 0, fmt.Errorf("unknown permission %q", name)
		}
		switch raw {
		case "", "unset":
			continue
		case "allow":
			allowBits |= bit
		case "deny":
			denyBits |= bit
		default:
			return 0, 0, fmt.Errorf("permission %q must be allow, deny, or unset; got %q", name, raw)
		}
	}

	return allowBits, denyBits, nil
}
