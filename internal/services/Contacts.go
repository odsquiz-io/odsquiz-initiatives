package services

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
)

const maxWebsiteLength = 255

var (
	instagramURL    = regexp.MustCompile(`^(?i:(?:https?://)?(?:www\.)?instagram\.com)/([^/?#]*)/?(?:[?#].*)?$`)
	instagramHandle = regexp.MustCompile(`^[A-Za-z0-9._]{1,30}$`)
)

// normalizeContacts stores WhatsApp as 55 + DDD + number, Instagram as the bare handle and the site as
// an absolute http(s) URL; a blank value clears the field and a nil one is left unchanged.
func normalizeContacts(one *models.Initiative) error {
	contacts := []struct {
		value     **string
		normalize func(string) (string, bool)
		invalid   apperrors.Code
	}{
		{&one.Whatsapp, normalizeWhatsapp, apperrors.CodeInvalidWhatsapp},
		{&one.Instagram, normalizeInstagram, apperrors.CodeInvalidInstagram},
		{&one.Website, normalizeWebsite, apperrors.CodeInvalidWebsite},
	}

	for _, contact := range contacts {
		if *contact.value == nil {
			continue
		}

		value := strings.TrimSpace(**contact.value)
		if value == "" {
			*contact.value = nil
			continue
		}

		normalized, ok := contact.normalize(value)
		if !ok {
			return apperrors.BadRequest(contact.invalid, nil)
		}

		*contact.value = &normalized
	}

	return nil
}

func normalizeWhatsapp(value string) (string, bool) {
	digits := strings.Map(func(char rune) rune {
		if char >= '0' && char <= '9' {
			return char
		}
		return -1
	}, value)

	if len(digits) == 10 || len(digits) == 11 {
		digits = "55" + digits
	}

	return digits, (len(digits) == 12 || len(digits) == 13) && strings.HasPrefix(digits, "55")
}

func normalizeInstagram(value string) (string, bool) {
	if match := instagramURL.FindStringSubmatch(value); match != nil {
		value = match[1]
	}

	handle := strings.TrimPrefix(value, "@")
	return handle, instagramHandle.MatchString(handle)
}

func normalizeWebsite(value string) (string, bool) {
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}

	if strings.ContainsFunc(value, unicode.IsSpace) || utf8.RuneCountInString(value) > maxWebsiteLength {
		return "", false
	}

	address, err := url.Parse(value)
	if err != nil || (address.Scheme != "http" && address.Scheme != "https") || address.User != nil {
		return "", false
	}

	return value, strings.Contains(address.Hostname(), ".")
}
