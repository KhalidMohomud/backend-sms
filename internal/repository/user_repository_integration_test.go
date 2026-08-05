//go:build integration

package repository

import (
	"backendapi/internal/authz"
	"backendapi/internal/database"
	"backendapi/internal/model"
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAuthenticationLookupNarrowsUpdatesToLoadedUser(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.ConfigureFoundationModels(db); err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyFoundationRLS(db); err != nil {
		t.Fatal(err)
	}
	var role model.Role
	if err := db.Where("role_name = ?", model.RoleSuperAdmin).First(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: fmt.Sprintf("auth-scope-%d", time.Now().UnixNano()), PasswordHash: "test-only", RoleID: role.ID, Status: model.UserStatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&user)
	repo := NewUserRepository(db)
	ctx, tx, err := database.BeginRequest(context.Background(), db, database.SecurityScope{AuthLookup: true, Principal: authz.Principal{}})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	loaded, err := repo.FindByUsername(ctx, user.Username)
	if err != nil || loaded.ID != user.ID {
		t.Fatalf("FindByUsername() = %#v, %v", loaded, err)
	}
	if err := repo.RecordFailedLogin(ctx, user.ID); err != nil {
		t.Fatalf("RecordFailedLogin() error = %v", err)
	}
	loaded.Username = "forbidden-auth-profile-change"
	if err := repo.UpdateProfile(ctx, loaded); err == nil {
		t.Fatal("authentication context changed identity fields")
	}
}

func TestConcurrentFailedLoginsLockAtFive(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.ConfigureFoundationModels(db); err != nil {
		t.Fatal(err)
	}
	var role model.Role
	if err := db.Where("role_name = ?", model.RoleRegistrar).First(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: fmt.Sprintf("concurrent-lock-%d", time.Now().UnixNano()), PasswordHash: "test-only", RoleID: role.ID, Status: model.UserStatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&user)
	repo := NewUserRepository(db)

	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := repo.RecordFailedLogin(context.Background(), user.ID); err != nil {
				t.Errorf("RecordFailedLogin() error = %v", err)
			}
		}()
	}
	wg.Wait()
	var saved model.User
	if err := db.First(&saved, "usr_no = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != model.UserStatusLocked || saved.FailedLogins != 5 {
		t.Fatalf("status = %s, failed_logins = %d; want locked, 5", saved.Status, saved.FailedLogins)
	}
}
