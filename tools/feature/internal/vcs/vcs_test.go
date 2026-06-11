package vcs

import "testing"

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Fix login redirect":               "fix-login-redirect",
		"Postgresql-HA wastes 32GB of RAM": "postgresql-ha-wastes-32gb-of-ram",
		"  Trim & punctuation!! ":          "trim-punctuation",
		"Enable CD via Jenkins":            "enable-cd-via-jenkins",
		"DIJO2 — Add AZ to VPC":            "dijo2-add-az-to-vpc",
		"":                                 "",
		"!!!":                              "",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBranch(t *testing.T) {
	if got := Branch("DRM-1", "Fix login"); got != "DRM-1-fix-login" {
		t.Errorf("Branch = %q", got)
	}
	if got := Branch("DRM-2", "!!!"); got != "DRM-2" {
		t.Errorf("Branch with empty slug = %q, want DRM-2", got)
	}
}
