package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	dataURL = "https://opendata.dwd.de/climate_environment/health/alerts/s31fg.json"
)

// Syncer represents a type responsible for updating
// the local storage with fresh pollen data.
type Syncer struct {
	storage  Storage
	interval time.Duration
	url      string
}

// NewSyncer returns a new syncer configured to fetch
// data from the opendata server.
func NewSyncer(s Storage, interval time.Duration) *Syncer {
	return &Syncer{
		storage:  s,
		interval: interval,
		url:      dataURL,
	}
}

var severityMap = map[string]string{
	"0":   "keine Belastung",
	"0-1": "keine bis geringe Belastung",
	"1":   "geringe Belastung",
	"1-2": "geringe bis mittlere Belastung",
	"2":   "mittlere Belastung",
	"2-3": "mittlere bis hohe Belastung",
	"3":   "hohe Belastung",
}

type openDataPollenResponse struct {
	NextUpdate string                    `json:"next_update"`
	Name       string                    `json:"name"`
	Sender     string                    `json:"sender"`
	Legend     *openDataLegend           `json:"legend"`
	Content    []*openDataLocationReport `json:"content"`
	LastUpdate string                    `json:"last_update"`
}

type openDataLegend struct {
	ID1     string `json:"id1"`
	ID1Desc string `json:"id1_desc"`
	ID2     string `json:"id2"`
	ID2Desc string `json:"id2_desc"`
	ID3     string `json:"id3"`
	ID3Desc string `json:"id3_desc"`
	ID4     string `json:"id4"`
	ID4Desc string `json:"id4_desc"`
	ID5     string `json:"id5"`
	ID5Desc string `json:"id5_desc"`
	ID6     string `json:"id6"`
	ID6Desc string `json:"id6_desc"`
	ID7     string `json:"id7"`
	ID7Desc string `json:"id7_desc"`
}

type openDataLocationReport struct {
	RegionID       int                   `json:"region_id"`
	RegionName     string                `json:"region_name"`
	PartRegionID   int                   `json:"partregion_id"`
	PartregionName string                `json:"partregion_name"`
	Pollen         *openDataPollenReport `json:"Pollen"`
}

type openDataPollenReport struct {
	Ambrosia *openDataSinglePollenReport `json:"Ambrosia"`
	Beifuss  *openDataSinglePollenReport `json:"Beifuss"`
	Birke    *openDataSinglePollenReport `json:"Birke"`
	Erle     *openDataSinglePollenReport `json:"Erle"`
	Esche    *openDataSinglePollenReport `json:"Esche"`
	Graeser  *openDataSinglePollenReport `json:"Graeser"`
	Hasel    *openDataSinglePollenReport `json:"Hasel"`
	Roggen   *openDataSinglePollenReport `json:"Roggen"`
}

type openDataSinglePollenReport struct {
	Tomorrow         string `json:"tomorrow"`
	Today            string `json:"today"`
	DayAfterTomorrow string `json:"dayafter_to"`
}

// Run starts the syncer daemon. It will fetch new data
// in the configured interval and save it to the storage.
func (s *Syncer) Run() {
	log.Printf("[sync] starting sync daemon…")
	done := make(chan struct{})

	go s.sync(done)

	for {
		<-done
		log.Printf("[sync] finished syncing…")
		time.Sleep(s.interval)
		go s.sync(done)
	}
}

func (s *Syncer) sync(done chan struct{}) {
	log.Printf("[sync] Starting sync run…")

	resp, err := http.Get(s.url)
	if err != nil {
		log.Printf("[sync] unable to fetch data: %q", err.Error())
		return
	}

	defer resp.Body.Close()

	var data openDataPollenResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("[sync] unable decode response: %q", err.Error())
		return
	}

	mapped := mapResponse(&data)

	for _, r := range mapped {
		s.storage.Save(r)
	}

	done <- struct{}{}
}

// PollenReport is the internal representation of the open data
// polen report with a slightly more sane structure.
type PollenReport struct {
	Region    string    `json:"region"`
	SubRegion string    `json:"sub_region"`
	Pollen    []*pollen `json:"pollen"`
}

type pollen struct {
	Name             string           `json:"name"`
	Today            *pollenDayReport `json:"today"`
	Tomorrow         *pollenDayReport `json:"tomorrow"`
	DayAfterTomorrow *pollenDayReport `json:"day_after_tomorrow"`
}

type pollenDayReport struct {
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

func mapResponse(r *openDataPollenResponse) []*PollenReport {
	var result []*PollenReport

	for _, lr := range r.Content {
		r := &PollenReport{
			strings.TrimSpace(lr.RegionName),
			strings.TrimSpace(lr.PartregionName),
			mapLocationReport(lr.Pollen),
		}

		result = append(result, r)
	}

	return result
}

func mapLocationReport(r *openDataPollenReport) []*pollen {
	var result []*pollen

	result = append(result, mapPollenReport("Ambrosia", r.Ambrosia))
	result = append(result, mapPollenReport("Beifuss", r.Beifuss))
	result = append(result, mapPollenReport("Birke", r.Birke))
	result = append(result, mapPollenReport("Erle", r.Erle))
	result = append(result, mapPollenReport("Esche", r.Esche))
	result = append(result, mapPollenReport("Gräser", r.Graeser))
	result = append(result, mapPollenReport("Hasel", r.Hasel))
	result = append(result, mapPollenReport("Roggen", r.Roggen))

	return result
}

func mapPollenReport(name string, r *openDataSinglePollenReport) *pollen {
	todayDesc, _ := severityMap[r.Today]
	tomorrowDesc, _ := severityMap[r.Tomorrow]
	dayAfterTomorrowDesc, _ := severityMap[r.DayAfterTomorrow]

	p := &pollen{
		name,
		&pollenDayReport{r.Today, todayDesc},
		&pollenDayReport{r.Tomorrow, tomorrowDesc},
		&pollenDayReport{r.DayAfterTomorrow, dayAfterTomorrowDesc},
	}

	return p
}
