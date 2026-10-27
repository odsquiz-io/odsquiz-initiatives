package services

import (
	"strings"
	"testing"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
)

func TestContactsAreNormalized(t *testing.T) {
	type field func(*models.Initiative) **string
	whatsapp := func(one *models.Initiative) **string { return &one.Whatsapp }
	instagram := func(one *models.Initiative) **string { return &one.Instagram }
	website := func(one *models.Initiative) **string { return &one.Website }

	tests := []struct {
		name, input, want string
		field             field
		code              apperrors.Code
	}{
		{"whatsapp mobile", "(11) 98765-4321", "5511987654321", whatsapp, ""},
		{"whatsapp landline", "11 3456-7890", "551134567890", whatsapp, ""},
		{"whatsapp with country code", "+55 11 98765-4321", "5511987654321", whatsapp, ""},
		{"whatsapp normalized", "551134567890", "551134567890", whatsapp, ""},
		{"whatsapp with DDD 55", "55 98765-4321", "5555987654321", whatsapp, ""},
		{"whatsapp blank", "   ", "", whatsapp, ""},
		{"whatsapp too short", "98765-4321", "", whatsapp, apperrors.CodeInvalidWhatsapp},
		{"whatsapp without digits", "abc", "", whatsapp, apperrors.CodeInvalidWhatsapp},
		{"whatsapp foreign", "+44 1198765-4321", "", whatsapp, apperrors.CodeInvalidWhatsapp},
		{"whatsapp too long", "+55 11 98765-43210", "", whatsapp, apperrors.CodeInvalidWhatsapp},
		{"instagram handle", "horta.viva", "horta.viva", instagram, ""},
		{"instagram at", " @horta_viva ", "horta_viva", instagram, ""},
		{"instagram host", "instagram.com/horta.viva", "horta.viva", instagram, ""},
		{"instagram url", "https://www.instagram.com/horta.viva/", "horta.viva", instagram, ""},
		{"instagram url with query", "https://www.instagram.com/horta.viva/?igsh=abc123", "horta.viva", instagram, ""},
		{"instagram url case", "HTTP://Instagram.com/HortaViva", "HortaViva", instagram, ""},
		{"instagram url with at", "instagram.com/@horta", "horta", instagram, ""},
		{"instagram blank", "", "", instagram, ""},
		{"instagram post url", "https://www.instagram.com/p/abc/", "", instagram, apperrors.CodeInvalidInstagram},
		{"instagram other site", "https://example.com/horta", "", instagram, apperrors.CodeInvalidInstagram},
		{"instagram space", "horta viva", "", instagram, apperrors.CodeInvalidInstagram},
		{"instagram only at", "@", "", instagram, apperrors.CodeInvalidInstagram},
		{"instagram too long", strings.Repeat("a", 31), "", instagram, apperrors.CodeInvalidInstagram},
		{"website without scheme", " horta.org.br ", "https://horta.org.br", website, ""},
		{"website http", "http://horta.org.br/sobre?x=1", "http://horta.org.br/sobre?x=1", website, ""},
		{"website upper-case scheme", "HTTPS://Horta.org.br", "HTTPS://Horta.org.br", website, ""},
		{"website at the limit", "a.br/" + strings.Repeat("a", 242), "https://a.br/" + strings.Repeat("a", 242), website, ""},
		{"website blank", " ", "", website, ""},
		{"website over the limit", "a.br/" + strings.Repeat("a", 243), "", website, apperrors.CodeInvalidWebsite},
		{"website without dot", "localhost:3000", "", website, apperrors.CodeInvalidWebsite},
		{"website other scheme", "ftp://horta.org.br", "", website, apperrors.CodeInvalidWebsite},
		{"website space", "https://horta org.br", "", website, apperrors.CodeInvalidWebsite},
		{"website script", "javascript:alert(1)", "", website, apperrors.CodeInvalidWebsite},
		{"website user info", "https://horta.org.br@example.com", "", website, apperrors.CodeInvalidWebsite},
		{"website without host", "https://", "", website, apperrors.CodeInvalidWebsite},
	}

	for _, test := range tests {
		service, _ := newService()
		one := &models.Initiative{}
		input := test.input
		*test.field(one) = &input

		_, err := service.CreateOne(ownerID, one)
		assertCode(t, err, test.code)
		if test.code != "" {
			continue
		}

		got := *test.field(one)
		if test.want == "" {
			if got != nil {
				t.Fatalf("%s: got %q, want null", test.name, *got)
			}
			continue
		}
		if got == nil || *got != test.want {
			t.Fatalf("%s: got %v, want %q", test.name, got, test.want)
		}
	}
}

func TestMissingContactsStayUntouched(t *testing.T) {
	service, _ := newService()
	website := "horta.org.br"
	one := &models.Initiative{ID: "active", Website: &website}

	_, err := service.UpdateOne(ownerID, one, "website")
	assertCode(t, err, "")
	if one.Whatsapp != nil || one.Instagram != nil || *one.Website != "https://horta.org.br" {
		t.Fatalf("contacts = %v %v %v", one.Whatsapp, one.Instagram, *one.Website)
	}

	invalid := "abc"
	_, err = service.UpdateOne(ownerID, &models.Initiative{ID: "active", Whatsapp: &invalid}, "whatsapp")
	assertCode(t, err, apperrors.CodeInvalidWhatsapp)
}
