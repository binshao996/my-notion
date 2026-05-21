package page

import (
	"testing"

	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&db.Workspace{}, &db.Page{}); err != nil {
		t.Fatal(err)
	}
	return database
}

func createTestWorkspace(t *testing.T, gdb *gorm.DB, name string) *db.Workspace {
	ws := &db.Workspace{Name: name}
	if err := gdb.Create(ws).Error; err != nil {
		t.Fatal(err)
	}
	return ws
}

func createTestPage(t *testing.T, gdb *gorm.DB, workspaceID, createdBy uint, title string, parentPageID *uint) *db.Page {
	page := &db.Page{
		WorkspaceID:  workspaceID,
		CreatedBy:    createdBy,
		Title:        title,
		ParentPageID: parentPageID,
	}
	if err := gdb.Create(page).Error; err != nil {
		t.Fatal(err)
	}
	return page
}

func ptr[T any](v T) *T {
	return &v
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"creates non-nil service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gdb := setupTestDB(t)
			svc := NewService(gdb)

			if svc == nil {
				t.Fatal("NewService returned nil")
			}
			if svc.DB != gdb {
				t.Error("service.DB does not match the database argument")
			}
			if svc.SearchService != nil {
				t.Error("expected SearchService to be nil by default")
			}
		})
	}
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name         string
		parentPageID *uint
		wantParent   *uint
	}{
		{
			name:         "create page with no parent",
			parentPageID: nil,
			wantParent:   nil,
		},
		{
			name:         "create page with parent",
			parentPageID: nil, // will be set dynamically
			wantParent:   nil, // will be set dynamically
		},
	}

	gdb := setupTestDB(t)
	ws := createTestWorkspace(t, gdb, "Create Test WS")
	createdBy := uint(1)

	// First create the parent page for the second test case
	parentPage := createTestPage(t, gdb, ws.ID, createdBy, "Parent Page", nil)
	tests[1].parentPageID = ptr(parentPage.ID)
	tests[1].wantParent = ptr(parentPage.ID)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(gdb)
			title := "Test Page"
			if tt.parentPageID != nil {
				title = "Child Page"
			}

			page, err := svc.Create(ws.ID, createdBy, title, tt.parentPageID)
			if err != nil {
				t.Fatalf("Create returned error: %v", err)
			}
			if page == nil {
				t.Fatal("Create returned nil page")
			}
			if page.WorkspaceID != ws.ID {
				t.Errorf("WorkspaceID = %d, want %d", page.WorkspaceID, ws.ID)
			}
			if page.CreatedBy != createdBy {
				t.Errorf("CreatedBy = %d, want %d", page.CreatedBy, createdBy)
			}
			if page.Title != title {
				t.Errorf("Title = %q, want %q", page.Title, title)
			}
			if page.ID == 0 {
				t.Error("page ID should not be zero after creation")
			}

			// Verify ParentPageID
			if tt.wantParent == nil {
				if page.ParentPageID != nil {
					t.Errorf("ParentPageID = %v, want nil", *page.ParentPageID)
				}
			} else {
				if page.ParentPageID == nil || *page.ParentPageID != *tt.wantParent {
					t.Errorf("ParentPageID = %v, want %v", page.ParentPageID, tt.wantParent)
				}
			}

			// Verify page was actually persisted
			fetched, err := svc.GetByID(page.ID)
			if err != nil {
				t.Fatalf("failed to fetch created page: %v", err)
			}
			if fetched.Title != title {
				t.Errorf("fetched Title = %q, want %q", fetched.Title, title)
			}
		})
	}
}

func TestGetByID(t *testing.T) {
	gdb := setupTestDB(t)
	ws := createTestWorkspace(t, gdb, "GetByID Test WS")
	createdBy := uint(1)

	existingPage := createTestPage(t, gdb, ws.ID, createdBy, "Existing Page", nil)
	svc := NewService(gdb)

	tests := []struct {
		name      string
		id        uint
		wantErr   bool
		wantTitle string
	}{
		{
			name:      "get existing page",
			id:        existingPage.ID,
			wantErr:   false,
			wantTitle: "Existing Page",
		},
		{
			name:      "get non-existent page",
			id:        99999,
			wantErr:   true,
			wantTitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := svc.GetByID(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if err.Error() != "page not found" {
					t.Errorf("error = %q, want \"page not found\"", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByID returned error: %v", err)
			}
			if page == nil {
				t.Fatal("GetByID returned nil page")
			}
			if page.ID != tt.id {
				t.Errorf("ID = %d, want %d", page.ID, tt.id)
			}
			if page.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", page.Title, tt.wantTitle)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	gdb := setupTestDB(t)
	ws := createTestWorkspace(t, gdb, "Update Test WS")
	createdBy := uint(1)
	svc := NewService(gdb)

	existingPage := createTestPage(t, gdb, ws.ID, createdBy, "Original Title", nil)

	tests := []struct {
		name          string
		id            uint
		updates       map[string]interface{}
		wantErr       bool
		errMsg        string
		checkField    string
		checkValue    interface{}
		checkByGet    bool // use GetByID after update to verify field
	}{
		{
			name:       "update title of existing page",
			id:         existingPage.ID,
			updates:    map[string]interface{}{"title": "Updated Title"},
			wantErr:    false,
			checkField: "title",
			checkValue: "Updated Title",
			checkByGet: true,
		},
		{
			name:       "update icon of existing page",
			id:         existingPage.ID,
			updates:    map[string]interface{}{"icon": "rocket"},
			wantErr:    false,
			checkField: "icon",
			checkValue: "rocket",
			checkByGet: true,
		},
		{
			name:       "update cover of existing page",
			id:         existingPage.ID,
			updates:    map[string]interface{}{"cover": "https://example.com/cover.png"},
			wantErr:    false,
			checkField: "cover",
			checkValue: "https://example.com/cover.png",
			checkByGet: true,
		},
		{
			name:       "archive page via update",
			id:         existingPage.ID,
			updates:    map[string]interface{}{"archived": true},
			wantErr:    false,
			checkField: "archived",
			checkValue: true,
			checkByGet: true,
		},
		{
			name:       "update multiple fields",
			id:         existingPage.ID,
			updates:    map[string]interface{}{"title": "Multi Update", "icon": "multi-icon"},
			wantErr:    false,
			checkField: "title",
			checkValue: "Multi Update",
			checkByGet: true,
		},
		{
			name:    "update non-existent page",
			id:      99999,
			updates: map[string]interface{}{"title": "x"},
			wantErr: true,
			errMsg:  "page not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, err := svc.Update(tt.id, tt.updates)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("Update returned error: %v", err)
			}
			if updated == nil {
				t.Fatal("Update returned nil page")
			}
			if updated.ID != tt.id {
				t.Errorf("ID = %d, want %d", updated.ID, tt.id)
			}

			// Verify updates were persisted by fetching from DB
			fetched, err := svc.GetByID(tt.id)
			if err != nil {
				t.Fatalf("GetByID after update failed: %v", err)
			}
			switch tt.checkField {
			case "title":
				if fetched.Title != tt.checkValue.(string) {
					t.Errorf("Title = %q, want %q", fetched.Title, tt.checkValue)
				}
			case "icon":
				if fetched.Icon != tt.checkValue.(string) {
					t.Errorf("Icon = %q, want %q", fetched.Icon, tt.checkValue)
				}
			case "cover":
				if fetched.Cover != tt.checkValue.(string) {
					t.Errorf("Cover = %q, want %q", fetched.Cover, tt.checkValue)
				}
			case "archived":
				if fetched.Archived != tt.checkValue.(bool) {
					t.Errorf("Archived = %v, want %v", fetched.Archived, tt.checkValue)
				}
			}
		})
	}
}

func TestGetChildren(t *testing.T) {
	gdb := setupTestDB(t)
	ws := createTestWorkspace(t, gdb, "Children Test WS")
	createdBy := uint(1)

	parentPage := createTestPage(t, gdb, ws.ID, createdBy, "Parent", nil)
	svc := NewService(gdb)

	// Create children for the parent
	createTestPage(t, gdb, ws.ID, createdBy, "Child C", ptr(parentPage.ID))
	createTestPage(t, gdb, ws.ID, createdBy, "Child A", ptr(parentPage.ID))
	createTestPage(t, gdb, ws.ID, createdBy, "Child B", ptr(parentPage.ID))
	// Create an archived child
	archivedChild := createTestPage(t, gdb, ws.ID, createdBy, "Archived Child", ptr(parentPage.ID))
	gdb.Model(archivedChild).Update("archived", true)

	// Create a page under a different parent (should not appear)
	otherParent := createTestPage(t, gdb, ws.ID, createdBy, "Other Parent", nil)
	createTestPage(t, gdb, ws.ID, createdBy, "Other Child", ptr(otherParent.ID))

	tests := []struct {
		name      string
		parentID  uint
		wantCount int
		wantOrder []string // expected titles in order
	}{
		{
			name:      "returns non-archived children sorted by title",
			parentID:  parentPage.ID,
			wantCount: 3,
			wantOrder: []string{"Child A", "Child B", "Child C"},
		},
		{
			name:      "empty parent returns no children",
			parentID:  99999,
			wantCount: 0,
			wantOrder: nil,
		},
		{
			name:      "other parent has one child",
			parentID:  otherParent.ID,
			wantCount: 1,
			wantOrder: []string{"Other Child"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			children, err := svc.GetChildren(tt.parentID)
			if err != nil {
				t.Fatalf("GetChildren returned error: %v", err)
			}
			if len(children) != tt.wantCount {
				t.Fatalf("got %d children, want %d", len(children), tt.wantCount)
			}
			if tt.wantOrder != nil {
				for i, want := range tt.wantOrder {
					if children[i].Title != want {
						t.Errorf("children[%d].Title = %q, want %q", i, children[i].Title, want)
					}
				}
			}
		})
	}
}

func TestGetChildrenExcludesArchived(t *testing.T) {
	gdb := setupTestDB(t)
	ws := createTestWorkspace(t, gdb, "Children Archived Test WS")
	createdBy := uint(1)
	svc := NewService(gdb)

	parentPage := createTestPage(t, gdb, ws.ID, createdBy, "Parent", nil)
	createTestPage(t, gdb, ws.ID, createdBy, "Visible Child", ptr(parentPage.ID))
	archivedChild := createTestPage(t, gdb, ws.ID, createdBy, "Hidden Child", ptr(parentPage.ID))
	gdb.Model(archivedChild).Update("archived", true)

	children, err := svc.GetChildren(parentPage.ID)
	if err != nil {
		t.Fatalf("GetChildren returned error: %v", err)
	}
	if len(children) != 1 {
		t.Fatalf("got %d children, want 1 (archived should be excluded)", len(children))
	}
	if children[0].Title != "Visible Child" {
		t.Errorf("Title = %q, want \"Visible Child\"", children[0].Title)
	}
}

func TestGetWorkspaceTree(t *testing.T) {
	gdb := setupTestDB(t)
	ws := createTestWorkspace(t, gdb, "Tree Test WS")
	otherWS := createTestWorkspace(t, gdb, "Other WS")
	createdBy := uint(1)

	// Create root-level pages in workspace
	createTestPage(t, gdb, ws.ID, createdBy, "Root C", nil)
	createTestPage(t, gdb, ws.ID, createdBy, "Root A", nil)
	createTestPage(t, gdb, ws.ID, createdBy, "Root B", nil)

	// Create an archived root page
	archivedRoot := createTestPage(t, gdb, ws.ID, createdBy, "Archived Root", nil)
	gdb.Model(archivedRoot).Update("archived", true)

	// Create child pages (should not appear in tree)
	parentPage := createTestPage(t, gdb, ws.ID, createdBy, "Parent Page", nil)
	createTestPage(t, gdb, ws.ID, createdBy, "Child Page", ptr(parentPage.ID))

	// Create root pages in another workspace (should not appear)
	createTestPage(t, gdb, otherWS.ID, createdBy, "Other WS Page", nil)

	svc := NewService(gdb)

	tests := []struct {
		name        string
		workspaceID uint
		wantCount   int
		wantOrder   []string
	}{
		{
			name:        "returns root pages sorted by title, excluding archived and children",
			workspaceID: ws.ID,
			wantCount:   4, // Root A, Root B, Root C, Parent Page
			wantOrder:   []string{"Parent Page", "Root A", "Root B", "Root C"},
		},
		{
			name:        "other workspace has its own root pages",
			workspaceID: otherWS.ID,
			wantCount:   1,
			wantOrder:   []string{"Other WS Page"},
		},
		{
			name:        "non-existent workspace returns empty",
			workspaceID: 99999,
			wantCount:   0,
			wantOrder:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, err := svc.GetWorkspaceTree(tt.workspaceID)
			if err != nil {
				t.Fatalf("GetWorkspaceTree returned error: %v", err)
			}
			if len(tree) != tt.wantCount {
				t.Fatalf("got %d pages, want %d", len(tree), tt.wantCount)
			}
			if tt.wantOrder != nil {
				for i, want := range tt.wantOrder {
					if tree[i].Title != want {
						t.Errorf("tree[%d].Title = %q, want %q", i, tree[i].Title, want)
					}
				}
			}
		})
	}
}

func TestGetWorkspaceTreeExcludesArchived(t *testing.T) {
	gdb := setupTestDB(t)
	ws := createTestWorkspace(t, gdb, "Tree Archived Test WS")
	createdBy := uint(1)
	svc := NewService(gdb)

	createTestPage(t, gdb, ws.ID, createdBy, "Visible Root", nil)
	archivedRoot := createTestPage(t, gdb, ws.ID, createdBy, "Hidden Root", nil)
	gdb.Model(archivedRoot).Update("archived", true)

	tree, err := svc.GetWorkspaceTree(ws.ID)
	if err != nil {
		t.Fatalf("GetWorkspaceTree returned error: %v", err)
	}
	if len(tree) != 1 {
		t.Fatalf("got %d pages, want 1 (archived should be excluded)", len(tree))
	}
	if tree[0].Title != "Visible Root" {
		t.Errorf("Title = %q, want \"Visible Root\"", tree[0].Title)
	}
}

func TestCRUDRoundTrip(t *testing.T) {
	gdb := setupTestDB(t)
	ws := createTestWorkspace(t, gdb, "RoundTrip WS")
	createdBy := uint(42)
	svc := NewService(gdb)

	// Step 1: Create
	title := "CRUD Test Page"
	created, err := svc.Create(ws.ID, createdBy, title, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Title != title {
		t.Errorf("after Create: Title = %q, want %q", created.Title, title)
	}
	if created.WorkspaceID != ws.ID {
		t.Errorf("after Create: WorkspaceID = %d, want %d", created.WorkspaceID, ws.ID)
	}

	// Step 2: GetByID
	fetched, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("after GetByID: ID = %d, want %d", fetched.ID, created.ID)
	}
	if fetched.Title != title {
		t.Errorf("after GetByID: Title = %q, want %q", fetched.Title, title)
	}
	if fetched.CreatedBy != createdBy {
		t.Errorf("after GetByID: CreatedBy = %d, want %d", fetched.CreatedBy, createdBy)
	}
	if fetched.Archived != false {
		t.Error("after GetByID: Archived should default to false")
	}

	// Step 3: Update title
	newTitle := "Updated CRUD Title"
	updated, err := svc.Update(created.ID, map[string]interface{}{"title": newTitle})
	if err != nil {
		t.Fatalf("Update title failed: %v", err)
	}
	if updated == nil {
		t.Fatal("Update returned nil")
	}

	// Step 4: GetByID to verify update persisted
	fetched2, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if fetched2.Title != newTitle {
		t.Errorf("after Update: Title = %q, want %q", fetched2.Title, newTitle)
	}

	// Step 5: Archive the page
	_, err = svc.Update(created.ID, map[string]interface{}{"archived": true})
	if err != nil {
		t.Fatalf("Archive update failed: %v", err)
	}
	fetched3, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID after archive failed: %v", err)
	}
	if fetched3.Archived != true {
		t.Error("after Archive: Archived should be true")
	}

	// Step 6: Verify archived page does not appear in tree or children
	tree, err := svc.GetWorkspaceTree(ws.ID)
	if err != nil {
		t.Fatalf("GetWorkspaceTree failed: %v", err)
	}
	for _, p := range tree {
		if p.ID == created.ID {
			t.Error("archived page should not be in workspace tree")
		}
	}

	// Step 7: Create a non-archived child under the archived page.
	// GetChildren filters by child.archived = false, not parent.archived,
	// so the child should still appear.
	createTestPage(t, gdb, ws.ID, createdBy, "Child of Archived", ptr(created.ID))
	children, err := svc.GetChildren(created.ID)
	if err != nil {
		t.Fatalf("GetChildren failed: %v", err)
	}
	found := false
	for _, c := range children {
		if c.Title == "Child of Archived" {
			found = true
			break
		}
	}
	if !found {
		t.Error("non-archived child should appear under archived parent (GetChildren checks child.archived, not parent.archived)")
	}
}

func TestSearchServiceNilSafety(t *testing.T) {
	gdb := setupTestDB(t)
	ws := createTestWorkspace(t, gdb, "NilSearch WS")
	createdBy := uint(1)
	svc := NewService(gdb)

	// Verify SearchService is nil
	if svc.SearchService != nil {
		t.Fatal("expected SearchService to be nil")
	}

	// Create should not panic with nil SearchService
	page, err := svc.Create(ws.ID, createdBy, "Safe Create", nil)
	if err != nil {
		t.Fatalf("Create with nil SearchService failed: %v", err)
	}
	if page == nil {
		t.Fatal("Create returned nil page")
	}

	// Update should not panic with nil SearchService
	_, err = svc.Update(page.ID, map[string]interface{}{"title": "Safe Update"})
	if err != nil {
		t.Fatalf("Update with nil SearchService failed: %v", err)
	}
}
