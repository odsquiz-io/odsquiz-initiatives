package services

import (
	"bytes"
	"context"
	"regexp"
	"testing"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/storage"
	"gorm.io/gorm"
)

var (
	pngLogo  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	jpegLogo = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
	webpLogo = []byte("RIFF\x24\x00\x00\x00WEBPVP8 \x18\x00\x00\x00")
)

type fakeStorage struct {
	objects     map[string][]byte
	unavailable bool
}

func (s *fakeStorage) Put(_ context.Context, name, _ string, data []byte) error {
	if s.unavailable {
		return storage.ErrUnavailable
	}

	s.objects[name] = data
	return nil
}

func (s *fakeStorage) Get(_ context.Context, name string) ([]byte, error) {
	if s.unavailable {
		return nil, storage.ErrUnavailable
	}

	data, ok := s.objects[name]
	if !ok {
		return nil, storage.ErrNotFound
	}

	return data, nil
}

func (s *fakeStorage) Delete(_ context.Context, name string) error {
	if s.unavailable {
		return storage.ErrUnavailable
	}

	if _, ok := s.objects[name]; !ok {
		return storage.ErrNotFound
	}

	delete(s.objects, name)
	return nil
}

type fakeLogos struct {
	repo *fakeRepository
}

func (l *fakeLogos) SetLogo(id string, object, contentType *string) error {
	one, ok := l.repo.ones[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}

	one.LogoObject = object
	one.LogoContentType = contentType
	l.repo.ones[id] = one
	return nil
}

func newLogoService() (*Service, *fakeStorage) {
	service, _ := newService()
	return service, service.Storage.(*fakeStorage)
}

func TestSetLogoAcceptsOnlySmallImages(t *testing.T) {
	tests := []struct {
		name, userID string
		data         []byte
		want         apperrors.Code
		contentType  string
	}{
		{"png", ownerID, pngLogo, "", "image/png"},
		{"jpeg", ownerID, jpegLogo, "", "image/jpeg"},
		{"webp", ownerID, webpLogo, "", "image/webp"},
		{"gif", ownerID, []byte("GIF89a\x01\x00\x01\x00"), apperrors.CodeInvalidImage, ""},
		{"svg", ownerID, []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), apperrors.CodeInvalidImage, ""},
		{"empty", ownerID, nil, apperrors.CodeInvalidImage, ""},
		{"too large", ownerID, append(bytes.Clone(pngLogo), make([]byte, MaxLogoSize)...), apperrors.CodeImageTooLarge, ""},
		{"not the owner", otherUserID, pngLogo, apperrors.CodeForbidden, ""},
	}

	for _, test := range tests {
		service, store := newLogoService()

		one, err := service.SetLogo(context.Background(), test.userID, "active", test.data)
		assertCode(t, err, test.want)
		if test.want != "" {
			if len(store.objects) != 0 {
				t.Fatalf("%s: stored %d objects", test.name, len(store.objects))
			}
			continue
		}

		if *one.LogoContentType != test.contentType {
			t.Fatalf("%s: content type = %q, want %q", test.name, *one.LogoContentType, test.contentType)
		}
		if !regexp.MustCompile(`^initiatives/active/logo-[0-9a-f]{16}\.(png|jpg|webp)$`).MatchString(*one.LogoObject) {
			t.Fatalf("%s: object = %q", test.name, *one.LogoObject)
		}
		if url := models.NewInitiativeResponse(*one).LogoURL; url == nil || *url != "/api/initiatives/active/logo?v="+models.LogoHash(one.LogoObject) {
			t.Fatalf("%s: logo_url = %v", test.name, url)
		}
	}
}

func TestSetLogoReplacesThePreviousObject(t *testing.T) {
	service, store := newLogoService()

	first, err := service.SetLogo(context.Background(), ownerID, "active", pngLogo)
	assertCode(t, err, "")
	firstObject := *first.LogoObject

	second, err := service.SetLogo(context.Background(), ownerID, "active", jpegLogo)
	assertCode(t, err, "")

	if _, ok := store.objects[firstObject]; ok {
		t.Fatal("previous logo was kept")
	}
	if _, ok := store.objects[*second.LogoObject]; !ok || len(store.objects) != 1 {
		t.Fatalf("objects = %v", store.objects)
	}

	_, err = service.SetLogo(context.Background(), ownerID, "active", jpegLogo)
	assertCode(t, err, "")
	if _, ok := store.objects[*second.LogoObject]; !ok {
		t.Fatal("uploading the same logo again removed it")
	}
}

func TestLogoEndpointsNeedStorage(t *testing.T) {
	service, store := newLogoService()

	_, err := service.SetLogo(context.Background(), ownerID, "active", pngLogo)
	assertCode(t, err, "")

	store.unavailable = true

	_, err = service.SetLogo(context.Background(), ownerID, "active", jpegLogo)
	assertCode(t, err, apperrors.CodeStorageUnavailable)

	_, err = service.RemoveLogo(context.Background(), ownerID, "active")
	assertCode(t, err, apperrors.CodeStorageUnavailable)

	logo, err := service.FindLogo("active")
	assertCode(t, err, "")
	_, err = service.ReadLogo(context.Background(), logo)
	assertCode(t, err, apperrors.CodeStorageUnavailable)
}

func TestRemoveLogoClearsObjectAndColumns(t *testing.T) {
	service, store := newLogoService()

	_, err := service.SetLogo(context.Background(), ownerID, "active", pngLogo)
	assertCode(t, err, "")

	_, err = service.RemoveLogo(context.Background(), otherUserID, "active")
	assertCode(t, err, apperrors.CodeForbidden)

	one, err := service.RemoveLogo(context.Background(), ownerID, "active")
	assertCode(t, err, "")
	if one.LogoObject != nil || one.LogoContentType != nil || len(store.objects) != 0 {
		t.Fatalf("logo left behind: %v %v %v", one.LogoObject, one.LogoContentType, store.objects)
	}

	_, err = service.FindLogo("active")
	assertCode(t, err, apperrors.CodeLogoNotFound)
}

func TestFindLogoHidesSuspendedInitiatives(t *testing.T) {
	service, _ := newLogoService()

	_, err := service.SetLogo(context.Background(), ownerID, "hidden", pngLogo)
	assertCode(t, err, "")

	_, err = service.FindLogo("hidden")
	assertCode(t, err, apperrors.CodeLogoNotFound)

	_, err = service.FindLogo("missing")
	assertCode(t, err, apperrors.CodeLogoNotFound)

	_, err = service.SetLogo(context.Background(), ownerID, "active", webpLogo)
	assertCode(t, err, "")

	logo, err := service.FindLogo("active")
	assertCode(t, err, "")
	data, err := service.ReadLogo(context.Background(), logo)
	assertCode(t, err, "")
	if logo.ContentType != "image/webp" || !bytes.Equal(data, webpLogo) {
		t.Fatalf("logo = %+v, %d bytes", logo, len(data))
	}
}
