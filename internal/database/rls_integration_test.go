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

func TestPhase3RLSIsolatesPeopleData(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is required for the Phase 3 RLS integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := MigrateFoundation(db); err != nil {
		t.Fatal(err)
	}
	if err := MigratePhase2(db); err != nil {
		t.Fatal(err)
	}
	if err := MigratePhase3(db); err != nil {
		t.Fatal(err)
	}
	if err := SeedFoundation(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := SeedPhase2(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := SeedPhase3(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyFoundationRLS(db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPhase2RLS(db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPhase3RLS(db); err != nil {
		t.Fatal(err)
	}

	suffix := time.Now().UTC().UnixNano()
	schoolA := model.School{Name: fmt.Sprintf("People RLS A %d", suffix), Status: model.SchoolStatusActive}
	schoolB := model.School{Name: fmt.Sprintf("People RLS B %d", suffix), Status: model.SchoolStatusActive}
	if err := db.Create(&schoolA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&schoolB).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&model.School{}, []uint64{schoolA.ID, schoolB.ID})

	addressA := model.Address{SchoolID: schoolA.ID, District: "Hodan A"}
	addressB := model.Address{SchoolID: schoolB.ID, District: "Hodan B"}
	if err := db.Create(&addressA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&addressB).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&model.Address{}, []uint64{addressA.ID, addressB.ID})
	guardianA := model.Responsible{SchoolID: schoolA.ID, Name: "Guardian A", Phone: "+252610000001", Relationship: "Father"}
	guardianB := model.Responsible{SchoolID: schoolB.ID, Name: "Guardian B", Phone: "+252610000002", Relationship: "Mother"}
	if err := db.Create(&guardianA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&guardianB).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&model.Responsible{}, []uint64{guardianA.ID, guardianB.ID})
	today := time.Now().UTC().Truncate(24 * time.Hour)
	studentA := model.Student{SchoolID: schoolA.ID, Name: "Student A", MotherName: "Mother A", Sex: model.SexFemale, AddressID: &addressA.ID, ResponsibleID: guardianA.ID, RegisteredOn: today, Status: model.StudentStatusActive}
	studentB := model.Student{SchoolID: schoolB.ID, Name: "Student B", MotherName: "Mother B", Sex: model.SexMale, AddressID: &addressB.ID, ResponsibleID: guardianB.ID, RegisteredOn: today, Status: model.StudentStatusActive}
	if err := db.Create(&studentA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&studentB).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&model.Student{}, []uint64{studentA.ID, studentB.ID})

	var job model.Job
	var decree model.Decree
	var activeStatus model.StaffStatusType
	if err := db.Where("LOWER(job_name) = 'teacher'").First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("LOWER(dec_name) = 'permanent'").First(&decree).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("LOWER(sst_name) = 'active'").First(&activeStatus).Error; err != nil {
		t.Fatal(err)
	}
	staffA := model.Staff{SchoolID: schoolA.ID, Name: "Staff A", Sex: model.SexMale, AddressID: &addressA.ID, JobID: job.ID, DecreeID: decree.ID, Salary: 500, HiredDate: today}
	staffB := model.Staff{SchoolID: schoolB.ID, Name: "Staff B", Sex: model.SexFemale, AddressID: &addressB.ID, JobID: job.ID, DecreeID: decree.ID, Salary: 500, HiredDate: today}
	if err := db.Create(&staffA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&staffB).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&model.Staff{}, []uint64{staffA.ID, staffB.ID})
	statusA := model.StaffStatus{SchoolID: schoolA.ID, StaffID: staffA.ID, StaffStatusTypeID: activeStatus.ID, StatusDate: today}
	statusB := model.StaffStatus{SchoolID: schoolB.ID, StaffID: staffB.ID, StaffStatusTypeID: activeStatus.ID, StatusDate: today}
	if err := db.Create(&statusA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&statusB).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&model.StaffStatus{}, []uint64{statusA.ID, statusB.ID})

	principalA := authz.Principal{UserID: 910001, SchoolID: &schoolA.ID, Role: model.RoleSchoolAdmin}
	ctxA, txA, err := BeginRequest(context.Background(), db, SecurityScope{Principal: principalA})
	if err != nil {
		t.Fatal(err)
	}
	assertOnlyTenantRow := func(table, idColumn string, ids []uint64, expected uint64) {
		t.Helper()
		var visible []uint64
		if err := FromContext(ctxA, db).Table(table).Where(idColumn+" IN ?", ids).Pluck(idColumn, &visible).Error; err != nil {
			t.Fatal(err)
		}
		if len(visible) != 1 || visible[0] != expected {
			t.Fatalf("%s visible IDs = %v, want [%d]", table, visible, expected)
		}
	}
	assertOnlyTenantRow("addresses", "add_no", []uint64{addressA.ID, addressB.ID}, addressA.ID)
	assertOnlyTenantRow("responsibles", "res_no", []uint64{guardianA.ID, guardianB.ID}, guardianA.ID)
	assertOnlyTenantRow("students", "std_id", []uint64{studentA.ID, studentB.ID}, studentA.ID)
	assertOnlyTenantRow("staff", "stf_no", []uint64{staffA.ID, staffB.ID}, staffA.ID)
	assertOnlyTenantRow("staff_status", "ss_no", []uint64{statusA.ID, statusB.ID}, statusA.ID)
	forbiddenAddress := model.Address{SchoolID: schoolB.ID, District: "Forbidden"}
	if err := FromContext(ctxA, db).Create(&forbiddenAddress).Error; err == nil {
		t.Fatal("school A inserted a school B address")
	}
	txA.Rollback()

	super := authz.Principal{UserID: 910000, Role: model.RoleSuperAdmin}
	ctxSuper, txSuper, err := BeginRequest(context.Background(), db, SecurityScope{Principal: super})
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := FromContext(ctxSuper, db).Model(&model.Student{}).Where("std_id IN ?", []uint64{studentA.ID, studentB.ID}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("SuperAdmin saw %d Phase 3 students; want 2", count)
	}
	txSuper.Rollback()
}

func TestAcademicRLSAndIntegrity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is required for the academic RLS integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := MigrateFoundation(db); err != nil {
		t.Fatal(err)
	}
	if err := MigratePhase2(db); err != nil {
		t.Fatal(err)
	}
	if err := MigratePhase3(db); err != nil {
		t.Fatal(err)
	}
	if err := MigrateAcademic(db); err != nil {
		t.Fatal(err)
	}
	if err := SeedFoundation(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := SeedPhase2(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := SeedPhase3(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := SeedAcademic(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyFoundationRLS(db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPhase2RLS(db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPhase3RLS(db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyAcademicRLS(db); err != nil {
		t.Fatal(err)
	}

	suffix := time.Now().UTC().UnixNano()
	schoolA := model.School{Name: fmt.Sprintf("Academic RLS A %d", suffix), Status: model.SchoolStatusActive}
	schoolB := model.School{Name: fmt.Sprintf("Academic RLS B %d", suffix), Status: model.SchoolStatusActive}
	if err := db.Create(&schoolA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&schoolB).Error; err != nil {
		t.Fatal(err)
	}
	defer func() {
		schoolIDs := []uint64{schoolA.ID, schoolB.ID}
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.ExamResult{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.ExamRegistration{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.SubjectClass{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.StudentClass{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.User{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.StaffStatus{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.Staff{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.Student{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.Responsible{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.Address{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.Class{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.Level{})
		db.Where("sch_no IN ?", schoolIDs).Delete(&model.AcademicYear{})
		db.Delete(&model.School{}, schoolIDs)
	}()

	yearStart := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)
	yearA := model.AcademicYear{SchoolID: schoolA.ID, YearName: fmt.Sprintf("A-%d", suffix%1_000_000_000), StartsOn: yearStart, EndsOn: yearEnd}
	yearB := model.AcademicYear{SchoolID: schoolB.ID, YearName: fmt.Sprintf("B-%d", suffix%1_000_000_000), StartsOn: yearStart, EndsOn: yearEnd}
	if err := db.Create(&yearA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&yearB).Error; err != nil {
		t.Fatal(err)
	}
	levelA := model.Level{SchoolID: schoolA.ID, Name: fmt.Sprintf("Academic A %d", suffix), Status: model.RecordStatusActive}
	levelB := model.Level{SchoolID: schoolB.ID, Name: fmt.Sprintf("Academic B %d", suffix), Status: model.RecordStatusActive}
	if err := db.Create(&levelA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&levelB).Error; err != nil {
		t.Fatal(err)
	}
	classA := model.Class{SchoolID: schoolA.ID, LevelID: levelA.ID, Name: fmt.Sprintf("Class A %d", suffix), Status: model.RecordStatusActive}
	classB := model.Class{SchoolID: schoolB.ID, LevelID: levelB.ID, Name: fmt.Sprintf("Class B %d", suffix), Status: model.RecordStatusActive}
	if err := db.Create(&classA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&classB).Error; err != nil {
		t.Fatal(err)
	}
	guardianA := model.Responsible{SchoolID: schoolA.ID, Name: "Academic Guardian A", Phone: fmt.Sprintf("A%d", suffix), Relationship: "Parent"}
	guardianB := model.Responsible{SchoolID: schoolB.ID, Name: "Academic Guardian B", Phone: fmt.Sprintf("B%d", suffix), Relationship: "Parent"}
	if err := db.Create(&guardianA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&guardianB).Error; err != nil {
		t.Fatal(err)
	}
	studentA := model.Student{SchoolID: schoolA.ID, Name: "Academic Student A", MotherName: "Mother A", Sex: model.SexFemale, ResponsibleID: guardianA.ID, RegisteredOn: yearStart, Status: model.StudentStatusActive}
	studentB := model.Student{SchoolID: schoolB.ID, Name: "Academic Student B", MotherName: "Mother B", Sex: model.SexMale, ResponsibleID: guardianB.ID, RegisteredOn: yearStart, Status: model.StudentStatusActive}
	if err := db.Create(&studentA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&studentB).Error; err != nil {
		t.Fatal(err)
	}
	var job model.Job
	var decree model.Decree
	var subject model.Subject
	var exam model.Exam
	if err := db.Where("LOWER(job_name) = 'teacher'").First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("LOWER(dec_name) = 'permanent'").First(&decree).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("status = ?", model.RecordStatusActive).First(&subject).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("status = ?", model.RecordStatusActive).First(&exam).Error; err != nil {
		t.Fatal(err)
	}
	staffA := model.Staff{SchoolID: schoolA.ID, Name: "Academic Teacher A", Sex: model.SexMale, JobID: job.ID, DecreeID: decree.ID, Salary: 1, HiredDate: yearStart}
	staffB := model.Staff{SchoolID: schoolB.ID, Name: "Academic Teacher B", Sex: model.SexFemale, JobID: job.ID, DecreeID: decree.ID, Salary: 1, HiredDate: yearStart}
	if err := db.Create(&staffA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&staffB).Error; err != nil {
		t.Fatal(err)
	}
	var schoolAdminRole model.Role
	if err := db.Where("role_name = ?", model.RoleSchoolAdmin).First(&schoolAdminRole).Error; err != nil {
		t.Fatal(err)
	}
	userA := model.User{SchoolID: &schoolA.ID, Username: fmt.Sprintf("academic-a-%d", suffix), PasswordHash: "integration", RoleID: schoolAdminRole.ID, Status: model.UserStatusActive}
	userB := model.User{SchoolID: &schoolB.ID, Username: fmt.Sprintf("academic-b-%d", suffix), PasswordHash: "integration", RoleID: schoolAdminRole.ID, Status: model.UserStatusActive}
	if err := db.Create(&userA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&userB).Error; err != nil {
		t.Fatal(err)
	}
	enrollmentA := model.StudentClass{SchoolID: schoolA.ID, StudentID: studentA.ID, ClassID: classA.ID, AcademicYearID: yearA.ID}
	enrollmentB := model.StudentClass{SchoolID: schoolB.ID, StudentID: studentB.ID, ClassID: classB.ID, AcademicYearID: yearB.ID}
	if err := db.Create(&enrollmentA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&enrollmentB).Error; err != nil {
		t.Fatal(err)
	}
	assignmentA := model.SubjectClass{SchoolID: schoolA.ID, SubjectID: subject.ID, ClassID: classA.ID, StaffID: staffA.ID, AcademicYearID: yearA.ID, MaxMark: 100}
	assignmentB := model.SubjectClass{SchoolID: schoolB.ID, SubjectID: subject.ID, ClassID: classB.ID, StaffID: staffB.ID, AcademicYearID: yearB.ID, MaxMark: 100}
	if err := db.Create(&assignmentA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&assignmentB).Error; err != nil {
		t.Fatal(err)
	}
	examA := model.ExamRegistration{SchoolID: schoolA.ID, ExamID: exam.ID, AcademicYearID: yearA.ID, StartsOn: yearStart.AddDate(0, 1, 0), EndsOn: yearStart.AddDate(0, 1, 5)}
	examB := model.ExamRegistration{SchoolID: schoolB.ID, ExamID: exam.ID, AcademicYearID: yearB.ID, StartsOn: yearStart.AddDate(0, 1, 0), EndsOn: yearStart.AddDate(0, 1, 5)}
	if err := db.Create(&examA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&examB).Error; err != nil {
		t.Fatal(err)
	}
	resultA := model.ExamResult{SchoolID: schoolA.ID, ExamRegistrationID: examA.ID, StudentClassID: enrollmentA.ID, SubjectClassID: assignmentA.ID, Marks: 80, RecordedBy: userA.ID}
	resultB := model.ExamResult{SchoolID: schoolB.ID, ExamRegistrationID: examB.ID, StudentClassID: enrollmentB.ID, SubjectClassID: assignmentB.ID, Marks: 90, RecordedBy: userB.ID}
	if err := db.Create(&resultA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&resultB).Error; err != nil {
		t.Fatal(err)
	}

	principalA := authz.Principal{UserID: userA.ID, SchoolID: &schoolA.ID, Role: model.RoleSchoolAdmin}
	ctxA, txA, err := BeginRequest(context.Background(), db, SecurityScope{Principal: principalA})
	if err != nil {
		t.Fatal(err)
	}
	assertOnlyAcademicTenant := func(table, idColumn string, ids []uint64, expected uint64) {
		t.Helper()
		var visible []uint64
		if err := FromContext(ctxA, db).Table(table).Where(idColumn+" IN ?", ids).Pluck(idColumn, &visible).Error; err != nil {
			t.Fatal(err)
		}
		if len(visible) != 1 || visible[0] != expected {
			t.Fatalf("%s visible IDs = %v, want [%d]", table, visible, expected)
		}
	}
	assertOnlyAcademicTenant("student_classes", "sc_no", []uint64{enrollmentA.ID, enrollmentB.ID}, enrollmentA.ID)
	assertOnlyAcademicTenant("subject_classes", "sb_cl_no", []uint64{assignmentA.ID, assignmentB.ID}, assignmentA.ID)
	assertOnlyAcademicTenant("exam_registrations", "ex_r_no", []uint64{examA.ID, examB.ID}, examA.ID)
	assertOnlyAcademicTenant("exam_results", "res_no", []uint64{resultA.ID, resultB.ID}, resultA.ID)
	forbidden := model.StudentClass{SchoolID: schoolB.ID, StudentID: studentB.ID, ClassID: classB.ID, AcademicYearID: yearB.ID}
	if err := FromContext(ctxA, db).Create(&forbidden).Error; err == nil {
		t.Fatal("school A inserted a school B enrollment")
	}
	txA.Rollback()

	tooHigh := model.ExamResult{SchoolID: schoolA.ID, ExamRegistrationID: examA.ID, StudentClassID: enrollmentA.ID, SubjectClassID: assignmentA.ID, Marks: 101, RecordedBy: userA.ID}
	if err := db.Create(&tooHigh).Error; err == nil {
		t.Fatal("database accepted marks above subject_classes.max_mark")
	}

	super := authz.Principal{UserID: 910000, Role: model.RoleSuperAdmin}
	ctxSuper, txSuper, err := BeginRequest(context.Background(), db, SecurityScope{Principal: super})
	if err != nil {
		t.Fatal(err)
	}
	var visibleResults int64
	if err := FromContext(ctxSuper, db).Model(&model.ExamResult{}).Where("res_no IN ?", []uint64{resultA.ID, resultB.ID}).Count(&visibleResults).Error; err != nil {
		t.Fatal(err)
	}
	if visibleResults != 2 {
		t.Fatalf("SuperAdmin saw %d academic results; want 2", visibleResults)
	}
	txSuper.Rollback()
}
