package main

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/pkg/errors"
)

var (
	// ErrNotFound is returned if no data exists for a provided
	// Region and SubRegion.
	ErrNotFound = errors.New("storage: not found")
	// ErrCouldNotConnectToStorage is returned if we failed to
	// connect to the configured storage.
	ErrCouldNotConnectToStorage = errors.New("storage: unable to connect")

	keyRemoveRegexp  = regexp.MustCompile(`[.,]`)
	keyReplaceRegexp = regexp.MustCompile(`[/\s-]`)
)

// Storage defines a type that can save and retrieve
// storage.PollenReport instances
type Storage interface {
	Save(r *PollenReport) error
	AllRegions() ([]string, error)
	AllSubregions() ([]string, error)
	AllReports() ([]*PollenReport, error)
	GetByRegion(region string) ([]*PollenReport, error)
	GetBySubregion(subregion string) (*PollenReport, error)
}

// RedisStorage is a storage that reads and writes to a
// safe reads and writes.
type RedisStorage struct {
	client *redis.Client

	// prefix gets prepended to every key
	prefix string
}

// NewEnvStorage returns a storage configured via environment
// variables. If the required variables are not set, it attempts
// to use sensible defaults.
func NewEnvStorage() (Storage, error) {
	addr, exists := os.LookupEnv("REDIS_HOST")
	if !exists {
		addr = "localhost:6379"
	}
	prefix, exists := os.LookupEnv("REDIS_KEY_PREFIX")
	if !exists {
		prefix = ""
	}
	password, exists := os.LookupEnv("REDIS_PASSWORD")
	if !exists {
		password = ""
	}
	return NewRedisStorage(addr, password, prefix, 5*time.Second)
}

// NewRedisStorage creates a new storage which reads and writes
// to the redis server located at the provided addr.
func NewRedisStorage(addr, password, prefix string, dialTimeout time.Duration) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        addr,
		Password:    password,
		DB:          0,
		DialTimeout: dialTimeout,
	})

	_, err := client.Ping().Result()
	if err != nil {
		return nil, ErrCouldNotConnectToStorage
	}

	return &RedisStorage{
		client: client,
		prefix: prefix,
	}, nil
}

// Save attempts to marshall the provided PollenReport to json
// and write it to the redis database. It uses the Region and
// SubRegion keys to create the redis key.
func (rs *RedisStorage) Save(r *PollenReport) error {
	json, err := json.Marshal(r)
	if err != nil {
		log.Printf("[storage] unable to marshal pollen report: %q", err.Error())
		return err
	}

	normalizedRegion := normalizeString(r.Region)
	normalizedSubregion := normalizeString(r.SubRegion)

	key := normalizedSubregion
	// Not all regions have sub regions. In this case, we want to
	// use the region name as the key instead
	if key == "" {
		key = normalizedRegion
	}
	key = rs.makeKey("report:" + key)
	rs.client.Set(key, json, 0)

	// Add the region to the reports set so we can later fetch
	// all supported regions.
	rs.client.SAdd(rs.makeKey("regions"), normalizedRegion)

	// Add the subregion to the subregion set so we can later
	// provide the user with human readable names of all subregions
	// for which reports exist.
	//
	// Not all regions have subregions. In these cases we will use
	// the region name instead so you can still query a singular
	// result instead of having to fetch all reports for the region
	// which would always result in an array of length 1. And that
	// is annoying to deal with.
	if r.SubRegion != "" {
		rs.client.SAdd(rs.makeKey("subregions"), normalizedSubregion)
	} else {
		rs.client.SAdd(rs.makeKey("subregions"), normalizedRegion)
	}

	// Tag this report with this region so we can easily look up
	// all reports for a region
	rs.client.SAdd(rs.makeKey("region:"+normalizedRegion+":reports"), key)

	// Add the key to the list of all reports so we can easily fetch
	// all reports
	rs.client.SAdd(rs.makeKey("reports"), key)

	return nil
}

// AllReports returns all reports
func (rs *RedisStorage) AllReports() ([]*PollenReport, error) {
	keys, err := rs.client.SMembers(rs.makeKey("reports")).Result()
	if err != nil {
		return nil, err
	}

	vals, err := rs.client.MGet(keys...).Result()
	if err != nil {
		log.Printf("[storage] couldn't fetch reports: %s", err)
		return nil, err
	}

	reports := make([]*PollenReport, len(keys))
	for i := 0; i < len(keys); i++ {
		pr, err := parseReport(vals[i])
		if err != nil {
			return nil, err
		}
		reports[i] = pr
	}

	return reports, nil
}

// GetBySubregion loads a PollenReport entry from the redis
// database identified by its SubRegion. If no results
// exists, it returns ErrNotFound
func (rs *RedisStorage) GetBySubregion(subregion string) (*PollenReport, error) {
	key := rs.makeKey("report:" + subregion)

	strValue, err := rs.client.Get(key).Result()

	if err != nil {
		if err == redis.Nil {
			log.Printf("[storage] unable to find report for key %q", key)
			return nil, ErrNotFound
		}
		log.Printf("[storage] unable to fetch data from redis: %q", err.Error())
		return nil, err
	}

	var pr PollenReport
	if err := json.Unmarshal([]byte(strValue), &pr); err != nil {
		log.Printf("[storage] unable to unmarshal data: %q", err.Error())
		return nil, err
	}

	return &pr, nil
}

// GetByRegion returns the pollen reports of all subregions
// of the provided region.
func (rs *RedisStorage) GetByRegion(region string) ([]*PollenReport, error) {
	reportKeys, err := rs.client.SMembers(rs.makeKey("region:" + normalizeString(region) + ":reports")).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrNotFound
		}
		log.Printf("[storage] unable to fetch data from redis: %q", err.Error())
		return nil, err
	}

	reportVals, err := rs.client.MGet(reportKeys...).Result()
	if err != nil {
		log.Printf("[storage] couldn't fetch reports: %q", err)
		return nil, err
	}

	reports := make([]*PollenReport, len(reportKeys))
	for i := 0; i < len(reportVals); i++ {
		r, err := parseReport(reportVals[i])
		if err != nil {
			return nil, err
		}
		reports[i] = r
	}

	return reports, nil
}

// AllRegions returns a list of all regions for which
// PollenResults exist
func (rs *RedisStorage) AllRegions() ([]string, error) {
	return rs.client.SMembers(rs.makeKey("regions")).Result()
}

// AllSubregions returns a human readable list of all subregions
// for which PollenResults exist
func (rs *RedisStorage) AllSubregions() ([]string, error) {
	return rs.client.SMembers(rs.makeKey("subregions")).Result()
}

func (rs *RedisStorage) makeKey(key string) string {
	key = normalizeString(key)
	if rs.prefix == "" {
		return key
	}
	return rs.prefix + ":" + key
}

func normalizeString(s string) string {
	s = keyRemoveRegexp.ReplaceAllLiteralString(s, "")
	s = keyReplaceRegexp.ReplaceAllLiteralString(s, "_")
	return s
}

func parseReport(report interface{}) (*PollenReport, error) {
	s, ok := report.(string)
	if !ok {
		return nil, errors.New("[storage] couldn't cast report to string. this should never happen")
	}

	var r PollenReport
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return nil, err
	}

	return &r, nil
}
