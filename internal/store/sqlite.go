package store

import (
	"encoding/json"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SqliteStore struct {
	// Path to the sqlite database file
	Path string `json:"path"`

	db *gorm.DB
}

func (s *SqliteStore) Init(config map[string]interface{}) error {
	jd, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(jd, s); err != nil {
		return err
	}
	if s.Path == "" {
		s.Path = "localdns.db"
	}
	db, err := gorm.Open(sqlite.Open(s.Path), &gorm.Config{})
	if err != nil {
		return err
	}
	s.db = db
	// ping the database to make sure it's alive
	d, err := db.DB()
	if err != nil {
		return err
	}
	return d.Ping()
}

func (s *SqliteStore) DB() *gorm.DB {
	return s.db
}

func (s *SqliteStore) Get(key string) ([]byte, error) {
	var kv KeyValue
	if err := s.db.First(&kv, "key = ?", key).Error; err != nil {
		return nil, err
	}
	return kv.Value, nil
}

func (s *SqliteStore) Set(key string, value []byte) error {
	return s.db.Save(&KeyValue{
		Key:   key,
		Value: value,
	}).Error
}

func (s *SqliteStore) Delete(key string) error {
	return s.db.Delete(&KeyValue{}, "key = ?", key).Error
}

func (s *SqliteStore) Close() error {
	d, err := s.db.DB()
	if err != nil {
		return err
	}
	return d.Close()
}
