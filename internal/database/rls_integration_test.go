//go:build integration

package database

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRLSIsolatesSchoolsAndAllowsSuperAdmin(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is required for the PostgreSQL RLS integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyFoundationRLS(db); err != nil {
		t.Fatal(err)
	}

	suffix := time.Now().UTC().UnixNano()
	schoolA := model.School{Name: fmt.Sprintf("RLS A %d", suffix), Status: model.SchoolStatusActive}
	schoolB := model.School{Name: fmt.Sprintf("RLS B %d", suffix), Status: model.SchoolStatusActive}
	if err := db.Create(&schoolA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&schoolB).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&model.School{}, []uint64{schoolA.ID, schoolB.ID})

	yearA := model.AcademicYear{SchoolID: schoolA.ID, YearName: "RLS-A", StartsOn: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), EndsOn: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)}
	yearB := model.AcademicYear{SchoolID: schoolB.ID, YearName: "RLS-B", StartsOn: yearA.StartsOn, EndsOn: yearA.EndsOn}
	if err := db.Create(&yearA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&yearB).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&model.AcademicYear{}, []uint64{yearA.ID, yearB.ID})

	principalA := authz.Principal{UserID: 900001, SchoolID: &schoolA.ID, Role: model.RoleSchoolAdmin}
	ctxA, txA, err := BeginRequest(context.Background(), db, SecurityScope{Principal: principalA})
	if err != nil {
		t.Fatal(err)
	}
	var visible []model.AcademicYear
	if err := FromContext(ctxA, db).Order("y_no").Find(&visible).Error; err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].ID != yearA.ID {
		t.Fatalf("school A saw %#v; expected only year %d", visible, yearA.ID)
	}
	txA.Rollback()

	ctxWrite, txWrite, err := BeginRequest(context.Background(), db, SecurityScope{Principal: principalA})
	if err != nil {
		t.Fatal(err)
	}
	forbidden := model.AcademicYear{SchoolID: schoolB.ID, YearName: "RLS-FORBIDDEN", StartsOn: yearA.StartsOn, EndsOn: yearA.EndsOn}
	if err := FromContext(ctxWrite, db).Create(&forbidden).Error; err == nil {
		t.Fatal("school A inserted an academic year for school B")
	}
	txWrite.Rollback()

	super := authz.Principal{UserID: 900000, Role: model.RoleSuperAdmin}
	ctxSuper, txSuper, err := BeginRequest(context.Background(), db, SecurityScope{Principal: super})
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := FromContext(ctxSuper, db).Model(&model.AcademicYear{}).Where("y_no IN ?", []uint64{yearA.ID, yearB.ID}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("SuperAdmin saw %d test academic years; expected 2", count)
	}
	txSuper.Rollback()
}
