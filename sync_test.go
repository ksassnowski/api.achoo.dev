package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type inMemoryStorage struct {
	data []*PollenReport
}

func (s *inMemoryStorage) Save(r *PollenReport) error {
	s.data = append(s.data, r)
	return nil
}

func (s *inMemoryStorage) GetByRegion(region string) ([]*PollenReport, error) {
	return nil, nil
}

func (s *inMemoryStorage) GetBySubregion(subregion string) (*PollenReport, error) {
	return nil, nil
}

func (s *inMemoryStorage) AllReports() ([]*PollenReport, error) {
	return s.data, nil
}

func (s *inMemoryStorage) AllRegions() ([]string, error) {
	return nil, nil
}

func (s *inMemoryStorage) AllSubregions() ([]string, error) {
	return nil, nil
}

var upstreamResponse = &openDataPollenResponse{
	Name:       "::name::",
	NextUpdate: "2020-01-01 11:00 Uhr",
	Content: []*openDataLocationReport{
		{
			RegionID:       123,
			RegionName:     "::region-a::",
			PartRegionID:   234,
			PartregionName: "::region-a-subregion-a::",
			Pollen: &openDataPollenReport{
				Ambrosia: &openDataSinglePollenReport{
					Tomorrow:         "0",
					Today:            "0-1",
					DayAfterTomorrow: "1-2",
				},
				Beifuss: &openDataSinglePollenReport{
					Tomorrow:         "1",
					Today:            "1-2",
					DayAfterTomorrow: "1-2",
				},
				Birke: &openDataSinglePollenReport{
					Tomorrow:         "2",
					Today:            "1",
					DayAfterTomorrow: "2-3",
				},
				Erle: &openDataSinglePollenReport{
					Tomorrow:         "1",
					Today:            "0",
					DayAfterTomorrow: "1-2",
				},
				Esche: &openDataSinglePollenReport{
					Tomorrow:         "2",
					Today:            "0",
					DayAfterTomorrow: "2-3",
				},
				Graeser: &openDataSinglePollenReport{
					Tomorrow:         "2",
					Today:            "0",
					DayAfterTomorrow: "2-3",
				},
				Hasel: &openDataSinglePollenReport{
					Tomorrow:         "2",
					Today:            "1-2",
					DayAfterTomorrow: "1",
				},
				Roggen: &openDataSinglePollenReport{
					Tomorrow:         "2",
					Today:            "0",
					DayAfterTomorrow: "2-3",
				},
			},
		},
	},
}

func TestSync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json, _ := json.Marshal(upstreamResponse)
		w.Write(json)
	}))
	defer server.Close()

	done := make(chan struct{})
	syncer := &Syncer{
		url:     server.URL,
		storage: &inMemoryStorage{},
	}
	go syncer.sync(done)

	want := []*PollenReport{
		{
			Region:    "::region-a::",
			SubRegion: "::region-a-subregion-a::",
			Pollen: []*pollen{
				{
					Name: "Ambrosia",
					Today: &pollenDayReport{
						Severity:    "0-1",
						Description: "keine bis geringe Belastung",
					},
					Tomorrow: &pollenDayReport{
						Severity:    "0",
						Description: "keine Belastung",
					},
					DayAfterTomorrow: &pollenDayReport{
						Severity:    "1-2",
						Description: "geringe bis mittlere Belastung",
					},
				},
				{
					Name: "Beifuss",
					Today: &pollenDayReport{
						Severity:    "1-2",
						Description: "geringe bis mittlere Belastung",
					},
					Tomorrow: &pollenDayReport{
						Severity:    "1",
						Description: "geringe Belastung",
					},
					DayAfterTomorrow: &pollenDayReport{
						Severity:    "1-2",
						Description: "geringe bis mittlere Belastung",
					},
				},
				{
					Name: "Birke",
					Today: &pollenDayReport{
						Severity:    "1",
						Description: "geringe Belastung",
					},
					Tomorrow: &pollenDayReport{
						Severity:    "2",
						Description: "mittlere Belastung",
					},
					DayAfterTomorrow: &pollenDayReport{
						Severity:    "2-3",
						Description: "mittlere bis hohe Belastung",
					},
				},
				{
					Name: "Erle",
					Today: &pollenDayReport{
						Severity:    "0",
						Description: "keine Belastung",
					},
					Tomorrow: &pollenDayReport{
						Severity:    "1",
						Description: "geringe Belastung",
					},
					DayAfterTomorrow: &pollenDayReport{
						Severity:    "1-2",
						Description: "geringe bis mittlere Belastung",
					},
				},
				{
					Name: "Esche",
					Today: &pollenDayReport{
						Severity:    "0",
						Description: "keine Belastung",
					},
					Tomorrow: &pollenDayReport{
						Severity:    "2",
						Description: "mittlere Belastung",
					},
					DayAfterTomorrow: &pollenDayReport{
						Severity:    "2-3",
						Description: "mittlere bis hohe Belastung",
					},
				},
				{
					Name: "Gräser",
					Today: &pollenDayReport{
						Severity:    "0",
						Description: "keine Belastung",
					},
					Tomorrow: &pollenDayReport{
						Severity:    "2",
						Description: "mittlere Belastung",
					},
					DayAfterTomorrow: &pollenDayReport{
						Severity:    "2-3",
						Description: "mittlere bis hohe Belastung",
					},
				},
				{
					Name: "Hasel",
					Today: &pollenDayReport{
						Severity:    "1-2",
						Description: "geringe bis mittlere Belastung",
					},
					Tomorrow: &pollenDayReport{
						Severity:    "2",
						Description: "mittlere Belastung",
					},
					DayAfterTomorrow: &pollenDayReport{
						Severity:    "1",
						Description: "geringe Belastung",
					},
				},
				{
					Name: "Roggen",
					Today: &pollenDayReport{
						Severity:    "0",
						Description: "keine Belastung",
					},
					Tomorrow: &pollenDayReport{
						Severity:    "2",
						Description: "mittlere Belastung",
					},
					DayAfterTomorrow: &pollenDayReport{
						Severity:    "2-3",
						Description: "mittlere bis hohe Belastung",
					},
				},
			},
		},
	}

	<-done

	got, _ := syncer.storage.AllReports()
	diff := cmp.Diff(got, want)
	if diff != "" {
		t.Error(diff)
	}
}
