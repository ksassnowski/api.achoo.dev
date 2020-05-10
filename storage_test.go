package main

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v7"
	"github.com/google/go-cmp/cmp"
)

var (
	regionASubRegionA  = createPollenReport("region-a", "subregion-aa")
	regionASubRegionB  = createPollenReport("region-a", "subregion-ab")
	regionBSubRegionA  = createPollenReport("region-b", "subregion-ba")
	regionCNoSubregion = createPollenReport("region-c", "")
)

func newMiniRedisServer() *miniredis.Miniredis {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	return s
}

func newStorage(mr *miniredis.Miniredis) *RedisStorage {
	s := &RedisStorage{
		client: redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		}),
	}

	rs := []*PollenReport{
		regionASubRegionA,
		regionASubRegionB,
		regionBSubRegionA,
		regionCNoSubregion,
	}

	for _, r := range rs {
		if err := s.Save(r); err != nil {
			panic(err)
		}
	}

	return s
}

func createPollenReport(region, subregion string) *PollenReport {
	return &PollenReport{
		Region:    region,
		SubRegion: subregion,
		Pollen: []*pollen{
			{
				Name: "Roggen",
				Today: &pollenDayReport{
					Description: "mittlere Belastung",
					Severity:    "2",
				},
			},
		},
	}
}

func TestFetchByRegion(t *testing.T) {
	mr := newMiniRedisServer()
	defer mr.Close()
	s := newStorage(mr)

	testCases := []struct {
		description   string
		region        string
		expectedCount int
		want          []*PollenReport
	}{
		{
			"multiple subregions",
			"region-a",
			2,
			[]*PollenReport{regionASubRegionA, regionASubRegionB},
		},
		{
			"single subregion",
			"region-b",
			1,
			[]*PollenReport{regionBSubRegionA},
		},
		{
			"region with no subregion is its own subregion",
			"region-c",
			1,
			[]*PollenReport{regionCNoSubregion},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			got, err := s.GetByRegion(tc.region)
			if err != nil {
				t.Errorf("tried to fetch reports for region, got error instead: %q", err)
			}

			if len(got) != tc.expectedCount {
				t.Errorf("expected exactly %d results for region-a, got %d instead", tc.expectedCount, len(got))
			}

			if !cmp.Equal(got, tc.want) {
				t.Errorf("wanted %+v, got %+v", tc.want, got)
			}
		})
	}
}

func TestFetchBySubregion(t *testing.T) {
	mr := newMiniRedisServer()
	defer mr.Close()
	s := newStorage(mr)

	tests := []struct {
		description string
		subregion   string
		want        *PollenReport
		err         error
	}{
		{
			"existing region",
			"subregion-aa",
			regionASubRegionA,
			nil,
		},
		{
			"region without subregions can be queried as its own subregion",
			"region-c",
			regionCNoSubregion,
			nil,
		},
		{
			"non existent region",
			"subregion-ca",
			nil,
			ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			got, err := s.GetBySubregion(tc.subregion)
			if err != nil && err != tc.err {
				t.Errorf("wanted error %q, got %q", tc.err, err)
			} else {
				if !cmp.Equal(got, tc.want) {
					t.Errorf("wanted %+v, got %+v", tc.want, got)
				}
			}

		})
	}
}

func TestGetAllRegions(t *testing.T) {
	mr := newMiniRedisServer()
	defer mr.Close()
	s := newStorage(mr)

	// region names get normalized, so the dashes
	// should have been replaced by underscores
	want := []string{
		"region_a",
		"region_b",
		"region_c",
	}

	got, err := s.AllRegions()
	if err != nil {
		t.Errorf("got error %q", err)
	}

	if !cmp.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGetAllSubRegions(t *testing.T) {
	mr := newMiniRedisServer()
	defer mr.Close()
	s := newStorage(mr)

	want := []string{
		// regions without subregions get treated as
		// their own subregion
		"region_c",

		// subregion names get normalized, so the dashes
		// should have been replaced by underscores
		"subregion_aa",
		"subregion_ab",
		"subregion_ba",
	}

	got, err := s.AllSubregions()
	if err != nil {
		t.Errorf("got error %q", err)
	}

	if !cmp.Equal(got, want) {
		t.Errorf("got, %q, want %q", got, want)
	}
}

func TestErrorHandling(t *testing.T) {
	t.Run("key doesn't exist", func(t *testing.T) {
		s := newMiniRedisServer()
		defer s.Close()

		storage := newStorage(s)

		_, err := storage.GetBySubregion("::doesnt-exist::")
		if err == nil {
			t.Error("expected error, but got no error instead")
		}

		if err != ErrNotFound {
			t.Errorf("expected custom error, got %q instead", err.Error())
		}
	})

	t.Run("cannot connect to redis server", func(t *testing.T) {
		_, err := NewRedisStorage("99.99.99.99:6379", "", "", 1*time.Millisecond)
		if err == nil {
			t.Error("expected error, got nothing")
		} else if err != ErrCouldNotConnectToStorage {
			t.Errorf("got unexpected error: %q", err)
		}
	})
}
