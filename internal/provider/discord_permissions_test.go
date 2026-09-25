package provider

import "testing"

func TestComputePermissionBits(t *testing.T) {
	cases := []struct {
		name         string
		flags        map[string]string
		allowExtends int64
		denyExtends  int64
		wantAllow    int64
		wantDeny     int64
		wantErr      bool
	}{
		{
			name:      "empty",
			flags:     map[string]string{},
			wantAllow: 0,
			wantDeny:  0,
		},
		{
			name: "allow manage_messages and kick_members",
			flags: map[string]string{
				"manage_messages": "allow",
				"kick_members":    "allow",
			},
			wantAllow: 0x2000 | 0x2,
			wantDeny:  0,
		},
		{
			name: "deny administrator",
			flags: map[string]string{
				"administrator": "deny",
			},
			wantAllow: 0,
			wantDeny:  0x8,
		},
		{
			name: "mixed allow deny and unset",
			flags: map[string]string{
				"view_channel":    "allow",
				"send_messages":   "allow",
				"manage_messages": "deny",
				"administrator":   "unset",
			},
			wantAllow: 0x400 | 0x800,
			wantDeny:  0x2000,
		},
		{
			name: "extends",
			flags: map[string]string{
				"kick_members": "allow",
			},
			allowExtends: 0x1,
			denyExtends:  0x4,
			wantAllow:    0x1 | 0x2,
			wantDeny:     0x4,
		},
		{
			name: "invalid value",
			flags: map[string]string{
				"kick_members": "yes",
			},
			wantErr: true,
		},
		{
			name: "unknown flag",
			flags: map[string]string{
				"not_a_real_permission": "allow",
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotAllow, gotDeny, err := computePermissionBits(tc.flags, tc.allowExtends, tc.denyExtends)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotAllow != tc.wantAllow || gotDeny != tc.wantDeny {
				t.Fatalf("got allow=%#x deny=%#x, want allow=%#x deny=%#x", gotAllow, gotDeny, tc.wantAllow, tc.wantDeny)
			}
		})
	}
}

func TestNewPermissionsDataSource(t *testing.T) {
	if NewPermissionsDataSource() == nil {
		t.Fatal("NewPermissionsDataSource() returned nil")
	}
}

func TestPermissionFlagNamesSortedAndComplete(t *testing.T) {
	names := permissionFlagNames()
	if len(names) != len(discordPermissionBits) {
		t.Fatalf("got %d names, want %d", len(names), len(discordPermissionBits))
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Fatalf("names not sorted: %q then %q", names[i-1], names[i])
		}
	}
}
