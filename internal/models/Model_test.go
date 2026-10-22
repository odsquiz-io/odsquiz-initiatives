package models

import (
	"encoding/json"
	"testing"
)

func TestInitiativeDetailHidesLogoStorage(t *testing.T) {
	object := "initiatives/abc/logo-0123456789abcdef.png"
	contentType := "image/png"
	role := ViewerRoleOwner

	detail := InitiativeDetail{
		InitiativeResponse: NewInitiativeResponse(Initiative{ID: "abc", LogoObject: &object, LogoContentType: &contentType}),
		ViewerRole:         &role,
		MembersCount:       2,
	}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}

	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"id":            "abc",
		"description":   "",
		"logo_url":      "/api/initiatives/abc/logo?v=0123456789abcdef",
		"viewer_role":   "owner",
		"members_count": float64(2),
	}
	for key, value := range want {
		if fields[key] != value {
			t.Fatalf("%s = %v, want %v", key, fields[key], value)
		}
	}

	for _, key := range []string{"logo_object", "logo_content_type", "LogoObject", "LogoContentType"} {
		if _, ok := fields[key]; ok {
			t.Fatalf("%s is exposed", key)
		}
	}
}
