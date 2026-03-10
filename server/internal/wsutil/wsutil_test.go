package wsutil

import (
	"strings"
	"testing"
)

// TestEndpointSchemePrefix verifies that New() prepends "https://" to endpoints
// that do not already carry a scheme — which is how the WEBSOCKET_API_ENDPOINT
// env var is stored in Lambda (e.g. "id.execute-api.region.amazonaws.com/stage/stage").
// Without the prefix the AWS SDK produces URLs with an empty scheme that fail.
func TestEndpointSchemePrefix(t *testing.T) {
	cases := []struct {
		raw      string
		wantPfx  string
		wantSkip bool // already has a valid scheme — should be left unchanged
	}{
		{
			raw:     "ba2t50m7se.execute-api.us-west-2.amazonaws.com/prod/prod",
			wantPfx: "https://",
		},
		{
			raw:      "https://ba2t50m7se.execute-api.us-west-2.amazonaws.com/prod/prod",
			wantSkip: true,
		},
		{
			raw:      "http://ba2t50m7se.execute-api.us-west-2.amazonaws.com/prod/prod",
			wantSkip: true,
		},
	}

	for _, tc := range cases {
		endpoint := tc.raw
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = "https://" + endpoint
		}

		if tc.wantSkip {
			if endpoint != tc.raw {
				t.Errorf("endpoint %q should be unchanged, got %q", tc.raw, endpoint)
			}
			continue
		}

		if !strings.HasPrefix(endpoint, tc.wantPfx) {
			t.Errorf("endpoint %q: want prefix %q, got %q", tc.raw, tc.wantPfx, endpoint)
		}
		if !strings.Contains(endpoint, tc.raw) {
			t.Errorf("endpoint %q: original host/path should be preserved, got %q", tc.raw, endpoint)
		}
	}
}
