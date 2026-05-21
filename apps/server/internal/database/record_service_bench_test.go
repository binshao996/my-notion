package database

import (
	"fmt"
	"testing"

	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRecordBenchDB(b *testing.B) (*gorm.DB, uint, uint) {
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}
	if err := gormDB.AutoMigrate(
		&db.Workspace{}, &db.Page{}, &db.Database{},
		&db.Property{}, &db.Record{}, &db.PropertyValue{}, &db.View{},
	); err != nil {
		b.Fatal(err)
	}

	ws := db.Workspace{Name: "bench-ws"}
	if err := gormDB.Create(&ws).Error; err != nil {
		b.Fatal(err)
	}

	svc := NewService(gormDB)
	dbase, err := svc.Create(ws.ID, "Bench DB", 1)
	if err != nil {
		b.Fatal(err)
	}

	var titleProp db.Property
	gormDB.Where("database_id = ? AND type = ?", dbase.ID, "title").First(&titleProp)

	return gormDB, dbase.ID, titleProp.ID
}

func BenchmarkCreateRecord(b *testing.B) {
	gormDB, databaseID, titlePID := setupRecordBenchDB(b)
	rs := NewRecordService(gormDB)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := rs.Create(databaseID, map[uint]string{
			titlePID: fmt.Sprintf(`{"text":"record-%d"}`, i),
		}, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListByDatabase_10kRecords(b *testing.B) {
	gormDB, databaseID, titlePID := setupRecordBenchDB(b)
	rs := NewRecordService(gormDB)

	// Seed 10000 records
	for i := 0; i < 10000; i++ {
		_, err := rs.Create(databaseID, map[uint]string{
			titlePID: fmt.Sprintf(`{"text":"r-%d"}`, i),
		}, 1)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := rs.ListByDatabase(databaseID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateRecord(b *testing.B) {
	gormDB, databaseID, titlePID := setupRecordBenchDB(b)
	rs := NewRecordService(gormDB)
	ps := NewPropertyService(gormDB)

	notesProp, err := ps.Create(databaseID, "Notes", "text", "{}")
	if err != nil {
		b.Fatal(err)
	}

	record, err := rs.Create(databaseID, map[uint]string{
		titlePID: `{"text":"update-target"}`,
	}, 1)
	if err != nil {
		b.Fatal(err)
	}
	gormDB.Create(&db.PropertyValue{RecordID: record.ID, PropertyID: notesProp.ID, Value: "{}"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := rs.Update(record.ID, map[uint]string{
			notesProp.ID: fmt.Sprintf(`{"text":"updated-%d"}`, i),
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
