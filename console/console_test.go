package console

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		in, want, wantErr string
	}{
		{in: "/whitelist get", want: "/whitelist get"},
		{in: " /whitelist add bob \n", want: "/whitelist add bob"},
		{in: "/config get max-players", want: "/config get max-players"},
		{in: "/bans", want: "/bans"},
		{in: "/banlist get", want: "/banlist get"},
		{in: "/mutes", want: "/mutes"},
		{in: "/players", wantErr: "not allowed"},
		{in: "/server-save", wantErr: "not allowed"},
		{in: "/kick bob griefing", wantErr: "not allowed"},
		{in: "/ban bob", wantErr: "not allowed"},
		{in: "/sc rcon.print(1)", wantErr: "not allowed"},
		{in: "/c game.print(1)", wantErr: "not allowed"},
		{in: "/Whitelist get", wantErr: "not allowed"},
		{in: "/whitelisted", wantErr: "not allowed"},
		{in: "/", wantErr: "not allowed"},
		{in: "", wantErr: "empty"},
		{in: "\n", wantErr: "empty"},
		{in: "whitelist get", wantErr: "must start with /"},
		{in: "/bans\n/players", wantErr: "single line"},
		{in: "/bans\r/players", wantErr: "single line"},
	}
	for _, tc := range cases {
		got, err := Validate(tc.in)
		if tc.wantErr != "" {
			require.Error(t, err, "input %q", tc.in)
			assert.Contains(t, err.Error(), tc.wantErr, "input %q", tc.in)
			continue
		}
		require.NoError(t, err, "input %q", tc.in)
		assert.Equal(t, tc.want, got)
	}
}

func TestAllowedNames(t *testing.T) {
	assert.Equal(t, []string{"/banlist", "/bans", "/config", "/mutes", "/whitelist"}, AllowedNames())
}
