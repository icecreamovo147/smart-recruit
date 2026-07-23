package persistence

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	platformmetadata "smart-recruit-platform-go/metadata"
	"smart-recruit-platform-go/tenantgorm"
)

func TestTenantGormStrictAndMixedIsolation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:tenant-boundary?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&strictTenantFixture{}, &mixedTenantFixture{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]strictTenantFixture{{TenantID: 1, Name: "tenant-1"}, {TenantID: 2, Name: "tenant-2"}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]mixedTenantFixture{{TenantID: nil, Name: "global"}, {TenantID: int64Ptr(1), Name: "tenant-1"}, {TenantID: int64Ptr(2), Name: "tenant-2"}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Use(tenantgorm.NewWithMixed([]string{"tenant_boundary_strict"}, []string{"tenant_boundary_mixed"})); err != nil {
		t.Fatal(err)
	}

	tenantOne := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{TenantID: 1, MembershipID: 11, UserID: 7, AccountType: "staff", ClientApp: "hr"})
	var strict []strictTenantFixture
	if err := db.WithContext(tenantOne).Find(&strict).Error; err != nil {
		t.Fatal(err)
	}
	if len(strict) != 1 || strict[0].TenantID != 1 {
		t.Fatalf("strict rows = %+v, want tenant 1 only", strict)
	}

	var mixed []mixedTenantFixture
	if err := db.WithContext(tenantOne).Order("name").Find(&mixed).Error; err != nil {
		t.Fatal(err)
	}
	if len(mixed) != 2 || mixed[0].Name != "global" || mixed[1].Name != "tenant-1" {
		t.Fatalf("mixed rows = %+v, want global plus tenant 1", mixed)
	}

	created := strictTenantFixture{TenantID: 2, Name: "forced-to-current"}
	if err := db.WithContext(tenantOne).Create(&created).Error; err != nil {
		t.Fatal(err)
	}
	if created.TenantID != 1 {
		t.Fatalf("created tenant = %d, want 1", created.TenantID)
	}
}

func TestTenantGormFailsClosedForStaffWithoutTenant(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:tenant-boundary-missing?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&strictTenantFixture{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Use(tenantgorm.New("tenant_boundary_strict")); err != nil {
		t.Fatal(err)
	}
	ctx := platformmetadata.WithAuthActor(context.Background(), 7, "staff")
	var rows []strictTenantFixture
	err = db.WithContext(ctx).Find(&rows).Error
	if !errors.Is(err, tenantgorm.ErrMissingTenantContext) {
		t.Fatalf("query error = %v, want ErrMissingTenantContext", err)
	}
}

type strictTenantFixture struct {
	ID       int64 `gorm:"primaryKey"`
	TenantID int64
	Name     string
}

func (strictTenantFixture) TableName() string { return "tenant_boundary_strict" }

type mixedTenantFixture struct {
	ID       int64 `gorm:"primaryKey"`
	TenantID *int64
	Name     string
}

func (mixedTenantFixture) TableName() string { return "tenant_boundary_mixed" }
func int64Ptr(value int64) *int64            { return &value }
