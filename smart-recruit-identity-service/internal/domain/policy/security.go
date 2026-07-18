package policy

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"

	"smart-recruit-identity-service/internal/domain/model"
)

var (
	ErrPasswordTooShort        = errors.New("密码长度至少8个字符")
	ErrPasswordTooLong         = errors.New("密码长度不能超过128个字符")
	ErrPasswordWeak            = errors.New("密码必须包含大小写字母、数字或特殊字符中的至少三类")
	ErrStaffInviteRequired     = errors.New("HR 账号注册需要有效的邀请码")
	ErrInvalidRegistrationRole = errors.New("无效的账号角色")
	ErrUsernameInvalid         = errors.New("用户名不能为空且不超过50字符")
	ErrUsernameBlank           = errors.New("用户名不能为空")
	ErrEmailTooLong            = errors.New("邮箱长度不能超过128个字符")
	ErrEmailInvalid            = errors.New("请输入有效的邮箱地址")
	ErrSelfSystemAdminRevoke   = errors.New("不能移除自己的系统管理员角色，请让其他系统管理员操作")
	ErrLastSystemAdmin         = errors.New("无法移除最后一个系统管理员，至少需要保留一个系统管理员")
)

type RegistrationPlan struct {
	LegacyRole     int32
	AccountType    string
	Status         string
	RoleKey        string
	ScopeKey       string
	RequiresInvite bool
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	if len(password) > 128 {
		return ErrPasswordTooLong
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}
	classes := 0
	for _, ok := range []bool{hasUpper, hasLower, hasDigit, hasSpecial} {
		if ok {
			classes++
		}
	}
	if classes < 3 {
		return ErrPasswordWeak
	}
	return nil
}

func RegistrationPlanFor(role int32, inviteCode string) (RegistrationPlan, error) {
	hasInvite := inviteCode != ""
	switch role {
	case model.LegacyRoleCandidate:
		return RegistrationPlan{
			LegacyRole:  model.LegacyRoleCandidate,
			AccountType: model.AccountTypeCandidate,
			Status:      model.UserStatusActive,
			RoleKey:     model.RoleCandidate,
		}, nil
	case model.LegacyRoleHR:
		if !hasInvite {
			return RegistrationPlan{}, ErrStaffInviteRequired
		}
		return RegistrationPlan{
			LegacyRole:     model.LegacyRoleHR,
			AccountType:    model.AccountTypeStaff,
			Status:         model.UserStatusActive,
			RoleKey:        model.RoleRecruiter,
			ScopeKey:       model.ScopeOwnJobs,
			RequiresInvite: true,
		}, nil
	default:
		return RegistrationPlan{}, ErrInvalidRegistrationRole
	}
}

func NormalizeUsername(username string) (string, error) {
	if username == "" || len(username) > 50 {
		return "", ErrUsernameInvalid
	}
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return "", ErrUsernameBlank
	}
	return trimmed, nil
}

func NormalizeEmail(email string) (string, error) {
	trimmed := strings.TrimSpace(email)
	if len(trimmed) > 128 {
		return "", ErrEmailTooLong
	}
	if trimmed != "" {
		if _, err := mail.ParseAddress(trimmed); err != nil {
			return "", ErrEmailInvalid
		}
	}
	return trimmed, nil
}

func CheckSystemAdminRevocation(targetUserID, adminUserID uint64, roleKey string, adminPrincipal *model.Principal, activeSystemAdmins int64) error {
	if roleKey != model.RoleSystemAdmin {
		return nil
	}
	if targetUserID == adminUserID && adminPrincipal != nil && !adminPrincipal.HasRole(model.RoleRecruitingAdmin) {
		return ErrSelfSystemAdminRevoke
	}
	if activeSystemAdmins <= 1 {
		return ErrLastSystemAdmin
	}
	return nil
}
