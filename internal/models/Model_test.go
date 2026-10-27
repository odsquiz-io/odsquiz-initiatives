package models

import (
	"encoding/json"
	"testing"
)

func TestInitiativeDetailHidesLogoStorage(t *testing.T) {
	object := "initiatives/abc/logo-0123456789abcdef.png"
	contentType := "image/png"
	role := ViewerRoleOwner
	instagram := "horta.viva"

	detail := InitiativeDetail{
		InitiativeResponse: NewInitiativeResponse(Initiative{ID: "abc", LogoObject: &object, LogoContentType: &contentType, Instagram: &instagram}),
		ViewerRole:         &role,
		MembersCount:       2,
	}

	fields := encodeFields(t, detail)

	want := map[string]any{
		"id":            "abc",
		"description":   "",
		"logo_url":      "/api/initiatives/abc/logo?v=0123456789abcdef",
		"viewer_role":   "owner",
		"members_count": float64(2),
		"whatsapp":      nil,
		"instagram":     "horta.viva",
		"website":       nil,
	}
	for key, value := range want {
		if got, ok := fields[key]; !ok || got != value {
			t.Fatalf("%s = %v, want %v", key, got, value)
		}
	}

	for _, key := range []string{"logo_object", "logo_content_type", "LogoObject", "LogoContentType"} {
		if _, ok := fields[key]; ok {
			t.Fatalf("%s is exposed", key)
		}
	}
}

func TestLogoHashAcceptsOnlyLogoObjectNames(t *testing.T) {
	tests := map[string]string{
		"initiatives/abc/logo-0123456789abcdef.png":  "0123456789abcdef",
		"initiatives/abc/logo-0123456789abcdef.webp": "0123456789abcdef",
		"initiatives/abc/logo-.png":                  "",
		"initiatives/abc/photo-0123456789abcdef.png": "",
		"initiatives/abc/logo-0123456789ABCDEF.png":  "",
		"initiatives/abc/logo-01234567&v=x.png":      "",
		"":                                           "",
	}

	for object, want := range tests {
		if got := LogoHash(&object); got != want {
			t.Errorf("LogoHash(%q) = %q, want %q", object, got, want)
		}
	}

	if LogoHash(nil) != "" || LogoURL("abc", nil) != nil {
		t.Fatal("missing logo must have no hash and no URL")
	}
}

func TestPhotoMatchesContract(t *testing.T) {
	photo := InitiativePhoto{
		ID:           "11111111-1111-4111-8111-111111111111",
		InitiativeID: "abc",
		Object:       "initiatives/abc/photos/11111111-1111-4111-8111-111111111111-0123456789abcdef.webp",
		ContentType:  "image/webp",
		Position:     3,
	}

	fields := encodeFields(t, NewPhoto(photo))

	want := map[string]any{
		"id":       photo.ID,
		"url":      "/api/initiatives/abc/photos/11111111-1111-4111-8111-111111111111/image?v=0123456789abcdef",
		"caption":  nil,
		"position": float64(3),
	}
	if len(fields) != len(want) {
		t.Fatalf("photo fields = %v, want %v", fields, want)
	}
	for key, value := range want {
		if got, ok := fields[key]; !ok || got != value {
			t.Fatalf("%s = %v, want %v", key, got, value)
		}
	}

	photo.Object = "initiatives/abc/photos/22222222-2222-4222-8222-222222222222-0123456789abcdef.webp"
	if photo.Hash() != "" {
		t.Fatal("an object named after another photo must have no hash")
	}
}

func encodeFields(t *testing.T, value any) map[string]any {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}

	return fields
}
