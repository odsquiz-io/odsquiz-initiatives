package services

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"gorm.io/gorm"
)

func (s *Service) ListMembers(userID, id string) (*models.MembersList, error) {
	one, err := s.readVisible(userID, id)
	if err != nil {
		return nil, err
	}

	owner := models.MemberUser{ID: one.Owner}
	account, err := s.Members.FindUser(one.Owner)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.Internal(err)
	}
	if account != nil {
		owner.Name = account.Name
	}

	isOwner := one.Owner == userID
	members, err := s.Members.ListMembers(id, isOwner)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	if !isOwner {
		for i := range members {
			members[i].User.Email = ""
		}
	}

	return &models.MembersList{Owner: owner, Members: members}, nil
}

func (s *Service) InviteMember(userID, id, email string) (*models.Member, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, apperrors.BadRequest(apperrors.CodeInvalidRequest, nil)
	}

	if _, err := s.authorizeChange(userID, id); err != nil {
		return nil, err
	}

	invitee, err := s.Members.FindUserByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NotFound(apperrors.CodeUserNotFound, err)
		}
		return nil, apperrors.Internal(err)
	}

	if invitee.ID == userID {
		return nil, apperrors.Conflict(apperrors.CodeCannotInviteSelf, nil)
	}

	if invitee.SuspendedAt != nil {
		return nil, apperrors.Conflict(apperrors.CodeTargetSuspended, nil)
	}

	_, err = s.Members.ReadMemberByUser(id, invitee.ID)
	if err == nil {
		return nil, apperrors.Conflict(apperrors.CodeAlreadyMember, nil)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.Internal(err)
	}

	member := &models.InitiativeMember{
		ID:           uuid.NewString(),
		InitiativeID: id,
		UserID:       invitee.ID,
		Status:       models.MemberStatusPending,
		InvitedBy:    userID,
		CreatedAt:    time.Now(),
	}
	if err := s.Members.CreateMember(member); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apperrors.Conflict(apperrors.CodeAlreadyMember, err)
		}
		return nil, apperrors.Internal(err)
	}

	return &models.Member{
		ID:        member.ID,
		Status:    member.Status,
		User:      models.MemberUser{ID: invitee.ID, Name: invitee.Name, Email: invitee.Email},
		CreatedAt: member.CreatedAt,
	}, nil
}

// RemoveMember lets the owner remove a member or cancel an invitation, and a member leave.
func (s *Service) RemoveMember(userID, id, memberID string) error {
	member, err := s.Members.ReadMember(memberID)
	if err != nil {
		return mapNotFound(err, apperrors.CodeMembershipNotFound)
	}

	if member.InitiativeID != id {
		return apperrors.NotFound(apperrors.CodeMembershipNotFound, nil)
	}

	if member.UserID != userID {
		if _, err := s.authorizeChange(userID, id); err != nil {
			return err
		}
	}

	return mapNotFound(s.Members.DeleteMember(memberID), apperrors.CodeMembershipNotFound)
}

func (s *Service) ListInvitations(userID string) ([]models.Invitation, error) {
	invitations, err := s.Members.ListInvitations(userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return invitations, nil
}

func (s *Service) AcceptInvitation(userID, memberID string) (*models.AcceptedInvitation, error) {
	if err := s.ensureActiveUser(userID); err != nil {
		return nil, err
	}

	member, err := s.readInvitation(userID, memberID)
	if err != nil {
		return nil, err
	}

	one, err := s.Repo.ReadOneByID(member.InitiativeID)
	if err != nil {
		return nil, mapNotFound(err, apperrors.CodeInvitationNotFound)
	}

	if one.SuspendedAt != nil {
		return nil, apperrors.NotFound(apperrors.CodeInvitationNotFound, nil)
	}

	if err := s.Members.AcceptMember(memberID, time.Now()); err != nil {
		return nil, mapNotFound(err, apperrors.CodeInvitationNotFound)
	}

	return &models.AcceptedInvitation{
		ID:         memberID,
		Status:     models.MemberStatusActive,
		Initiative: one.Summary(),
	}, nil
}

func (s *Service) DeclineInvitation(userID, memberID string) error {
	if _, err := s.readInvitation(userID, memberID); err != nil {
		return err
	}

	return mapNotFound(s.Members.DeleteMember(memberID), apperrors.CodeInvitationNotFound)
}

func (s *Service) ListMemberships(userID string) ([]models.Membership, error) {
	memberships, err := s.Members.ListMemberships(userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return memberships, nil
}

// readInvitation returns a pending invitation addressed to the user.
func (s *Service) readInvitation(userID, memberID string) (*models.InitiativeMember, error) {
	member, err := s.Members.ReadMember(memberID)
	if err != nil {
		return nil, mapNotFound(err, apperrors.CodeInvitationNotFound)
	}

	if member.UserID != userID || member.Status != models.MemberStatusPending {
		return nil, apperrors.NotFound(apperrors.CodeInvitationNotFound, nil)
	}

	return member, nil
}

func mapNotFound(err error, notFound apperrors.Code) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.NotFound(notFound, err)
	}

	return apperrors.Internal(err)
}
