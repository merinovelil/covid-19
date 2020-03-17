package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/montanaflynn/gountries"
)

var (
	baseURL  = "https://interaktiv.morgenpost.de"
	endpoint = "/corona-virus-karte-infektionen-deutschland-weltweit/data/Coronavirus.current.v2.csv"
)

type datum struct {
	parent    string
	label     string
	updated   int
	date      time.Time
	confirmed int
	recovered int
	deaths    int
	lon       float64
	lat       float64
	source    string
	sourceURL string
	scraper   string
}

type cases struct {
	Updated   int `json:"updated"`
	Confirmed int `json:"confirmed"`
	Recovered int `json:"recovered"`
	Deaths    int `json:"deaths"`
}

type results struct {
	Global  map[string]cases `json:"global"`
	USA     map[string]cases `json:"usa"`
	Canada  map[string]cases `json:"canada"`
	Germany map[string]cases `json:"germany"`
	China   map[string]cases `json:"china"`
}

func main() {
	res, err := http.Get(fmt.Sprintf("%s%s", baseURL, endpoint))
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	r := csv.NewReader(res.Body)

	var data []datum

	// skip the headers
	_, err = r.Read()
	if err == io.EOF {
		log.Fatal("missing csv headers")
	}

	// Iterate through the records
	for {
		// Read each record from csv
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		// fmt.Printf("%+v\n", record)
		updated, err := strconv.Atoi(record[2])
		if err != nil {
			log.Fatal(err)
		}

		date := time.Unix(int64(updated), 0)

		confirmed, err := strconv.Atoi(record[4])
		if err != nil {
			log.Fatal(err)
		}
		recovered, err := strconv.Atoi(record[5])
		if err != nil {
			log.Fatal(err)
		}
		deaths, err := strconv.Atoi(record[6])
		if err != nil {
			log.Fatal(err)
		}

		d := datum{
			parent:    record[0],
			label:     record[1],
			updated:   updated,
			date:      date,
			confirmed: confirmed,
			recovered: recovered,
			deaths:    deaths,
			source:    record[9],
			sourceURL: record[10],
			scraper:   record[11],
		}

		if record[7] != "null" {
			lon, err := strconv.ParseFloat(record[7], 64)
			if err != nil {
				log.Fatal(err)
			}
			d.lon = lon
		}

		if record[8] != "null" {
			lat, err := strconv.ParseFloat(record[8], 64)
			if err != nil {
				log.Fatal(err)
			}
			d.lat = lat
		}

		// fmt.Printf("%+v", d)

		data = append(data, d)
	}

	countryCounts := make(map[string]cases)
	usaCounts := make(map[string]cases)
	chinaCounts := make(map[string]cases)
	germanyCounts := make(map[string]cases)
	canadaCounts := make(map[string]cases)

	query := gountries.New()

	for _, d := range data {

		countryName := d.parent

		country, err := query.FindCountryByAlpha(countries[d.parent])
		if err == nil {
			countryName = country.Name.Common
		}

		if d.parent == "global" {
			countryName = d.label

			country, err := query.FindCountryByAlpha(countries[d.label])
			if err == nil {
				countryName = country.Name.Common
			}
		}

		c, ok := countryCounts[countryName]
		if !ok {
			countryCounts[countryName] = cases{d.updated, d.confirmed, d.recovered, d.deaths}
		} else {
			updatedConfirmed := c.Confirmed + d.confirmed
			updatedRecovered := c.Recovered + d.recovered
			updatedDeaths := c.Deaths + d.deaths
			updated := d.updated
			if c.Updated > d.updated {
				updated = c.Updated
			}
			countryCounts[countryName] = cases{updated, updatedConfirmed, updatedRecovered, updatedDeaths}
		}

		if countryName == "United States" || countryName == "Canada" || countryName == "Germany" || countryName == "China" {

			countMap := map[string]cases{}
			switch countryName {
			case "United States":
				countMap = usaCounts
			case "Canada":
				countMap = canadaCounts
			case "Germany":
				countMap = germanyCounts
			case "China":
				countMap = chinaCounts
			}

			labelName := d.label
			c, ok := countMap[labelName]
			if !ok {
				countMap[labelName] = cases{d.updated, d.confirmed, d.recovered, d.deaths}
			} else {
				updatedConfirmed := c.Confirmed + d.confirmed
				updatedRecovered := c.Recovered + d.recovered
				updatedDeaths := c.Deaths + d.deaths
				updated := d.updated
				if c.Updated > d.updated {
					updated = c.Updated
				}
				countMap[labelName] = cases{updated, updatedConfirmed, updatedRecovered, updatedDeaths}
			}
		}
	}

	output := results{
		Global:  countryCounts,
		USA:     usaCounts,
		Canada:  canadaCounts,
		Germany: germanyCounts,
		China:   chinaCounts,
	}
	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n", jsonBytes)

	return
}
