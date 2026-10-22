package services

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"gorm.io/gorm"
)

const strangerID = "stranger"

type fakeMembers struct {
	users   map[string]models.MemberAccount
	members map[string]models.InitiativeMember
	created int
}

func newFakeMembers() *fakeMembers {
	suspendedAt := time.Now()

	return &fakeMembers{
		users: map[string]models.MemberAccount{
			ownerID:     {ID: ownerID, Name: "Owner", Email: "owner@example.com"},
			otherUserID: {ID: otherUserID, Name: "Other", Email: "other@example.com"},
			strangerID:  {ID: strangerID, Name: "Stranger", Email: "stranger@example.com"},
			suspendedID: {ID: suspendedID, Name: "Suspended", Email: "suspended@example.com", SuspendedAt: &suspendedAt},
		},
		members: map[string]models.InitiativeMember{},
	}
}

func (r *fakeMembers) FindUser(id string) (*models.MemberAccount, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}

	return &user, nil
}

func (r *fakeMembers) FindUserByEmail(email string) (*models.MemberAccount, error) {
	for _, user := range r.users {
		if strings.EqualFold(user.Email, email) {
			return &user, nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

func (r *fakeMembers) CreateMember(member *models.InitiativeMember) error {
	if _, err := r.ReadMemberByUser(member.InitiativeID, member.UserID); err == nil {
		return gorm.ErrDuplicatedKey
	}

	r.created++
	r.members[member.ID] = *member
	return nil
}

func (r *fakeMembers) ReadMember(id string) (*models.InitiativeMember, error) {
	member, ok := r.members[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}

	return &member, nil
}

func (r *fakeMembers) ReadMemberByUser(initiativeID, userID string) (*models.InitiativeMember, error) {
	for _, member := range r.members {
		if member.InitiativeID == initiativeID && member.UserID == userID {
			return &member, nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

func (r *fakeMembers) IsActiveMember(initiativeID, userID string) (bool, error) {
	member, err := r.ReadMemberByUser(initiativeID, userID)
	return err == nil && member.Status == models.MemberStatusActive, nil
}

func (r *fakeMembers) CountActiveMembers(initiativeID string) (int64, error) {
	var count int64
	for _, member := range r.members {
		if member.InitiativeID == initiativeID && member.Status == models.MemberStatusActive {
			count++
		}
	}

	return count, nil
}

func (r *fakeMembers) ListMembers(initiativeID string, withPending bool) ([]models.Member, error) {
	members := []models.Member{}
	for _, member := range r.members {
		if member.InitiativeID != initiativeID || (!withPending && member.Status != models.MemberStatusActive) {
			continue
		}

		user := r.users[member.UserID]
		members = append(members, models.Member{
			ID:     member.ID,
			Status: member.Status,
			User:   models.MemberUser{ID: user.ID, Name: user.Name, Email: user.Email},
		})
	}

	slices.SortFunc(members, func(a, b models.Member) int { return strings.Compare(a.ID, b.ID) })
	return members, nil
}

func (r *fakeMembers) ListInvitations(string) ([]models.Invitation, error) {
	return nil, nil
}

func (r *fakeMembers) ListMemberships(string) ([]models.Membership, error) {
	return nil, nil
}

func (r *fakeMembers) AcceptMember(id string, respondedAt time.Time) error {
	member, ok := r.members[id]
	if !ok || member.Status != models.MemberStatusPending {
		return gorm.ErrRecordNotFound
	}

	member.Status = models.MemberStatusActive
	member.RespondedAt = &respondedAt
	r.members[id] = member
	return nil
}

func (r *fakeMembers) DeleteMember(id string) error {
	if _, ok := r.members[id]; !ok {
		return gorm.ErrRecordNotFound
	}

	delete(r.members, id)
	return nil
}

func (r *fakeMembers) add(id, initiativeID, userID, status string) {
	r.members[id] = models.InitiativeMember{
		ID:           id,
		InitiativeID: initiativeID,
		UserID:       userID,
		Status:       status,
		InvitedBy:    ownerID,
	}
}

func newMembersService() (*Service, *fakeMembers) {
	service, _ := newService()
	return service, service.Members.(*fakeMembers)
}

func TestInviteMemberRules(t *testing.T) {
	tests := []struct {
		name, userID, id, email string
		want                    apperrors.Code
	}{
		{"case-insensitive and trimmed", ownerID, "active", "  Other@Example.COM ", ""},
		{"empty e-mail", ownerID, "active", "  ", apperrors.CodeInvalidRequest},
		{"unknown e-mail", ownerID, "active", "nobody@example.com", apperrors.CodeUserNotFound},
		{"self", ownerID, "active", "owner@example.com", apperrors.CodeCannotInviteSelf},
		{"suspended target", ownerID, "active", "suspended@example.com", apperrors.CodeTargetSuspended},
		{"pending invitee", ownerID, "active", "stranger@example.com", apperrors.CodeAlreadyMember},
		{"not the owner", otherUserID, "active", "stranger@example.com", apperrors.CodeForbidden},
		{"suspended owner", suspendedID, "from-blocked", "other@example.com", apperrors.CodeAccountSuspended},
		{"missing initiative", ownerID, "missing", "other@example.com", apperrors.CodeInitiativeNotFound},
	}

	for _, test := range tests {
		service, members := newMembersService()
		members.add("pending-stranger", "active", strangerID, models.MemberStatusPending)

		member, err := service.InviteMember(test.userID, test.id, test.email)
		assertCode(t, err, test.want)
		if test.want != "" {
			if members.created != 0 {
				t.Fatalf("%s: created %d invitations", test.name, members.created)
			}
			continue
		}

		if member.Status != models.MemberStatusPending || member.User.ID != otherUserID || member.User.Email == "" {
			t.Fatalf("%s: member = %+v", test.name, member)
		}
		if stored, _ := members.ReadMember(member.ID); stored == nil || stored.InvitedBy != ownerID {
			t.Fatalf("%s: stored = %+v", test.name, stored)
		}
	}
}

func TestInviteMemberRejectsActiveMembers(t *testing.T) {
	service, members := newMembersService()
	members.add("active-other", "active", otherUserID, models.MemberStatusActive)

	_, err := service.InviteMember(ownerID, "active", "other@example.com")
	assertCode(t, err, apperrors.CodeAlreadyMember)
}

func TestOnlyTheInviteeAcceptsAnInvitation(t *testing.T) {
	service, members := newMembersService()
	members.add("invite", "active", otherUserID, models.MemberStatusPending)
	members.add("hidden-invite", "hidden", otherUserID, models.MemberStatusPending)
	members.add("blocked-invite", "active", suspendedID, models.MemberStatusPending)

	tests := []struct {
		userID, memberID string
		want             apperrors.Code
	}{
		{ownerID, "invite", apperrors.CodeInvitationNotFound},
		{strangerID, "invite", apperrors.CodeInvitationNotFound},
		{otherUserID, "missing", apperrors.CodeInvitationNotFound},
		{otherUserID, "hidden-invite", apperrors.CodeInvitationNotFound},
		{suspendedID, "blocked-invite", apperrors.CodeAccountSuspended},
		{otherUserID, "invite", ""},
		{otherUserID, "invite", apperrors.CodeInvitationNotFound},
	}

	for _, test := range tests {
		accepted, err := service.AcceptInvitation(test.userID, test.memberID)
		assertCode(t, err, test.want)
		if test.want == "" && (accepted.Status != models.MemberStatusActive || accepted.Initiative.ID != "active") {
			t.Fatalf("accepted = %+v", accepted)
		}
	}

	if member := members.members["invite"]; member.Status != models.MemberStatusActive || member.RespondedAt == nil {
		t.Fatalf("member = %+v", member)
	}
}

func TestOnlyTheInviteeDeclinesAnInvitation(t *testing.T) {
	service, members := newMembersService()
	members.add("invite", "active", otherUserID, models.MemberStatusPending)
	members.add("joined", "active", strangerID, models.MemberStatusActive)

	assertCode(t, service.DeclineInvitation(ownerID, "invite"), apperrors.CodeInvitationNotFound)
	assertCode(t, service.DeclineInvitation(strangerID, "invite"), apperrors.CodeInvitationNotFound)
	assertCode(t, service.DeclineInvitation(strangerID, "joined"), apperrors.CodeInvitationNotFound)
	assertCode(t, service.DeclineInvitation(otherUserID, "invite"), "")

	if _, ok := members.members["invite"]; ok {
		t.Fatal("declined invitation was kept")
	}
	if _, ok := members.members["joined"]; !ok {
		t.Fatal("active membership was removed")
	}
}

func TestOwnerRemovesMembersAndMembersLeave(t *testing.T) {
	tests := []struct {
		name, userID, id, memberID string
		want                       apperrors.Code
	}{
		{"owner removes", ownerID, "active", "joined", ""},
		{"owner cancels invitation", ownerID, "active", "invite", ""},
		{"member leaves", otherUserID, "active", "joined", ""},
		{"invitee cannot remove another member", strangerID, "active", "joined", apperrors.CodeForbidden},
		{"other initiative", ownerID, "hidden", "joined", apperrors.CodeMembershipNotFound},
		{"missing membership", ownerID, "active", "missing", apperrors.CodeMembershipNotFound},
	}

	for _, test := range tests {
		service, members := newMembersService()
		members.add("joined", "active", otherUserID, models.MemberStatusActive)
		members.add("invite", "active", strangerID, models.MemberStatusPending)

		err := service.RemoveMember(test.userID, test.id, test.memberID)
		assertCode(t, err, test.want)

		want := 2
		if test.want == "" {
			want = 1
		}
		if len(members.members) != want {
			t.Fatalf("%s: %d rows left, want %d", test.name, len(members.members), want)
		}
	}
}

func TestGetOneByIDReportsViewerRoleAndMembersCount(t *testing.T) {
	service, members := newMembersService()
	members.add("joined", "active", otherUserID, models.MemberStatusActive)
	members.add("invite", "active", strangerID, models.MemberStatusPending)

	tests := []struct {
		userID, want string
	}{
		{ownerID, models.ViewerRoleOwner},
		{otherUserID, models.ViewerRoleMember},
		{strangerID, ""},
		{suspendedID, ""},
	}

	for _, test := range tests {
		detail, err := service.GetOneByID(test.userID, "active")
		assertCode(t, err, "")

		role := ""
		if detail.ViewerRole != nil {
			role = *detail.ViewerRole
		}
		if role != test.want {
			t.Fatalf("%s: viewer_role = %q, want %q", test.userID, role, test.want)
		}
		if detail.MembersCount != 1 {
			t.Fatalf("%s: members_count = %d, want 1", test.userID, detail.MembersCount)
		}
	}
}

func TestListMembersShowsInvitationsAndEmailsOnlyToTheOwner(t *testing.T) {
	service, members := newMembersService()
	members.add("joined", "active", otherUserID, models.MemberStatusActive)
	members.add("invite", "active", strangerID, models.MemberStatusPending)

	list, err := service.ListMembers(ownerID, "active")
	assertCode(t, err, "")
	if list.Owner.Name != "Owner" || len(list.Members) != 2 || list.Members[0].User.Email == "" {
		t.Fatalf("owner view = %+v", list)
	}

	list, err = service.ListMembers(strangerID, "active")
	assertCode(t, err, "")
	if len(list.Members) != 1 || list.Members[0].ID != "joined" || list.Members[0].User.Email != "" {
		t.Fatalf("member view = %+v", list)
	}

	_, err = service.ListMembers(otherUserID, "hidden")
	assertCode(t, err, apperrors.CodeInitiativeNotFound)
}

func TestDescriptionIsTrimmedAndBounded(t *testing.T) {
	service, _ := newService()

	description := "  Sobre a iniciativa  "
	updated := &models.Initiative{ID: "active", Description: &description}
	_, err := service.UpdateOne(ownerID, updated)
	assertCode(t, err, "")
	if *updated.Description != "Sobre a iniciativa" {
		t.Fatalf("description = %q", *updated.Description)
	}

	long := strings.Repeat("é", maxDescriptionLength+1)
	_, err = service.UpdateOne(ownerID, &models.Initiative{ID: "active", Description: &long})
	assertCode(t, err, apperrors.CodeInvalidRequest)

	limit := " " + strings.Repeat("é", maxDescriptionLength) + " "
	_, err = service.UpdateOne(ownerID, &models.Initiative{ID: "active", Description: &limit})
	assertCode(t, err, "")
}
