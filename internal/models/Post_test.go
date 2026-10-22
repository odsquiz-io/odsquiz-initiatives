package models

import (
	"encoding/json"
	"slices"
	"testing"
	"time"
)

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

func TestPostResponseMatchesContract(t *testing.T) {
	data, err := json.Marshal(PostPage{Items: []PostResponse{{
		ID:         "p",
		Initiative: InitiativeSummary{ID: "i", Name: "Horta"},
		Author:     PostAuthor{ID: "u", Name: "Ana"},
		Body:       "texto",
		CreatedAt:  time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC),
	}}})
	if err != nil {
		t.Fatal(err)
	}

	var page struct {
		Items []map[string]json.RawMessage `json:"items"`
		Next  json.RawMessage              `json:"next_cursor"`
	}
	if err := json.Unmarshal(data, &page); err != nil {
		t.Fatal(err)
	}

	item := page.Items[0]
	var keys []string
	for key := range item {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	want := []string{"author", "body", "comments_count", "created_at", "edited_at", "id", "initiative", "viewer_can_delete", "viewer_can_edit"}
	if !slices.Equal(keys, want) {
		t.Fatalf("post keys = %v, want %v", keys, want)
	}

	if string(item["edited_at"]) != "null" || string(page.Next) != "null" ||
		string(item["initiative"]) != `{"id":"i","name":"Horta","main_ods":0,"city":"","state":"","logo_url":null}` {
		t.Fatalf("unexpected encoding %s", data)
	}
}
