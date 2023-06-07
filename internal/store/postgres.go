package store

import (
	"encoding/json"
	"errors"
	"strconv"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type PostgresStore struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	User    string `json:"user"`
	Pass    string `json:"pass"`
	Dbname  string `json:"dbname"`
	Sslmode string `json:"sslmode"`

	db *gorm.DB
}

func (s *PostgresStore) Init(config map[string]interface{}) error {
	jd, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(jd, s); err != nil {
		return err
	}
	if s.Host == "" {
		return errors.New("missing host")
	}
	if s.Port == 0 {
		s.Port = 5432
	}
	if s.User == "" {
		return errors.New("missing user")
	}
	if s.Dbname == "" {
		return errors.New("missing dbname")
	}
	dsn := "host=" + s.Host + " port=" + strconv.Itoa(s.Port) + " user=" + s.User + " dbname=" + s.Dbname
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
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

func (s *PostgresStore) DB() *gorm.DB {
	return s.db
}

func (s *PostgresStore) Get(key string) ([]byte, error) {
	var kv KeyValue
	if err := s.db.First(&kv, "key = ?", key).Error; err != nil {
		return nil, err
	}
	return kv.Value, nil
}

func (s *PostgresStore) Set(key string, value []byte) error {
	return s.db.Save(&KeyValue{
		Key:   key,
		Value: value,
	}).Error
}

func (s *PostgresStore) Delete(key string) error {
	return s.db.Delete(&KeyValue{}, "key = ?", key).Error
}

func (s *PostgresStore) Close() error {
	d, err := s.db.DB()
	if err != nil {
		return err
	}
	return d.Close()
}
