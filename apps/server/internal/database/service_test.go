package database

import (
	"testing"

	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	gdb.AutoMigrate(
		&db.Workspace{}, &db.Page{}, &db.Database{},
		&db.Property{}, &db.Record{}, &db.PropertyValue{}, &db.View{},
	)
	return gdb
}

func seedTestData(gdb *gorm.DB) (workspaceID, databaseID uint) {
	ws := db.Workspace{Name: "test-workspace"}
	gdb.Create(&ws)

	page := db.Page{WorkspaceID: ws.ID, Title: "DB Page", CreatedBy: 1}
	gdb.Create(&page)

	dbase := db.Database{WorkspaceID: ws.ID, PageID: page.ID, Name: "Test DB"}
	gdb.Create(&dbase)

	return ws.ID, dbase.ID
}

// --- DatabaseService tests ---

func TestNewService(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewService(gdb)
	assert.NotNil(t, svc, "NewService should return a non-nil service")
}

func TestServiceCreate(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewService(gdb)

	t.Run("creates database with correct fields", func(t *testing.T) {
		ws := db.Workspace{Name: "ws"}
		gdb.Create(&ws)

		database, err := svc.Create(ws.ID, "My Database", 42)
		assert.NoError(t, err)
		assert.NotNil(t, database)
		assert.Greater(t, database.ID, uint(0), "ID should be non-zero")
		assert.Equal(t, ws.ID, database.WorkspaceID)
		assert.Equal(t, "My Database", database.Name)
		assert.NotZero(t, database.PageID, "should have an associated page")
	})

	t.Run("creates associated page", func(t *testing.T) {
		ws := db.Workspace{Name: "ws2"}
		gdb.Create(&ws)

		database, err := svc.Create(ws.ID, "DB with Page", 99)
		assert.NoError(t, err)

		var page db.Page
		err = gdb.First(&page, database.PageID).Error
		assert.NoError(t, err)
		assert.Equal(t, ws.ID, page.WorkspaceID)
		assert.Equal(t, "DB with Page", page.Title)
		assert.Equal(t, uint(99), page.CreatedBy)
	})

	t.Run("creates title property automatically", func(t *testing.T) {
		ws := db.Workspace{Name: "ws3"}
		gdb.Create(&ws)

		database, err := svc.Create(ws.ID, "DB with Title", 1)
		assert.NoError(t, err)

		var props []db.Property
		gdb.Where("database_id = ?", database.ID).Find(&props)
		assert.Len(t, props, 1)
		assert.Equal(t, "Name", props[0].Name)
		assert.Equal(t, "title", props[0].Type)
		assert.Equal(t, "a0", props[0].Position)
	})
}

func TestServiceGetByID(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewService(gdb)
	_, databaseID := seedTestData(gdb)

	t.Run("returns database for valid ID", func(t *testing.T) {
		result, err := svc.GetByID(databaseID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, databaseID, result.ID)
		assert.Equal(t, "Test DB", result.Name)
	})

	t.Run("returns error for nonexistent ID", func(t *testing.T) {
		result, err := svc.GetByID(99999)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestServiceGetByPageID(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewService(gdb)
	_, databaseID := seedTestData(gdb)

	t.Run("returns database for valid page ID", func(t *testing.T) {
		var dbase db.Database
		gdb.First(&dbase, databaseID)

		result, err := svc.GetByPageID(dbase.PageID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, databaseID, result.ID)
	})

	t.Run("returns error for nonexistent page ID", func(t *testing.T) {
		result, err := svc.GetByPageID(88888)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestServiceUpdate(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewService(gdb)
	_, databaseID := seedTestData(gdb)

	t.Run("updates database name", func(t *testing.T) {
		updates := map[string]interface{}{"name": "Renamed DB"}
		result, err := svc.Update(databaseID, updates)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Renamed DB", result.Name)
	})

	t.Run("returns error for nonexistent ID", func(t *testing.T) {
		updates := map[string]interface{}{"name": "Ghost"}
		result, err := svc.Update(77777, updates)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("zero-value name update", func(t *testing.T) {
		_, db2ID := seedTestData(gdb)
		updates := map[string]interface{}{"name": ""}
		result, err := svc.Update(db2ID, updates)
		assert.NoError(t, err)
		assert.Equal(t, "", result.Name)
	})
}

func TestServiceDelete(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewService(gdb)

	t.Run("deletes existing database", func(t *testing.T) {
		_, databaseID := seedTestData(gdb)

		err := svc.Delete(databaseID)
		assert.NoError(t, err)

		// Verify database is gone
		var dbase db.Database
		err = gdb.First(&dbase, databaseID).Error
		assert.Error(t, err)
	})

	t.Run("returns error for nonexistent database", func(t *testing.T) {
		err := svc.Delete(12345)
		assert.Error(t, err)
	})

	t.Run("cascade deletes associated records", func(t *testing.T) {
		_, databaseID := seedTestData(gdb)

		// Create a record in this database
		record := db.Record{DatabaseID: databaseID, PageID: 999, Position: "a0"}
		gdb.Create(&record)

		err := svc.Delete(databaseID)
		assert.NoError(t, err)

		// Verify records are deleted
		var records []db.Record
		gdb.Where("database_id = ?", databaseID).Find(&records)
		assert.Empty(t, records)
	})
}

// --- PropertyService tests ---

func TestPropertyServiceCreate(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewPropertyService(gdb)
	_, databaseID := seedTestData(gdb)

	t.Run("creates property with correct fields", func(t *testing.T) {
		prop, err := svc.Create(databaseID, "Status", "select", `{"options":["Done","Todo"]}`)
		assert.NoError(t, err)
		assert.NotNil(t, prop)
		assert.NotZero(t, prop.ID)
		assert.Equal(t, databaseID, prop.DatabaseID)
		assert.Equal(t, "Status", prop.Name)
		assert.Equal(t, "select", prop.Type)
		assert.Contains(t, prop.Config, "Done")
		assert.Equal(t, "a0", prop.Position)
	})

	t.Run("assigns position to subsequent properties", func(t *testing.T) {
		p1, _ := svc.Create(databaseID, "Prop1", "text", "{}")
		p2, _ := svc.Create(databaseID, "Prop2", "number", "{}")
		p3, _ := svc.Create(databaseID, "Prop3", "text", "{}")

		// Lexicographic ordering: a0 < a1 < a2
		assert.True(t, p1.Position < p2.Position, "p2 should be after p1")
		assert.True(t, p2.Position < p3.Position, "p3 should be after p2")
	})

	t.Run("creates property without config", func(t *testing.T) {
		_, dbID := seedTestData(gdb)
		prop, err := svc.Create(dbID, "Simple", "text", "")
		assert.NoError(t, err)
		assert.Equal(t, "text", prop.Type)
	})
}

func TestPropertyServiceListByDatabase(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewPropertyService(gdb)
	_, databaseID := seedTestData(gdb)

	t.Run("lists properties ordered by position", func(t *testing.T) {
		svc.Create(databaseID, "ZZZ", "text", "{}")
		svc.Create(databaseID, "AAA", "text", "{}")

		props, err := svc.ListByDatabase(databaseID)
		assert.NoError(t, err)
		assert.Len(t, props, 2)
		assert.Equal(t, "ZZZ", props[0].Name, "first created should be first")
		assert.Equal(t, "AAA", props[1].Name)
	})

	t.Run("returns empty list when no properties", func(t *testing.T) {
		_, emptyDBID := seedTestData(gdb)
		props, err := svc.ListByDatabase(emptyDBID)
		assert.NoError(t, err)
		assert.Empty(t, props)
	})
}

func TestPropertyServiceUpdate(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewPropertyService(gdb)
	_, databaseID := seedTestData(gdb)

	t.Run("updates property name", func(t *testing.T) {
		prop, _ := svc.Create(databaseID, "Old Name", "text", "{}")

		updated, err := svc.Update(prop.ID, map[string]interface{}{"name": "New Name"})
		assert.NoError(t, err)
		assert.Equal(t, "New Name", updated.Name)
	})

	t.Run("updates property config", func(t *testing.T) {
		prop, _ := svc.Create(databaseID, "Options", "select", `{"options":["A"]}`)

		updated, err := svc.Update(prop.ID, map[string]interface{}{"config": `{"options":["A","B"]}`})
		assert.NoError(t, err)
		assert.Contains(t, updated.Config, "B")
	})

	t.Run("returns error for nonexistent", func(t *testing.T) {
		result, err := svc.Update(99999, map[string]interface{}{"name": "Nope"})
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestPropertyServiceDelete(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewPropertyService(gdb)
	_, databaseID := seedTestData(gdb)

	t.Run("deletes existing property", func(t *testing.T) {
		prop, _ := svc.Create(databaseID, "TempProp", "text", "{}")

		err := svc.Delete(prop.ID)
		assert.NoError(t, err)

		var found db.Property
		err = gdb.First(&found, prop.ID).Error
		assert.Error(t, err, "property should no longer exist")
	})

	t.Run("deletes associated property values", func(t *testing.T) {
		prop, _ := svc.Create(databaseID, "WithValues", "text", "{}")

		// Create a record + property value
		record := db.Record{DatabaseID: databaseID, PageID: 888, Position: "z9"}
		gdb.Create(&record)
		pv := db.PropertyValue{RecordID: record.ID, PropertyID: prop.ID, Value: `{"text":"hello"}`}
		gdb.Create(&pv)

		err := svc.Delete(prop.ID)
		assert.NoError(t, err)

		var vals []db.PropertyValue
		gdb.Where("property_id = ?", prop.ID).Find(&vals)
		assert.Empty(t, vals, "property values should be cascade-deleted")
	})
}

// --- ViewService tests ---

func TestViewServiceCreate(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewViewService(gdb)
	_, databaseID := seedTestData(gdb)

	t.Run("creates view with correct fields", func(t *testing.T) {
		view, err := svc.Create(databaseID, "Grid View", "grid")
		assert.NoError(t, err)
		assert.NotNil(t, view)
		assert.NotZero(t, view.ID)
		assert.Equal(t, databaseID, view.DatabaseID)
		assert.Equal(t, "Grid View", view.Name)
		assert.Equal(t, "grid", view.Type)
		assert.Equal(t, "{}", view.Config)
		assert.Equal(t, "a0", view.Position)
	})

	t.Run("assigns sequential positions", func(t *testing.T) {
		v1, _ := svc.Create(databaseID, "View 1", "table")
		v2, _ := svc.Create(databaseID, "View 2", "board")
		v3, _ := svc.Create(databaseID, "View 3", "calendar")

		assert.True(t, v1.Position < v2.Position)
		assert.True(t, v2.Position < v3.Position)
	})

	t.Run("creates views of different types", func(t *testing.T) {
		_, dbID := seedTestData(gdb)
		for _, vtype := range []string{"table", "board", "calendar", "list", "gallery", "timeline"} {
			view, err := svc.Create(dbID, vtype, vtype)
			assert.NoError(t, err)
			assert.Equal(t, vtype, view.Type)
		}
	})
}

func TestViewServiceListByDatabase(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewViewService(gdb)
	_, databaseID := seedTestData(gdb)

	t.Run("lists views ordered by position", func(t *testing.T) {
		svc.Create(databaseID, "First", "table")
		svc.Create(databaseID, "Second", "board")
		svc.Create(databaseID, "Third", "calendar")

		views, err := svc.ListByDatabase(databaseID)
		assert.NoError(t, err)
		assert.Len(t, views, 3)
		assert.Equal(t, "First", views[0].Name)
		assert.Equal(t, "Second", views[1].Name)
		assert.Equal(t, "Third", views[2].Name)
	})

	t.Run("returns empty when no views exist", func(t *testing.T) {
		_, emptyDBID := seedTestData(gdb)
		views, err := svc.ListByDatabase(emptyDBID)
		assert.NoError(t, err)
		assert.Empty(t, views)
	})
}

func TestViewServiceUpdate(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewViewService(gdb)
	_, databaseID := seedTestData(gdb)

	t.Run("updates view name", func(t *testing.T) {
		view, _ := svc.Create(databaseID, "Old View", "table")
		updated, err := svc.Update(view.ID, map[string]interface{}{"name": "New View"})
		assert.NoError(t, err)
		assert.Equal(t, "New View", updated.Name)
	})

	t.Run("updates view config", func(t *testing.T) {
		view, _ := svc.Create(databaseID, "Config View", "table")
		updated, err := svc.Update(view.ID, map[string]interface{}{"config": `{"sortColumn":"id"}`})
		assert.NoError(t, err)
		assert.Contains(t, updated.Config, "sortColumn")
	})

	t.Run("returns error for nonexistent", func(t *testing.T) {
		result, err := svc.Update(99999, map[string]interface{}{"name": "Ghost"})
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestViewServiceDelete(t *testing.T) {
	gdb := setupTestDB(t)
	svc := NewViewService(gdb)
	_, databaseID := seedTestData(gdb)

	t.Run("deletes existing view", func(t *testing.T) {
		view, _ := svc.Create(databaseID, "ToDelete", "table")
		err := svc.Delete(view.ID)
		assert.NoError(t, err)

		var found db.View
		err = gdb.First(&found, view.ID).Error
		assert.Error(t, err, "view should no longer exist")
	})

	// GORM's Delete returns nil error even when no rows match, matching
	// the convention of other GORM write operations.
}
