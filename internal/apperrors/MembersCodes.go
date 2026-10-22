package apperrors

const KindUnavailable Kind = "unavailable"

const (
	CodeInvalidImage       Code = "invalid_image"
	CodeImageTooLarge      Code = "image_too_large"
	CodeLogoNotFound       Code = "logo_not_found"
	CodeStorageUnavailable Code = "storage_unavailable"
	CodeUserNotFound       Code = "user_not_found"
	CodeCannotInviteSelf   Code = "cannot_invite_self"
	CodeAlreadyMember      Code = "already_member"
	CodeTargetSuspended    Code = "target_suspended"
	CodeMembershipNotFound Code = "membership_not_found"
	CodeInvitationNotFound Code = "invitation_not_found"
)

func Unavailable(code Code, err error) *Error {
	return &Error{Kind: KindUnavailable, Code: code, Err: err}
}
