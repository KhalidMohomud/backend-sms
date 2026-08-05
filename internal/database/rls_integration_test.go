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
	if err := MigratePhase2(db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPhase2RLS(db); err != nil {
		t.Fatal(err)
	}
	if err := ConfigureFoundationModels(db); err != nil {
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
	var visibleSchools []model.School
	if err := FromContext(ctxA, db).Where("sch_no IN ?", []uint64{schoolA.ID, schoolB.ID}).Find(&visibleSchools).Error; err != nil {
		t.Fatal(err)
	}
	if len(visibleSchools) != 1 || visibleSchools[0].ID != schoolA.ID {
		t.Fatalf("school A saw schools %#v; expected only school %d", visibleSchools, schoolA.ID)
	}
	txA.Rollback()

	levelA := model.Level{SchoolID: schoolA.ID, Name: fmt.Sprintf("RLS Level A %d", suffix), Price: 10, Status: model.RecordStatusActive}
	levelB := model.Level{SchoolID: schoolB.ID, Name: fmt.Sprintf("RLS Level B %d", suffix), Price: 10, Status: model.RecordStatusActive}
	if err := db.Create(&levelA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&levelB).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&model.Level{}, []uint64{levelA.ID, levelB.ID})
	classA := model.Class{SchoolID: schoolA.ID, LevelID: levelA.ID, Name: fmt.Sprintf("RLS Class A %d", suffix), Status: model.RecordStatusActive}
	classB := model.Class{SchoolID: schoolB.ID, LevelID: levelB.ID, Name: fmt.Sprintf("RLS Class B %d", suffix), Status: model.RecordStatusActive}
	if err := db.Create(&classA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&classB).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&model.Class{}, []uint64{classA.ID, classB.ID})
	mismatchedClass := model.Class{
		SchoolID: schoolA.ID, LevelID: levelB.ID,
		Name: fmt.Sprintf("RLS mismatched class %d", suffix), Status: model.RecordStatusActive,
	}
	if err := db.Create(&mismatchedClass).Error; err == nil {
		t.Fatal("database accepted a class whose level belongs to another school")
	}

	ctxLevel, txLevel, err := BeginRequest(context.Background(), db, SecurityScope{Principal: principalA})
	if err != nil {
		t.Fatal(err)
	}
	var visibleLevels []model.Level
	if err := FromContext(ctxLevel, db).Where("lev_no IN ?", []uint64{levelA.ID, levelB.ID}).Find(&visibleLevels).Error; err != nil {
		t.Fatal(err)
	}
	if len(visibleLevels) != 1 || visibleLevels[0].ID != levelA.ID {
		t.Fatalf("school A saw levels %#v; expected only level %d", visibleLevels, levelA.ID)
	}
	var visibleClasses []model.Class
	if err := FromContext(ctxLevel, db).Where("cl_no IN ?", []uint64{classA.ID, classB.ID}).Find(&visibleClasses).Error; err != nil {
		t.Fatal(err)
	}
	if len(visibleClasses) != 1 || visibleClasses[0].ID != classA.ID {
		t.Fatalf("school A saw classes %#v; expected only class %d", visibleClasses, classA.ID)
	}
	txLevel.Rollback()

	ctxLevelWrite, txLevelWrite, err := BeginRequest(context.Background(), db, SecurityScope{Principal: principalA})
	if err != nil {
		t.Fatal(err)
	}
	forbiddenLevel := model.Level{SchoolID: schoolB.ID, Name: fmt.Sprintf("RLS forbidden %d", suffix), Status: model.RecordStatusActive}
	if err := FromContext(ctxLevelWrite, db).Create(&forbiddenLevel).Error; err == nil {
		t.Fatal("school A inserted a level for school B")
	}
	txLevelWrite.Rollback()

	ctxLookupWrite, txLookupWrite, err := BeginRequest(context.Background(), db, SecurityScope{Principal: principalA})
	if err != nil {
		t.Fatal(err)
	}
	if err := FromContext(ctxLookupWrite, db).Table("subjects").Create(map[string]any{
		"sub_name": fmt.Sprintf("RLS forbidden %d", suffix),
		"status":   "active",
	}).Error; err == nil {
		t.Fatal("SchoolAdmin mutated a SuperAdmin-only global lookup")
	}
	txLookupWrite.Rollback()

	var registrar model.Role
	if err := db.Where("role_name = ?", model.RoleRegistrar).First(&registrar).Error; err != nil {
		t.Fatal(err)
	}
	loginUser := model.User{SchoolID: &schoolA.ID, Username: fmt.Sprintf("rls-login-%d", suffix), PasswordHash: "test-only", RoleID: registrar.ID, Status: model.UserStatusActive}
	otherUser := model.User{SchoolID: &schoolB.ID, Username: fmt.Sprintf("rls-other-%d", suffix), PasswordHash: "test-only", RoleID: registrar.ID, Status: model.UserStatusActive}
	if err := db.Create(&loginUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&model.User{}, []uint64{loginUser.ID, otherUser.ID})
	auditA := model.AuditLog{UserID: &loginUser.ID, SchoolID: &schoolA.ID, Action: "RLS_TEST", ResourceType: "users", RecordID: &loginUser.ID}
	auditB := model.AuditLog{UserID: &otherUser.ID, SchoolID: &schoolB.ID, Action: "RLS_TEST", ResourceType: "users", RecordID: &otherUser.ID}
	if err := db.Create(&auditA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&auditB).Error; err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Exec("ALTER TABLE audit_logs DISABLE TRIGGER audit_logs_no_update_or_delete").Error; err != nil {
			t.Errorf("disable audit cleanup trigger: %v", err)
			return
		}
		if err := db.Delete(&model.AuditLog{}, []uint64{auditA.ID, auditB.ID}).Error; err != nil {
			t.Errorf("delete RLS test audits: %v", err)
		}
		if err := db.Exec("ALTER TABLE audit_logs ENABLE TRIGGER audit_logs_no_update_or_delete").Error; err != nil {
			t.Errorf("enable audit cleanup trigger: %v", err)
		}
	}()

	ctxTenantData, txTenantData, err := BeginRequest(context.Background(), db, SecurityScope{Principal: principalA})
	if err != nil {
		t.Fatal(err)
	}
	var visibleUsers []model.User
	if err := FromContext(ctxTenantData, db).Where("usr_no IN ?", []uint64{loginUser.ID, otherUser.ID}).Find(&visibleUsers).Error; err != nil {
		t.Fatal(err)
	}
	if len(visibleUsers) != 1 || visibleUsers[0].ID != loginUser.ID {
		t.Fatalf("school A saw users %#v; expected only user %d", visibleUsers, loginUser.ID)
	}
	var visibleAudits []model.AuditLog
	if err := FromContext(ctxTenantData, db).Where("log_no IN ?", []uint64{auditA.ID, auditB.ID}).Find(&visibleAudits).Error; err != nil {
		t.Fatal(err)
	}
	if len(visibleAudits) != 1 || visibleAudits[0].ID != auditA.ID {
		t.Fatalf("school A saw audit logs %#v; expected only audit %d", visibleAudits, auditA.ID)
	}
	result := FromContext(ctxTenantData, db).Model(&model.User{}).Where("usr_no = ?", otherUser.ID).Update("status", model.UserStatusDisabled)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	if result.RowsAffected != 0 {
		t.Fatal("school A updated school B user")
	}
	txTenantData.Rollback()

	ctxLogin, txLogin, err := BeginRequest(context.Background(), db, SecurityScope{AuthLookup: true})
	if err != nil {
		t.Fatal(err)
	}
	var loaded model.User
	if err := FromContext(ctxLogin, db).Preload("School").Preload("Role.Permissions").First(&loaded, "usr_no = ?", loginUser.ID).Error; err != nil {
		t.Fatal(err)
	}
	if loaded.School == nil || loaded.School.ID != schoolA.ID {
		t.Fatalf("authentication lookup did not load school context: %#v", loaded.School)
	}
	var authenticationAudits []model.AuditLog
	if err := FromContext(ctxLogin, db).Where("log_no IN ?", []uint64{auditA.ID, auditB.ID}).Find(&authenticationAudits).Error; err != nil {
		t.Fatal(err)
	}
	if len(authenticationAudits) != 0 {
		t.Fatalf("authentication context read audit logs: %#v", authenticationAudits)
	}
	unauthorizedUpdate := FromContext(ctxLogin, db).Model(&model.User{}).Where("usr_no = ?", otherUser.ID).Update("status", model.UserStatusDisabled)
	if unauthorizedUpdate.Error != nil {
		t.Fatal(unauthorizedUpdate.Error)
	}
	if unauthorizedUpdate.RowsAffected != 0 {
		t.Fatal("anonymous authentication context updated a user")
	}
	if err := FromContext(ctxLogin, db).Exec("SELECT set_config('app.current_user', ?, true)", fmt.Sprint(loginUser.ID)).Error; err != nil {
		t.Fatal(err)
	}
	if err := FromContext(ctxLogin, db).Model(&model.User{}).Where("usr_no = ?", loginUser.ID).Update("username", "auth-escalation-attempt").Error; err == nil {
		t.Fatal("authentication context changed user identity")
	}
	txLogin.Rollback()

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

func TestRequestTransactionRollsBackBusinessChangeWhenAuditFails(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is required for the PostgreSQL transaction integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	super := authz.Principal{UserID: 900000, Role: model.RoleSuperAdmin}
	ctx, tx, err := BeginRequest(context.Background(), db, SecurityScope{Principal: super})
	if err != nil {
		t.Fatal(err)
	}
	school := model.School{Name: fmt.Sprintf("Atomic rollback %d", time.Now().UnixNano()), Status: model.SchoolStatusActive}
	if err := FromContext(ctx, db).Create(&school).Error; err != nil {
		t.Fatal(err)
	}
	missingSchool := uint64(9223372036854775807)
	audit := model.AuditLog{UserID: nil, SchoolID: &missingSchool, Action: "INSERT", ResourceType: "schools", RecordID: &school.ID}
	if err := FromContext(ctx, db).Create(&audit).Error; err == nil {
		t.Fatal("invalid audit row unexpectedly succeeded")
	}
	tx.Rollback()

	var count int64
	if err := db.Model(&model.School{}).Where("sch_no = ?", school.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("school %d remained after audit failure rollback", school.ID)
	}
}
