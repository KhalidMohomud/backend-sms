package service

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"context"
	"testing"
)

func TestLookupManagementIsSuperAdminOnlyAndAudited(t *testing.T) {
	structure := newMemoryStructure()
	audits := &memoryAudits{}
	svc := NewStructureService(structure, &memoryFoundation{}, NewAuditWriter(audits))
	super := authz.Principal{UserID: 1, Role: model.RoleSuperAdmin}
	schoolID := uint64(10)
	schoolAdmin := authz.Principal{
		UserID: 2, SchoolID: &schoolID, Role: model.RoleSchoolAdmin,
		Permissions: []string{model.PermissionManageLookups},
	}

	if _, err := svc.CreateLookup(context.Background(), schoolAdmin, repository.LookupSubjects, CreateLookupInput{Name: "Biology"}, RequestMetadata{}); err != ErrForbidden {
		t.Fatalf("SchoolAdmin CreateLookup() error = %v, want ErrForbidden", err)
	}
	item, err := svc.CreateLookup(context.Background(), super, repository.LookupSubjects, CreateLookupInput{Name: "  Biology  "}, RequestMetadata{})
	if err != nil || item.Name != "Biology" {
		t.Fatalf("CreateLookup() = %#v, %v", item, err)
	}
	if err := svc.ArchiveLookup(context.Background(), super, repository.LookupSubjects, item.ID, RequestMetadata{}); err != nil {
		t.Fatalf("ArchiveLookup() error = %v", err)
	}
	if len(audits.entries) != 2 || audits.entries[0].Action != "INSERT" || audits.entries[1].Action != "DELETE" {
		t.Fatalf("lookup audits = %#v", audits.entries)
	}
}

func TestSchoolStructureIsTenantScopedAndProtectsActiveDependencies(t *testing.T) {
	schoolA := uint64(10)
	schoolB := uint64(20)
	foundation := &memoryFoundation{schools: map[uint64]*model.School{
		schoolA: {ID: schoolA, Status: model.SchoolStatusActive},
		schoolB: {ID: schoolB, Status: model.SchoolStatusActive},
	}}
	structure := newMemoryStructure()
	audits := &memoryAudits{}
	svc := NewStructureService(structure, foundation, NewAuditWriter(audits))
	adminA := authz.Principal{
		UserID: 2, SchoolID: &schoolA, Role: model.RoleSchoolAdmin,
		Permissions: []string{model.PermissionManageStructure},
	}

	level, err := svc.CreateLevel(context.Background(), adminA, schoolA, CreateLevelInput{Name: "Form 1", Price: 25}, RequestMetadata{})
	if err != nil {
		t.Fatalf("CreateLevel() error = %v", err)
	}
	if _, err := svc.CreateLevel(context.Background(), adminA, schoolB, CreateLevelInput{Name: "Forbidden"}, RequestMetadata{}); err != ErrForbidden {
		t.Fatalf("cross-school CreateLevel() error = %v, want ErrForbidden", err)
	}
	class, err := svc.CreateClass(context.Background(), adminA, schoolA, CreateClassInput{Name: "Form 1-A", LevelID: level.ID}, RequestMetadata{})
	if err != nil || class.LevelID != level.ID {
		t.Fatalf("CreateClass() = %#v, %v", class, err)
	}
	if err := svc.ArchiveLevel(context.Background(), adminA, schoolA, level.ID, RequestMetadata{}); err != ErrConflict {
		t.Fatalf("ArchiveLevel() error = %v, want ErrConflict", err)
	}
	if err := svc.ArchiveClass(context.Background(), adminA, schoolA, class.ID, RequestMetadata{}); err != nil {
		t.Fatalf("ArchiveClass() error = %v", err)
	}
	if err := svc.ArchiveLevel(context.Background(), adminA, schoolA, level.ID, RequestMetadata{}); err != nil {
		t.Fatalf("ArchiveLevel() after class deactivation error = %v", err)
	}
	if structure.levels[level.ID].Status != model.RecordStatusInactive || structure.classes[class.ID].Status != model.RecordStatusInactive {
		t.Fatalf("safe delete statuses: level=%s class=%s", structure.levels[level.ID].Status, structure.classes[class.ID].Status)
	}
}

type memoryStructure struct {
	lookups map[repository.LookupKind]map[uint64]*model.LookupItem
	levels  map[uint64]*model.Level
	classes map[uint64]*model.Class
	nextID  uint64
}

func newMemoryStructure() *memoryStructure {
	return &memoryStructure{
		lookups: make(map[repository.LookupKind]map[uint64]*model.LookupItem),
		levels:  make(map[uint64]*model.Level),
		classes: make(map[uint64]*model.Class),
		nextID:  1,
	}
}

func (m *memoryStructure) id() uint64 {
	id := m.nextID
	m.nextID++
	return id
}

func (m *memoryStructure) ListLookups(_ context.Context, kind repository.LookupKind) ([]model.LookupItem, error) {
	var result []model.LookupItem
	for _, item := range m.lookups[kind] {
		result = append(result, *item)
	}
	return result, nil
}

func (m *memoryStructure) FindLookupByID(_ context.Context, kind repository.LookupKind, id uint64) (*model.LookupItem, error) {
	item, ok := m.lookups[kind][id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copy := *item
	return &copy, nil
}

func (m *memoryStructure) CreateLookup(_ context.Context, kind repository.LookupKind, item *model.LookupItem) error {
	if m.lookups[kind] == nil {
		m.lookups[kind] = make(map[uint64]*model.LookupItem)
	}
	item.ID = m.id()
	copy := *item
	m.lookups[kind][item.ID] = &copy
	return nil
}

func (m *memoryStructure) UpdateLookup(_ context.Context, kind repository.LookupKind, item *model.LookupItem) error {
	if _, ok := m.lookups[kind][item.ID]; !ok {
		return repository.ErrNotFound
	}
	copy := *item
	m.lookups[kind][item.ID] = &copy
	return nil
}

func (m *memoryStructure) CreateLevel(_ context.Context, level *model.Level) error {
	level.ID = m.id()
	copy := *level
	m.levels[level.ID] = &copy
	return nil
}

func (m *memoryStructure) ListLevels(_ context.Context, schoolID uint64) ([]model.Level, error) {
	var result []model.Level
	for _, level := range m.levels {
		if level.SchoolID == schoolID {
			result = append(result, *level)
		}
	}
	return result, nil
}

func (m *memoryStructure) FindLevelByID(_ context.Context, schoolID, id uint64) (*model.Level, error) {
	level, ok := m.levels[id]
	if !ok || level.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *level
	return &copy, nil
}

func (m *memoryStructure) UpdateLevel(_ context.Context, level *model.Level) error {
	if _, ok := m.levels[level.ID]; !ok {
		return repository.ErrNotFound
	}
	copy := *level
	m.levels[level.ID] = &copy
	return nil
}

func (m *memoryStructure) CountActiveClassesByLevel(_ context.Context, schoolID, levelID uint64) (int64, error) {
	var count int64
	for _, class := range m.classes {
		if class.SchoolID == schoolID && class.LevelID == levelID && class.Status == model.RecordStatusActive {
			count++
		}
	}
	return count, nil
}

func (m *memoryStructure) CreateClass(_ context.Context, class *model.Class) error {
	class.ID = m.id()
	copy := *class
	m.classes[class.ID] = &copy
	return nil
}

func (m *memoryStructure) ListClasses(_ context.Context, schoolID uint64) ([]model.Class, error) {
	var result []model.Class
	for _, class := range m.classes {
		if class.SchoolID == schoolID {
			result = append(result, *class)
		}
	}
	return result, nil
}

func (m *memoryStructure) FindClassByID(_ context.Context, schoolID, id uint64) (*model.Class, error) {
	class, ok := m.classes[id]
	if !ok || class.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *class
	return &copy, nil
}

func (m *memoryStructure) UpdateClass(_ context.Context, class *model.Class) error {
	if _, ok := m.classes[class.ID]; !ok {
		return repository.ErrNotFound
	}
	copy := *class
	m.classes[class.ID] = &copy
	return nil
}
