package services

import (
	"bytes"
	"cmp"
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/repositories"
	"gorm.io/gorm"
)

type fakePhotos struct {
	repo       *fakeRepository
	rows       map[string]models.InitiativePhoto
	failCreate error
}

func (r *fakePhotos) ListPhotos(initiativeID string) ([]models.InitiativePhoto, error) {
	photos := []models.InitiativePhoto{}
	for _, photo := range r.rows {
		if photo.InitiativeID == initiativeID {
			photos = append(photos, photo)
		}
	}

	slices.SortFunc(photos, func(a, b models.InitiativePhoto) int { return cmp.Compare(a.Position, b.Position) })
	return photos, nil
}

func (r *fakePhotos) ReadPhoto(initiativeID, photoID string) (*models.InitiativePhoto, error) {
	photo, ok := r.rows[photoID]
	if !ok || photo.InitiativeID != initiativeID {
		return nil, gorm.ErrRecordNotFound
	}

	return &photo, nil
}

func (r *fakePhotos) CreatePhoto(photo *models.InitiativePhoto, limit int) error {
	if r.failCreate != nil {
		return r.failCreate
	}

	photos, _ := r.ListPhotos(photo.InitiativeID)
	if len(photos) >= limit {
		return repositories.ErrPhotoLimitReached
	}

	photo.Position = len(photos)
	r.rows[photo.ID] = *photo
	return nil
}

func (r *fakePhotos) UpdateCaption(initiativeID, photoID string, caption *string) error {
	photo, err := r.ReadPhoto(initiativeID, photoID)
	if err != nil {
		return err
	}

	photo.Caption = caption
	r.rows[photoID] = *photo
	return nil
}

func (r *fakePhotos) ReorderPhotos(initiativeID string, photoIDs []string) error {
	for position, photoID := range photoIDs {
		photo, err := r.ReadPhoto(initiativeID, photoID)
		if err != nil {
			return repositories.ErrPhotosChanged
		}

		photo.Position = position
		r.rows[photoID] = *photo
	}

	return nil
}

func (r *fakePhotos) DeletePhoto(initiativeID, photoID string) (*models.InitiativePhoto, error) {
	photo, err := r.ReadPhoto(initiativeID, photoID)
	if err != nil {
		return nil, err
	}

	delete(r.rows, photoID)
	for id, other := range r.rows {
		if other.InitiativeID == initiativeID && other.Position > photo.Position {
			other.Position--
			r.rows[id] = other
		}
	}

	return photo, nil
}

func (r *fakePhotos) deleteInitiative(initiativeID string) []string {
	objects := []string{}
	for id, photo := range r.rows {
		if photo.InitiativeID == initiativeID {
			objects = append(objects, photo.Object)
			delete(r.rows, id)
		}
	}

	return objects
}

func newPhotosService() (*Service, *fakePhotos, *fakeStorage) {
	service, repo := newService()
	return service, repo.photos, service.Storage.(*fakeStorage)
}

func addPhotos(t *testing.T, service *Service, id string, count int) []models.Photo {
	t.Helper()

	photos := make([]models.Photo, 0, count)
	for range count {
		photo, err := service.AddPhoto(context.Background(), ownerID, id, pngLogo, nil)
		assertCode(t, err, "")
		photos = append(photos, *photo)
	}

	return photos
}

func photoIDs(photos []models.Photo) []string {
	ids := make([]string, 0, len(photos))
	for _, photo := range photos {
		ids = append(ids, photo.ID)
	}

	return ids
}

func TestAddPhotoAppendsUpToTheLimit(t *testing.T) {
	service, photos, store := newPhotosService()

	added := addPhotos(t, service, "active", MaxPhotos)
	object := regexp.MustCompile(`^initiatives/active/photos/[0-9a-f-]{36}-[0-9a-f]{16}\.png$`)
	for position, photo := range added {
		row := photos.rows[photo.ID]
		if photo.Position != position || !object.MatchString(row.Object) || row.ContentType != "image/png" {
			t.Fatalf("photo %d = %+v, row %+v", position, photo, row)
		}
		if photo.URL != "/api/initiatives/active/photos/"+photo.ID+"/image?v="+row.Hash() {
			t.Fatalf("url = %q", photo.URL)
		}
	}

	_, err := service.AddPhoto(context.Background(), ownerID, "active", pngLogo, nil)
	assertCode(t, err, apperrors.CodePhotoLimitReached)
	if len(store.objects) != MaxPhotos || len(photos.rows) != MaxPhotos {
		t.Fatalf("%d objects and %d rows after the limit", len(store.objects), len(photos.rows))
	}
}

func TestAddPhotoRemovesTheImageWhenTheRowFails(t *testing.T) {
	service, photos, store := newPhotosService()
	photos.failCreate = repositories.ErrPhotoLimitReached

	_, err := service.AddPhoto(context.Background(), ownerID, "active", pngLogo, nil)
	assertCode(t, err, apperrors.CodePhotoLimitReached)
	if len(store.objects) != 0 {
		t.Fatalf("objects = %v", store.objects)
	}
}

func TestAddPhotoRules(t *testing.T) {
	long := strings.Repeat("é", maxCaptionLength+1)
	tests := []struct {
		name, userID, id string
		data             []byte
		caption          string
		unavailable      bool
		want             apperrors.Code
	}{
		{"owner", ownerID, "active", jpegLogo, "", false, ""},
		{"owner of a suspended initiative", ownerID, "hidden", webpLogo, "", false, ""},
		{"not the owner", otherUserID, "active", pngLogo, "", false, apperrors.CodeForbidden},
		{"suspended owner", suspendedID, "from-blocked", pngLogo, "", false, apperrors.CodeAccountSuspended},
		{"missing initiative", ownerID, "missing", pngLogo, "", false, apperrors.CodeInitiativeNotFound},
		{"gif", ownerID, "active", []byte("GIF89a\x01\x00\x01\x00"), "", false, apperrors.CodeInvalidImage},
		{"too large", ownerID, "active", append(bytes.Clone(pngLogo), make([]byte, MaxImageSize)...), "", false, apperrors.CodeImageTooLarge},
		{"caption too long", ownerID, "active", pngLogo, long, false, apperrors.CodeInvalidRequest},
		{"storage unavailable", ownerID, "active", pngLogo, "", true, apperrors.CodeStorageUnavailable},
	}

	for _, test := range tests {
		service, photos, store := newPhotosService()
		store.unavailable = test.unavailable

		_, err := service.AddPhoto(context.Background(), test.userID, test.id, test.data, &test.caption)
		assertCode(t, err, test.want)

		stored := 0
		if test.want == "" {
			stored = 1
		}
		if len(photos.rows) != stored || len(store.objects) != stored {
			t.Fatalf("%s: %d rows and %d objects", test.name, len(photos.rows), len(store.objects))
		}
	}
}

func TestPhotoCaptionIsTrimmedAndBounded(t *testing.T) {
	limit := strings.Repeat("é", maxCaptionLength)
	tests := []struct {
		name, userID string
		caption      *string
		want         *string
		code         apperrors.Code
	}{
		{"trimmed", ownerID, new("  Mutirão de plantio  "), new("Mutirão de plantio"), ""},
		{"at the limit", ownerID, new(" " + limit + " "), &limit, ""},
		{"blank clears", ownerID, new("   "), nil, ""},
		{"null clears", ownerID, nil, nil, ""},
		{"too long", ownerID, new(limit + "é"), nil, apperrors.CodeInvalidRequest},
		{"not the owner", otherUserID, new("Outra"), nil, apperrors.CodeForbidden},
	}

	for _, test := range tests {
		service, photos, _ := newPhotosService()
		added, err := service.AddPhoto(context.Background(), ownerID, "active", pngLogo, new("Antiga"))
		assertCode(t, err, "")

		photo, err := service.UpdatePhotoCaption(test.userID, "active", added.ID, test.caption)
		assertCode(t, err, test.code)
		if test.code != "" {
			if caption := photos.rows[added.ID].Caption; caption == nil || *caption != "Antiga" {
				t.Fatalf("%s: caption changed to %v", test.name, caption)
			}
			continue
		}

		if (photo.Caption == nil) != (test.want == nil) || (photo.Caption != nil && *photo.Caption != *test.want) {
			t.Fatalf("%s: caption = %v, want %v", test.name, photo.Caption, test.want)
		}
	}

	service, _, _ := newPhotosService()
	added := addPhotos(t, service, "active", 1)
	_, err := service.UpdatePhotoCaption(ownerID, "hidden", added[0].ID, nil)
	assertCode(t, err, apperrors.CodePhotoNotFound)
}

func TestReorderPhotosNeedsAPermutation(t *testing.T) {
	service, _, _ := newPhotosService()
	ids := photoIDs(addPhotos(t, service, "active", 3))
	other := photoIDs(addPhotos(t, service, "hidden", 1))

	tests := []struct {
		name, userID string
		order        []string
		want         apperrors.Code
	}{
		{"missing photo", ownerID, ids[:2], apperrors.CodeInvalidRequest},
		{"repeated photo", ownerID, []string{ids[0], ids[0], ids[1]}, apperrors.CodeInvalidRequest},
		{"extra photo", ownerID, append(slices.Clone(ids), ids[0]), apperrors.CodeInvalidRequest},
		{"photo of another initiative", ownerID, []string{ids[0], ids[1], other[0]}, apperrors.CodeInvalidRequest},
		{"empty", ownerID, nil, apperrors.CodeInvalidRequest},
		{"not the owner", otherUserID, []string{ids[2], ids[0], ids[1]}, apperrors.CodeForbidden},
		{"permutation", ownerID, []string{ids[2], ids[0], ids[1]}, ""},
	}

	for _, test := range tests {
		reordered, err := service.ReorderPhotos(test.userID, "active", test.order)
		assertCode(t, err, test.want)
		if test.want != "" {
			continue
		}

		if !slices.Equal(photoIDs(reordered), test.order) {
			t.Fatalf("order = %v, want %v", photoIDs(reordered), test.order)
		}
		for position, photo := range reordered {
			if photo.Position != position {
				t.Fatalf("photo %s at position %d, want %d", photo.ID, photo.Position, position)
			}
		}
	}
}

func TestDeletePhotoClosesTheGap(t *testing.T) {
	service, photos, store := newPhotosService()
	ids := photoIDs(addPhotos(t, service, "active", 3))
	cover := photos.rows[ids[0]].Object

	assertCode(t, service.DeletePhoto(context.Background(), otherUserID, "active", ids[0]), apperrors.CodeForbidden)
	assertCode(t, service.DeletePhoto(context.Background(), ownerID, "hidden", ids[0]), apperrors.CodePhotoNotFound)
	assertCode(t, service.DeletePhoto(context.Background(), ownerID, "active", ids[0]), "")
	assertCode(t, service.DeletePhoto(context.Background(), ownerID, "active", ids[0]), apperrors.CodePhotoNotFound)

	left, err := service.ListPhotos(ownerID, "active")
	assertCode(t, err, "")
	if !slices.Equal(photoIDs(left), ids[1:]) || left[0].Position != 0 || left[1].Position != 1 {
		t.Fatalf("photos left = %+v", left)
	}
	if _, ok := store.objects[cover]; ok || len(store.objects) != 2 {
		t.Fatalf("objects = %v", store.objects)
	}

	store.unavailable = true
	assertCode(t, service.DeletePhoto(context.Background(), ownerID, "active", ids[1]), "")
	if len(photos.rows) != 1 {
		t.Fatalf("rows = %v", photos.rows)
	}
}

func TestPhotosFollowInitiativeVisibility(t *testing.T) {
	service, _, _ := newPhotosService()
	active := addPhotos(t, service, "active", 1)
	hidden := addPhotos(t, service, "hidden", 1)

	tests := []struct {
		userID, id string
		want       apperrors.Code
	}{
		{otherUserID, "active", ""},
		{ownerID, "hidden", ""},
		{otherUserID, "hidden", apperrors.CodeInitiativeNotFound},
		{otherUserID, "missing", apperrors.CodeInitiativeNotFound},
	}
	for _, test := range tests {
		photos, err := service.ListPhotos(test.userID, test.id)
		assertCode(t, err, test.want)
		if test.want == "" && len(photos) != 1 {
			t.Fatalf("%s listing %s: %d photos", test.userID, test.id, len(photos))
		}
	}

	_, err := service.FindPhoto("hidden", hidden[0].ID)
	assertCode(t, err, apperrors.CodePhotoNotFound)
	_, err = service.FindPhoto("active", hidden[0].ID)
	assertCode(t, err, apperrors.CodePhotoNotFound)
	_, err = service.FindPhoto("missing", active[0].ID)
	assertCode(t, err, apperrors.CodePhotoNotFound)

	image, err := service.FindPhoto("active", active[0].ID)
	assertCode(t, err, "")
	data, err := service.ReadPhoto(context.Background(), image)
	assertCode(t, err, "")
	if image.ContentType != "image/png" || !bytes.Equal(data, pngLogo) || !strings.HasSuffix(active[0].URL, "?v="+image.Hash) {
		t.Fatalf("image = %+v, %d bytes", image, len(data))
	}
}

func TestDeleteOneRemovesPhotoAndLogoObjects(t *testing.T) {
	service, photos, store := newPhotosService()
	addPhotos(t, service, "active", 2)
	addPhotos(t, service, "hidden", 1)
	_, err := service.SetLogo(context.Background(), ownerID, "active", jpegLogo)
	assertCode(t, err, "")

	assertCode(t, service.DeleteOne(ownerID, "active"), "")

	if len(store.objects) != 1 || len(photos.rows) != 1 {
		t.Fatalf("objects = %v, rows = %v", store.objects, photos.rows)
	}
}
