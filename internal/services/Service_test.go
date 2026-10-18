package services

import (
	"errors"
	"testing"
	"time"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"gorm.io/gorm"
)

const (
	ownerID     = "owner"
	otherUserID = "other"
	suspendedID = "suspended"
)

type fakeRepository struct {
	ones        map[string]models.Initiative
	deleted     []string
	usersFailed bool
}

func (r *fakeRepository) CreateOne(one *models.Initiative) (*models.Initiative, error) {
	return one, nil
}

func (r *fakeRepository) ReadActiveOnes() ([]models.Initiative, error) {
	return nil, nil
}

func (r *fakeRepository) ReadOnesByOwner(string) ([]models.Initiative, error) {
	return nil, nil
}

func (r *fakeRepository) ReadOneByID(id string) (*models.Initiative, error) {
	one, ok := r.ones[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}

	return &one, nil
}

func (r *fakeRepository) ReadOneByEmail(string) (*models.Initiative, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeRepository) UpdateOne(one *models.Initiative) (*models.Initiative, error) {
	return one, nil
}

func (r *fakeRepository) DeleteOne(id string) error {
	r.deleted = append(r.deleted, id)
	return nil
}

func (r *fakeRepository) IsUserSuspended(id string) (bool, error) {
	if r.usersFailed {
		return false, errors.New("database unavailable")
	}

	return id == suspendedID, nil
}

func newService() (*Service, *fakeRepository) {
	suspendedAt := time.Now()
	repo := &fakeRepository{ones: map[string]models.Initiative{
		"active":       {ID: "active", Owner: ownerID},
		"hidden":       {ID: "hidden", Owner: ownerID, SuspendedAt: &suspendedAt},
		"from-blocked": {ID: "from-blocked", Owner: suspendedID},
	}}

	return &Service{Repo: repo}, repo
}

func assertCode(t *testing.T, err error, want apperrors.Code) {
	t.Helper()

	if want == "" {
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		return
	}

	appErr, ok := apperrors.From(err)
	if !ok || appErr.Code != want {
		t.Fatalf("err = %v, want code %s", err, want)
	}
}

func TestCreateOneTakesOwnerFromTokenAndBlocksSuspendedUsers(t *testing.T) {
	service, _ := newService()

	created, err := service.CreateOne(ownerID, &models.Initiative{Owner: otherUserID})
	assertCode(t, err, "")
	if created.Owner != ownerID {
		t.Fatalf("Owner = %q, want %q", created.Owner, ownerID)
	}

	_, err = service.CreateOne(suspendedID, &models.Initiative{})
	assertCode(t, err, apperrors.CodeAccountSuspended)
}

func TestGetOneByIDHidesSuspendedInitiativesFromOthers(t *testing.T) {
	service, _ := newService()

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
		_, err := service.GetOneByID(test.userID, test.id)
		assertCode(t, err, test.want)
	}
}

func TestChangesNeedAnActiveOwner(t *testing.T) {
	tests := []struct {
		userID, id string
		want       apperrors.Code
	}{
		{ownerID, "active", ""},
		{ownerID, "hidden", ""},
		{otherUserID, "active", apperrors.CodeForbidden},
		{suspendedID, "from-blocked", apperrors.CodeAccountSuspended},
		{ownerID, "missing", apperrors.CodeInitiativeNotFound},
	}

	for _, test := range tests {
		service, repo := newService()

		_, err := service.UpdateOne(test.userID, &models.Initiative{ID: test.id, Name: "changed"})
		assertCode(t, err, test.want)

		err = service.DeleteOne(test.userID, test.id)
		assertCode(t, err, test.want)

		if deleted := len(repo.deleted) > 0; deleted != (test.want == "") {
			t.Fatalf("%s deleting %s: deleted = %v", test.userID, test.id, deleted)
		}
	}
}

func TestSuspensionLookupFailureIsInternal(t *testing.T) {
	service, repo := newService()
	repo.usersFailed = true

	_, err := service.CreateOne(ownerID, &models.Initiative{})
	assertCode(t, err, apperrors.CodeInternalError)
}
