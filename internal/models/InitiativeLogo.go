package models

import (
	"path"
	"strings"
)

type InitiativeSummary struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	MainODS int     `json:"main_ods"`
	City    string  `json:"city"`
	State   string  `json:"state"`
	LogoURL *string `json:"logo_url"`
}

// LogoHash extracts the hash from an object named "initiatives/<id>/logo-<hash>.<ext>".
func LogoHash(logoObject *string) string {
	if logoObject == nil {
		return ""
	}

	return objectHash(*logoObject, "logo-")
}

// objectHash extracts the content hash that follows the prefix in a stored image's file name.
func objectHash(object, prefix string) string {
	name := path.Base(object)
	hash, ok := strings.CutPrefix(strings.TrimSuffix(name, path.Ext(name)), prefix)
	if !ok || hash == "" || strings.Trim(hash, "0123456789abcdef") != "" {
		return ""
	}

	return hash
}

func LogoURL(initiativeID string, logoObject *string) *string {
	hash := LogoHash(logoObject)
	if hash == "" {
		return nil
	}

	url := "/api/initiatives/" + initiativeID + "/logo?v=" + hash
	return &url
}
