package store

import (
	"errors"
	"os"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type StoreType string

const (
	StoreTypeSqlite   StoreType = "sqlite"
	StoreTypePostgres StoreType = "postgres"
)

type Store interface {
	// Init initializes the store with the optional config
	Init(config map[string]any) error
	// Get retrieves the value for the given key.
	// If the key does not exist, ErrNotFound is returned.
	Get(key string) ([]byte, error)
	// Set stores the value for the given key.
	Set(key string, value []byte) error
	// Delete deletes the value for the given key.
	Delete(key string) error
	// DB returns the underlying database object.
	DB() *gorm.DB
	// Close closes the underlying database connection.
	Close() error
}

type KeyValue struct {
	Key   string `gorm:"primaryKey"`
	Value []byte
}

func NewStore(storeType StoreType) Store {
	switch storeType {
	case StoreTypeSqlite:
		return &SqliteStore{}
	case StoreTypePostgres:
		return &PostgresStore{}
	default:
		return nil
	}
}

func Init(storeType StoreType, config map[string]any) (Store, error) {
	l := log.WithFields(log.Fields{
		"store": storeType,
	})
	l.Debug("initializing store")
	if storeType == "" {
		storeType = StoreTypeSqlite
	}
	s := NewStore(storeType)
	if s == nil {
		return nil, errors.New("store not initialized")
	}
	for k, v := range config {
		// envsubst the value
		if s, ok := v.(string); ok {
			v = os.ExpandEnv(s)
			config[k] = v
		}
	}
	if err := s.Init(config); err != nil {
		return nil, err
	}
	if err := s.DB().AutoMigrate(&KeyValue{}); err != nil {
		return nil, err
	}
	return s, nil
}
