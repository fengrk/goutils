package dnsutils

import (
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/frkhit/logger"
	"sync"
	"time"
)

const rrBucket = "rr"

type DBCache interface {
	Get(string) (string, error)
	Set(string, string) error
	Delete(string) error
	Clear() error
	BatchDelete([]string) (error)
	BatchSet(map[string]string) (error)
	Close()
}

type MemCache struct {
	record sync.Map
}

func NewMemCache(dbPath string) DBCache {
	return &MemCache{record: sync.Map{}}
}

func (db *MemCache) Get(key string) (string, error) {
	value, exists := db.record.Load(key)
	if exists {
		return value.(string), nil
	} else {
		return "", fmt.Errorf("record not found: %s", key)
	}
}

func (db *MemCache) Set(key string, value string) (error) {
	db.record.Store(key, value)
	return nil
}

func (db *MemCache) Delete(key string) (error) {
	db.record.Delete(key)
	return nil
}

func (db *MemCache) BatchDelete(keyList []string) (error) {
	for _, key := range keyList {
		db.record.Delete(key)
	}
	return nil
}

func (db *MemCache) BatchSet(record map[string]string) (error) {
	for key, value := range record {
		db.record.Store(key, value)
	}
	return nil
}

func (db *MemCache) Clear() (error) {
	db.record.Range(func(key interface{}, value interface{}) bool {
		db.record.Delete(key)
		return true
	})
	return nil
}
func (db *MemCache) Close() {
	db.Clear()
}

// todo page error when running in WSL
type BoltDBCache struct {
	bdb *bolt.DB
}

func (db *BoltDBCache) createBucket(bucket string) (error) {
	return db.bdb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			e := errors.New("Create bucket: " + bucket)
			logger.Infoln(e.Error())
			return e
		}
		return nil
	})
}

func NewBoltDBCache(dbPath string) DBCache {
	bdb, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		logger.Fatal("Failed to open dbPath[%s]: %s\n", dbPath, err.Error())
	}
	cache := &BoltDBCache{bdb: bdb}
	
	// Create dns bucket if doesn't exist
	err = cache.createBucket(rrBucket)
	if err != nil {
		logger.Fatalf("Failed to create bucket: %sn ", err.Error())
	}
	
	return cache
}

func (db *BoltDBCache) Get(key string) (string, error) {
	var v []byte
	err := db.bdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(rrBucket))
		v = b.Get([]byte(key))
		
		if string(v) == "" {
			return errors.New("Record not found, key: " + key)
		}
		return nil
	})
	if err == nil {
		return string(v), nil
	}
	return "", err
}

func (db *BoltDBCache) Set(key string, value string) (error) {
	return db.bdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(rrBucket))
		return b.Put([]byte(key), []byte(value))
	})
}

func (db *BoltDBCache) Delete(key string) (error) {
	return db.bdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(rrBucket))
		e := b.Delete([]byte(key))
		if e != nil {
			return e
		}
		return nil
	})
}

func (db *BoltDBCache) BatchDelete(keyList []string) (error) {
	//const size = 1
	//// buffered so we never leak goroutines
	//ch := make(chan error, size)
	//put := func(i int) {
	//	ch <- db.Batch(func(tx *bolt.Tx) error {
	//		return tx.Bucket([]byte("widgets")).Put(u64tob(uint64(i)), []byte{})
	//	})
	//}
	//
	//db.MaxBatchSize = 1000
	//db.MaxBatchDelay = 0
	//
	//go put(1)
	//
	//// Batch must trigger by time alone.
	//
	//// Check all responses to make sure there's no error.
	//for i := 0; i < size; i++ {
	//	if err := <-ch; err != nil {
	//		t.Fatal(err)
	//	}
	//}
	return nil
}

func (db *BoltDBCache) BatchSet(record map[string]string) (error) {
	// todo not profile
	for key, value := range record {
		if err := db.Set(key, value); err != nil {
			return err
		}
	}
	return nil
}

func (db *BoltDBCache) Clear() (error) {
	if delErr := db.bdb.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(rrBucket))
		if err != nil {
			return fmt.Errorf("delete bucket: %s, error: %s", rrBucket, err)
		}
		return nil
	}); delErr != nil {
		return delErr
	}
	return db.createBucket(rrBucket)
}
func (db *BoltDBCache) Close() {
	db.bdb.Close()
}
