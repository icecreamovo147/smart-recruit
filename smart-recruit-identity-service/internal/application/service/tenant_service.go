package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"smart-recruit-identity-service/internal/domain/model"
	"smart-recruit-identity-service/internal/domain/repository"
	platformmetadata "smart-recruit-platform-go/metadata"
)

var (
	ErrTenantInvalid    = errors.New("invalid tenant")
	tenantSlugPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)
	tenantLocalePattern = regexp.MustCompile(`^[a-z]{2,3}(?:-[A-Z]{2})?$`)
)

type TenantService struct {
	repo  repository.TenantRepository
	authz repository.AuthzRepository
}

func NewTenantService(repo repository.TenantRepository, authz repository.AuthzRepository) *TenantService {
	return &TenantService{repo: repo, authz: authz}
}

func (s *TenantService) Create(ctx context.Context, slug, name, timezone, locale string) (*model.Tenant, error) {
	if err := s.authorizePlatform(ctx); err != nil {
		return nil, err
	}
	slug = strings.ToLower(strings.TrimSpace(slug))
	name = strings.TrimSpace(name)
	if s.repo == nil || !tenantSlugPattern.MatchString(slug) || name == "" || len(name) > 128 {
		return nil, ErrTenantInvalid
	}
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	if locale == "" {
		locale = "zh-CN"
	}
	if _, err := time.LoadLocation(timezone); err != nil || !tenantLocalePattern.MatchString(locale) {
		return nil, ErrTenantInvalid
	}
	tenant := &model.Tenant{TenantKey: uuid.NewString(), Slug: slug, Name: name, Status: "active", Timezone: timezone, Locale: locale}
	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *TenantService) List(ctx context.Context, page, pageSize int32, keyword, status string) ([]model.Tenant, int64, error) {
	if err := s.authorizePlatform(ctx); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	status = strings.TrimSpace(status)
	if status != "" && status != "active" && status != "suspended" && status != "disabled" {
		return nil, 0, ErrTenantInvalid
	}
	return s.repo.List(ctx, int((page-1)*pageSize), int(pageSize), strings.TrimSpace(keyword), status)
}

func (s *TenantService) UpdateStatus(ctx context.Context, tenantID int64, status string) (*model.Tenant, error) {
	if err := s.authorizePlatform(ctx); err != nil {
		return nil, err
	}
	if tenantID <= 0 || (status != "active" && status != "suspended" && status != "disabled") {
		return nil, ErrTenantInvalid
	}
	tenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil || tenant.IsDefault {
		return nil, ErrTenantInvalid
	}
	if tenant.Status == status {
		return tenant, nil
	}
	return s.repo.UpdateStatus(ctx, tenantID, status)
}

func (s *TenantService) ListMemberships(ctx context.Context, tenantID int64, page, pageSize int32) ([]model.TenantMembership, int64, error) {
	if err := s.authorizePlatform(ctx); err != nil {
		return nil, 0, err
	}
	if tenantID <= 0 {
		return nil, 0, ErrTenantInvalid
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListTenantMemberships(ctx, tenantID, int((page-1)*pageSize), int(pageSize))
}

func (s *TenantService) authorizePlatform(ctx context.Context) error {
	if s.authz == nil || platformmetadata.GetAuthClientApp(ctx) != "platform" || platformmetadata.GetAuthTenantID(ctx) != 0 {
		return ErrInvalidCredentials
	}
	principal, err := s.authz.LoadPlatformPrincipal(ctx, uint64(platformmetadata.GetAuthUserID(ctx)))
	if err != nil || principal == nil {
		return ErrInvalidCredentials
	}
	for _, role := range principal.Roles {
		if role == model.RolePlatformAdmin {
			return nil
		}
	}
	return ErrInvalidCredentials
}
