package testutil

import (
	"testing"
)

func TestIntegrationEnabled(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"", false},
		{"false", false},
		{"0", false},
		{"no", false},
		{"true", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{" YES ", true},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("INTEGRATION_TESTS", tc.env)
			if got := IntegrationEnabled(); got != tc.want {
				t.Fatalf("IntegrationEnabled()=%v want %v for INTEGRATION_TESTS=%q", got, tc.want, tc.env)
			}
		})
	}
}

func TestSkipUnlessIntegration_skipsWhenUnset(t *testing.T) {
	t.Setenv("INTEGRATION_TESTS", "")
	done := false
	t.Run("child", func(t *testing.T) {
		SkipUnlessIntegration(t)
		done = true
	})
	if done {
		t.Fatal("expected child test to skip")
	}
}

func TestSkipUnlessIntegration_runsWhenEnabled(t *testing.T) {
	t.Setenv("INTEGRATION_TESTS", "true")
	done := false
	t.Run("child", func(t *testing.T) {
		SkipUnlessIntegration(t)
		done = true
	})
	if !done {
		t.Fatal("expected child test to run when INTEGRATION_TESTS=true")
	}
}
