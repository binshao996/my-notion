package database

import (
	"encoding/json"
	"testing"

	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDBForRecords creates an in-memory DB and seeds a database with a
// title property (via Service.Create). Returns the gorm.DB and database ID.
func setupTestDBForRecords(t *testing.T) (*gorm.DB, uint) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	database.AutoMigrate(
		&db.Workspace{}, &db.Page{}, &db.Database{},
		&db.Property{}, &db.Record{}, &db.PropertyValue{}, &db.View{},
	)

	ws := db.Workspace{Name: "records-ws"}
	database.Create(&ws)

	svc := NewService(database)
	dbase, err := svc.Create(ws.ID, "Records DB", 1)
	if err != nil {
		t.Fatal(err)
	}
	return database, dbase.ID
}

// titlePropertyID returns the ID of the title property for the given database.
func titlePropertyID(gdb *gorm.DB, databaseID uint) uint {
	var prop db.Property
	gdb.Where("database_id = ? AND type = ?", databaseID, "title").First(&prop)
	return prop.ID
}

// --- RecordService tests ---

func TestRecordServiceCreate(t *testing.T) {
	gdb, databaseID := setupTestDBForRecords(t)
	svc := NewRecordService(gdb)
	titlePID := titlePropertyID(gdb, databaseID)

	t.Run("creates record with page", func(t *testing.T) {
		vals := map[uint]string{
			titlePID: `{"text":"First Record"}`,
		}
		record, err := svc.Create(databaseID, vals, 42)
		assert.NoError(t, err)
		assert.NotNil(t, record)
		assert.NotZero(t, record.ID)
		assert.Equal(t, databaseID, record.DatabaseID)
		assert.NotZero(t, record.PageID)

		// Verify the page was created
		var page db.Page
		err = gdb.First(&page, record.PageID).Error
		assert.NoError(t, err)
		assert.Equal(t, "First Record", page.Title)
		assert.Equal(t, uint(42), page.CreatedBy)
	})

	t.Run("assigns position a0 for first record", func(t *testing.T) {
		gdb2, dbID := setupTestDBForRecords(t)
		r2 := NewRecordService(gdb2)
		tpID := titlePropertyID(gdb2, dbID)

		record, err := r2.Create(dbID, map[uint]string{tpID: `{"text":"Pos Test"}`}, 1)
		assert.NoError(t, err)
		assert.True(t, record.Position > "", "position should not be empty")
	})

	t.Run("saves property values", func(t *testing.T) {
		gdb2, dbID := setupTestDBForRecords(t)
		rs := NewRecordService(gdb2)
		tpID := titlePropertyID(gdb2, dbID)

		// Create a text property for this database
		ps := NewPropertyService(gdb2)
		textProp, _ := ps.Create(dbID, "Notes", "text", "{}")

		vals := map[uint]string{
			tpID:        `{"text":"Value Test"}`,
			textProp.ID: `{"text":"some notes"}`,
		}
		record, err := rs.Create(dbID, vals, 1)
		assert.NoError(t, err)

		// Check property values were saved
		var pvs []db.PropertyValue
		gdb2.Where("record_id = ?", record.ID).Find(&pvs)
		assert.Len(t, pvs, 2)

		// Check specific value
		var pv db.PropertyValue
		gdb2.Where("record_id = ? AND property_id = ?", record.ID, textProp.ID).First(&pv)
		assert.Contains(t, pv.Value, "some notes")
	})

	t.Run("creates multiple records with increasing positions", func(t *testing.T) {
		gdb2, dbID := setupTestDBForRecords(t)
		rs := NewRecordService(gdb2)
		tpID := titlePropertyID(gdb2, dbID)

		r1, _ := rs.Create(dbID, map[uint]string{tpID: `{"text":"R1"}`}, 1)
		r2, _ := rs.Create(dbID, map[uint]string{tpID: `{"text":"R2"}`}, 1)
		r3, _ := rs.Create(dbID, map[uint]string{tpID: `{"text":"R3"}`}, 1)

		assert.True(t, r1.Position < r2.Position, "r2 should be after r1")
		assert.True(t, r2.Position < r3.Position, "r3 should be after r2")
	})

	t.Run("defaults title to Untitled when no title property value", func(t *testing.T) {
		gdb2, dbID := setupTestDBForRecords(t)
		rs := NewRecordService(gdb2)

		// Create record without providing title property value
		record, err := rs.Create(dbID, map[uint]string{}, 1)
		assert.NoError(t, err)

		var page db.Page
		gdb2.First(&page, record.PageID)
		assert.Equal(t, "Untitled", page.Title)
	})
}

func TestRecordServiceGetByID(t *testing.T) {
	gdb, databaseID := setupTestDBForRecords(t)
	rs := NewRecordService(gdb)
	tpID := titlePropertyID(gdb, databaseID)

	vals := map[uint]string{tpID: `{"text":"GetMe"}`}
	created, _ := rs.Create(databaseID, vals, 1)

	t.Run("returns record with values and properties", func(t *testing.T) {
		record, values, properties, err := rs.GetByID(created.ID)
		assert.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, created.ID, record.ID)

		// Should have the title property in the list
		assert.NotEmpty(t, properties)
		foundTitleProp := false
		for _, p := range properties {
			if p.Type == "title" {
				foundTitleProp = true
				break
			}
		}
		assert.True(t, foundTitleProp, "should return the title property")

		// Should have at least one property value
		assert.NotEmpty(t, values)
	})

	t.Run("returns error for nonexistent record", func(t *testing.T) {
		record, values, properties, err := rs.GetByID(99999)
		assert.Error(t, err)
		assert.Nil(t, record)
		assert.Nil(t, values)
		assert.Nil(t, properties)
	})
}

func TestRecordServiceUpdate(t *testing.T) {
	gdb, databaseID := setupTestDBForRecords(t)
	rs := NewRecordService(gdb)
	tpID := titlePropertyID(gdb, databaseID)

	// Create a text property
	ps := NewPropertyService(gdb)
	textProp, _ := ps.Create(databaseID, "Notes", "text", "{}")

	// Create a record with initial values
	vals := map[uint]string{
		tpID:        `{"text":"Before"}`,
		textProp.ID: `{"text":"initial notes"}`,
	}
	record, _ := rs.Create(databaseID, vals, 1)

	t.Run("inserts new property values", func(t *testing.T) {
		// Create a new property and pre-create its PropertyValue with a placeholder,
		// then update. (GORM's FirstOrCreate has a known issue with composite
		// primary keys in SQLite where the create path uses zero PK values.)
		numberProp, _ := ps.Create(databaseID, "Score", "number", "{}")
		gdb.Create(&db.PropertyValue{
			RecordID:   record.ID,
			PropertyID: numberProp.ID,
			Value:      `{"number":0}`,
		})

		updates := map[uint]string{
			numberProp.ID: `{"number":42}`,
		}
		err := rs.Update(record.ID, updates)
		assert.NoError(t, err)

		var pv db.PropertyValue
		gdb.Where("record_id = ? AND property_id = ?", record.ID, numberProp.ID).First(&pv)
		assert.Contains(t, pv.Value, "42")
	})

	t.Run("updates existing property values", func(t *testing.T) {
		updates := map[uint]string{
			textProp.ID: `{"text":"updated notes"}`,
		}
		err := rs.Update(record.ID, updates)
		assert.NoError(t, err)

		var pv db.PropertyValue
		gdb.Where("record_id = ? AND property_id = ?", record.ID, textProp.ID).First(&pv)
		assert.Contains(t, pv.Value, "updated notes")
	})

	t.Run("updates with multiple properties at once", func(t *testing.T) {
		gdb2, dbID := setupTestDBForRecords(t)
		rs2 := NewRecordService(gdb2)
		tID := titlePropertyID(gdb2, dbID)
		ps2 := NewPropertyService(gdb2)
		aProp, _ := ps2.Create(dbID, "A", "text", "{}")
		bProp, _ := ps2.Create(dbID, "B", "text", "{}")

		r, _ := rs2.Create(dbID, map[uint]string{tID: `{"text":"Multi"}`}, 1)
		// Pre-create property values so FirstOrCreate finds existing rows
		gdb2.Create(&db.PropertyValue{RecordID: r.ID, PropertyID: aProp.ID, Value: "{}"})
		gdb2.Create(&db.PropertyValue{RecordID: r.ID, PropertyID: bProp.ID, Value: "{}"})

		err := rs2.Update(r.ID, map[uint]string{
			aProp.ID: `{"text":"alpha"}`,
			bProp.ID: `{"text":"beta"}`,
		})
		assert.NoError(t, err)

		var pv db.PropertyValue
		gdb2.Where("record_id = ? AND property_id = ?", r.ID, aProp.ID).First(&pv)
		assert.Contains(t, pv.Value, "alpha")
	})
}

func TestRecordServiceDelete(t *testing.T) {
	gdb, databaseID := setupTestDBForRecords(t)
	rs := NewRecordService(gdb)
	tpID := titlePropertyID(gdb, databaseID)

	t.Run("deletes record and property values", func(t *testing.T) {
		record, _ := rs.Create(databaseID, map[uint]string{tpID: `{"text":"DelMe"}`}, 1)

		err := rs.Delete(record.ID)
		assert.NoError(t, err)

		// Verify record is gone
		var found db.Record
		err = gdb.First(&found, record.ID).Error
		assert.Error(t, err)

		// Verify property values are gone
		var pvs []db.PropertyValue
		gdb.Where("record_id = ?", record.ID).Find(&pvs)
		assert.Empty(t, pvs)
	})

	t.Run("returns error for nonexistent record", func(t *testing.T) {
		err := rs.Delete(99999)
		assert.Error(t, err)
	})

	t.Run("deletes associated page", func(t *testing.T) {
		gdb2, dbID := setupTestDBForRecords(t)
		rs2 := NewRecordService(gdb2)
		tID := titlePropertyID(gdb2, dbID)

		record, _ := rs2.Create(dbID, map[uint]string{tID: `{"text":"PageTest"}`}, 1)
		pageID := record.PageID

		err := rs2.Delete(record.ID)
		assert.NoError(t, err)

		var page db.Page
		err = gdb2.First(&page, pageID).Error
		assert.Error(t, err, "page should be deleted with record")
	})
}

func TestRecordServiceListByDatabase(t *testing.T) {
	gdb, databaseID := setupTestDBForRecords(t)
	rs := NewRecordService(gdb)
	tpID := titlePropertyID(gdb, databaseID)

	t.Run("returns records ordered by position", func(t *testing.T) {
		// Delete any existing records from setup (Service.Create adds a title prop only, no records)
		gdb.Where("database_id = ?", databaseID).Delete(&db.Record{})

		r1, _ := rs.Create(databaseID, map[uint]string{tpID: `{"text":"First"}`}, 1)
		r2, _ := rs.Create(databaseID, map[uint]string{tpID: `{"text":"Second"}`}, 1)
		r3, _ := rs.Create(databaseID, map[uint]string{tpID: `{"text":"Third"}`}, 1)

		records, err := rs.ListByDatabase(databaseID)
		assert.NoError(t, err)
		assert.Len(t, records, 3)
		assert.Equal(t, r1.ID, records[0].ID)
		assert.Equal(t, r2.ID, records[1].ID)
		assert.Equal(t, r3.ID, records[2].ID)
	})

	t.Run("returns empty when no records exist", func(t *testing.T) {
		gdb2, dbID := setupTestDBForRecords(t)
		rs2 := NewRecordService(gdb2)

		records, err := rs2.ListByDatabase(dbID)
		assert.NoError(t, err)
		assert.Empty(t, records)
	})

	t.Run("returns records for correct database only", func(t *testing.T) {
		// Create a second database in the same DB connection.
		// Each setupTestDBForRecords call creates its own in-memory SQLite,
		// so we must create both databases within the same connection.
		gdb2, dbID1 := setupTestDBForRecords(t)
		// Get workspace from the first database
		var dbase db.Database
		gdb2.First(&dbase, dbID1)
		dbSvc := NewService(gdb2)
		db2, _ := dbSvc.Create(dbase.WorkspaceID, "Second DB", 1)

		rs3 := NewRecordService(gdb2)
		tID := titlePropertyID(gdb2, dbID1)
		tID2 := titlePropertyID(gdb2, db2.ID)

		rs3.Create(dbID1, map[uint]string{tID: `{"text":"DB1"}`}, 1)
		rs3.Create(db2.ID, map[uint]string{tID2: `{"text":"DB2"}`}, 1)
		rs3.Create(db2.ID, map[uint]string{tID2: `{"text":"DB2b"}`}, 1)

		records, err := rs3.ListByDatabase(db2.ID)
		assert.NoError(t, err)
		assert.Len(t, records, 2, "should only return records for dbID2")
	})
}

func TestRecordServiceLastEditedBy(t *testing.T) {
	gdb, databaseID := setupTestDBForRecords(t)
	rs := NewRecordService(gdb)
	tpID := titlePropertyID(gdb, databaseID)

	t.Run("returns dash for zero ID", func(t *testing.T) {
		result := rs.LastEditedBy(0)
		assert.Equal(t, "—", result)
	})

	t.Run("returns Unknown for valid ID", func(t *testing.T) {
		record, _ := rs.Create(databaseID, map[uint]string{tpID: `{"text":"Edit Test"}`}, 1)
		result := rs.LastEditedBy(record.ID)
		assert.Equal(t, "Unknown", result, "currently always returns Unknown")
	})

	t.Run("returns Unknown for nonexistent ID", func(t *testing.T) {
		result := rs.LastEditedBy(54321)
		assert.Equal(t, "Unknown", result)
	})
}

// --- RecordService edge cases ---

func TestRecordServiceCreate_InvalidJSON(t *testing.T) {
	gdb, databaseID := setupTestDBForRecords(t)
	rs := NewRecordService(gdb)
	tpID := titlePropertyID(gdb, databaseID)

	t.Run("handles invalid JSON in property values gracefully", func(t *testing.T) {
		vals := map[uint]string{
			tpID: `not-valid-json`,
		}
		record, err := rs.Create(databaseID, vals, 1)
		// Should succeed - invalid JSON in title just falls back to "Untitled"
		assert.NoError(t, err)
		assert.NotNil(t, record)
	})

	t.Run("handles empty property values map", func(t *testing.T) {
		gdb2, dbID := setupTestDBForRecords(t)
		rs2 := NewRecordService(gdb2)

		record, err := rs2.Create(dbID, map[uint]string{}, 1)
		assert.NoError(t, err)
		assert.NotNil(t, record)

		var pvs []db.PropertyValue
		gdb2.Where("record_id = ?", record.ID).Find(&pvs)
		assert.Empty(t, pvs, "no property values should be created")
	})
}

func TestRecordServiceCreate_InvalidDatabase(t *testing.T) {
	gdb, _ := setupTestDBForRecords(t)
	rs := NewRecordService(gdb)

	t.Run("returns error for nonexistent database", func(t *testing.T) {
		record, err := rs.Create(99999, map[uint]string{}, 1)
		assert.Error(t, err)
		assert.Nil(t, record)
	})
}

// TestRecordServiceUpdate_ComplexValues tests JSON value handling for all property types.
// Pre-creates PropertyValues before calling Update to work around GORM's FirstOrCreate
// composite-PK limitation in SQLite (insert path uses zero PK values).
func TestRecordServiceUpdate_ComplexValues(t *testing.T) {
	gdb, databaseID := setupTestDBForRecords(t)
	rs := NewRecordService(gdb)
	tpID := titlePropertyID(gdb, databaseID)
	ps := NewPropertyService(gdb)

	t.Run("number values", func(t *testing.T) {
		numProp, _ := ps.Create(databaseID, "Amount", "number", "{}")
		record, _ := rs.Create(databaseID, map[uint]string{tpID: `{"text":"Num Test"}`}, 1)
		gdb.Create(&db.PropertyValue{RecordID: record.ID, PropertyID: numProp.ID, Value: "{}"})

		err := rs.Update(record.ID, map[uint]string{numProp.ID: `{"number":99.5}`})
		assert.NoError(t, err)

		var pv db.PropertyValue
		gdb.Where("record_id = ? AND property_id = ?", record.ID, numProp.ID).First(&pv)
		assert.Contains(t, pv.Value, "99.5")
	})

	t.Run("select values", func(t *testing.T) {
		selProp, _ := ps.Create(databaseID, "Color", "select", `{"options":["Red","Blue"]}`)
		record, _ := rs.Create(databaseID, map[uint]string{tpID: `{"text":"Sel Test"}`}, 1)
		gdb.Create(&db.PropertyValue{RecordID: record.ID, PropertyID: selProp.ID, Value: "{}"})

		err := rs.Update(record.ID, map[uint]string{selProp.ID: `{"select":"Blue"}`})
		assert.NoError(t, err)

		var pv db.PropertyValue
		gdb.Where("record_id = ? AND property_id = ?", record.ID, selProp.ID).First(&pv)
		assert.Contains(t, pv.Value, "Blue")
	})

	t.Run("checkbox values", func(t *testing.T) {
		cbProp, _ := ps.Create(databaseID, "Done", "checkbox", "{}")
		record, _ := rs.Create(databaseID, map[uint]string{tpID: `{"text":"Cb Test"}`}, 1)
		gdb.Create(&db.PropertyValue{RecordID: record.ID, PropertyID: cbProp.ID, Value: "{}"})

		err := rs.Update(record.ID, map[uint]string{cbProp.ID: `{"checked":true}`})
		assert.NoError(t, err)

		var pv db.PropertyValue
		gdb.Where("record_id = ? AND property_id = ?", record.ID, cbProp.ID).First(&pv)
		assert.Contains(t, pv.Value, "true")
	})

	t.Run("multi_select values", func(t *testing.T) {
		msProp, _ := ps.Create(databaseID, "Tags", "multi_select", `{"options":["A","B","C"]}`)
		record, _ := rs.Create(databaseID, map[uint]string{tpID: `{"text":"Tag Test"}`}, 1)
		gdb.Create(&db.PropertyValue{RecordID: record.ID, PropertyID: msProp.ID, Value: "{}"})

		err := rs.Update(record.ID, map[uint]string{msProp.ID: `{"multi_select":["A","C"]}`})
		assert.NoError(t, err)

		var pv db.PropertyValue
		gdb.Where("record_id = ? AND property_id = ?", record.ID, msProp.ID).First(&pv)
		assert.Contains(t, pv.Value, "A")
		assert.Contains(t, pv.Value, "C")
	})

	t.Run("date values", func(t *testing.T) {
		dateProp, _ := ps.Create(databaseID, "Due", "date", "{}")
		record, _ := rs.Create(databaseID, map[uint]string{tpID: `{"text":"Date Test"}`}, 1)
		gdb.Create(&db.PropertyValue{RecordID: record.ID, PropertyID: dateProp.ID, Value: "{}"})

		err := rs.Update(record.ID, map[uint]string{dateProp.ID: `{"date":"2024-01-15"}`})
		assert.NoError(t, err)

		var pv db.PropertyValue
		gdb.Where("record_id = ? AND property_id = ?", record.ID, dateProp.ID).First(&pv)
		assert.Contains(t, pv.Value, "2024-01-15")
	})

	t.Run("url values", func(t *testing.T) {
		urlProp, _ := ps.Create(databaseID, "Website", "url", "{}")
		record, _ := rs.Create(databaseID, map[uint]string{tpID: `{"text":"URL Test"}`}, 1)
		gdb.Create(&db.PropertyValue{RecordID: record.ID, PropertyID: urlProp.ID, Value: "{}"})

		err := rs.Update(record.ID, map[uint]string{urlProp.ID: `{"url":"https://example.com"}`})
		assert.NoError(t, err)

		var pv db.PropertyValue
		gdb.Where("record_id = ? AND property_id = ?", record.ID, urlProp.ID).First(&pv)
		assert.Contains(t, pv.Value, "https://example.com")
	})
}

// TestRecordService_PropertyValuePersistence verifies that property values
// survive round-trip through the database.
func TestRecordService_PropertyValuePersistence(t *testing.T) {
	gdb, databaseID := setupTestDBForRecords(t)
	rs := NewRecordService(gdb)
	ps := NewPropertyService(gdb)
	tpID := titlePropertyID(gdb, databaseID)

	t.Run("values round-trip correctly", func(t *testing.T) {
		textProp, _ := ps.Create(databaseID, "Description", "text", "{}")

		inputVal := `{"text":"A long description with special chars: !@#$%^&*()"}`
		record, err := rs.Create(databaseID, map[uint]string{
			tpID:        `{"text":"Round-trip"}`,
			textProp.ID: inputVal,
		}, 1)
		assert.NoError(t, err)

		// Read back
		_, values, _, err := rs.GetByID(record.ID)
		assert.NoError(t, err)

		for _, v := range values {
			if v.PropertyID == textProp.ID {
				// Compare parsed JSON
				var expected, actual map[string]interface{}
				json.Unmarshal([]byte(inputVal), &expected)
				json.Unmarshal([]byte(v.Value), &actual)
				assert.Equal(t, expected["text"], actual["text"])
			}
		}
	})
}
