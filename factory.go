package database

import (
	"os"
	"sync"

	"github.com/gwaylib/conf/ini"
	"github.com/gwaylib/errors"
)

var (
	cacheLock = sync.Mutex{}
	cache     = map[string]*DB{}
)

func regCache(iniFileName, sectionName string, db *DB) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	key := iniFileName + sectionName
	cache[key] = db
}

func cacheDB(iniFileName, sectionName string) (*DB, error) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	key := iniFileName + sectionName

	// get from cache
	db, ok := cache[key]
	if ok {
		return db, nil
	}

	// create a new
	cfg, err := ini.GetFile(iniFileName)
	if err != nil {
		return nil, errors.As(err, iniFileName)
	}
	section, err := cfg.GetSection(sectionName)
	if err != nil {
		return nil, errors.As(err, sectionName)
	}
	drvName, err := section.GetKey("driver")
	if err != nil {
		return nil, errors.As(err, "not found 'driver'")
	}
	dsn, err := section.GetKey("dsn")
	if err != nil {
		return nil, errors.As(err, "not found 'dsn'")
	}
	db, err = Open(drvName.String(), os.ExpandEnv(dsn.String()))
	if err != nil {
		return nil, errors.As(err)
	}
	cache[key] = db
	return db, nil
}
