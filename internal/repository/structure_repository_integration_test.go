//go:build integration

package repository

import (
	"backendapi/internal/authz"
	"backendapi/internal/database"
	"backendapi/internal/model"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestLookupRepositoryUsesAllowlistedTableAndRLS(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.MigratePhase2(db); err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyFoundationRLS(db); err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyPhase2RLS(db); err != nil {
		t.Fatal(err)
	}

	repo := NewStructureRepository(db)
	super := authz.Principal{UserID: 900000, Role: model.RoleSuperAdmin}
	ctxSuper, txSuper, err := database.BeginRequest(context.Background(), db, database.SecurityScope{Principal: super})
	if err != nil {
		t.Fatal(err)
	}
	item := &model.LookupItem{Name: fmt.Sprintf("Integration Subject %d", time.Now().UnixNano()), Status: model.RecordStatusActive}
	if err := repo.CreateLookup(ctxSuper, LookupSubjects, item); err != nil {
		t.Fatal(err)
	}
	if item.ID == 0 || item.Name == "" {
		t.Fatalf("created lookup was not returned: %#v", item)
	}
	loaded, err := repo.FindLookupByID(ctxSuper, LookupSubjects, item.ID)
	if err != nil || loaded.ID != item.ID {
		t.Fatalf("FindLookupByID() = %#v, %v", loaded, err)
	}
	duplicate := &model.LookupItem{Name: strings.ToLower(item.Name), Status: model.RecordStatusActive}
	if err := repo.CreateLookup(ctxSuper, LookupSubjects, duplicate); err == nil {
		t.Fatal("case-insensitive duplicate lookup name was accepted")
	}
	txSuper.Rollback()

	schoolID := uint64(1)
	schoolAdmin := authz.Principal{UserID: 900001, SchoolID: &schoolID, Role: model.RoleSchoolAdmin}
	ctxSchool, txSchool, err := database.BeginRequest(context.Background(), db, database.SecurityScope{Principal: schoolAdmin})
	if err != nil {
		t.Fatal(err)
	}
	forbidden := &model.LookupItem{Name: fmt.Sprintf("Forbidden Subject %d", time.Now().UnixNano()), Status: model.RecordStatusActive}
	if err := repo.CreateLookup(ctxSchool, LookupSubjects, forbidden); err == nil {
		t.Fatal("SchoolAdmin created a global lookup through the repository")
	}
	txSchool.Rollback()
}
