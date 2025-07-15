package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/lib/pq"
)

type Address struct {
	Road        string  `json:"road,omitempty"`
	Suburb      string  `json:"suburb,omitempty"`
	City        string  `json:"city,omitempty"`
	County      string  `json:"county,omitempty"`
	State       string  `json:"state,omitempty"`
	Country     string  `json:"country,omitempty"`
	CountryCode string  `json:"country_code,omitempty"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

func main() {
	connStr := "host=localhost port=5432 user=osm_user password=osm_pass dbname=osm_db sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/whereami", func(w http.ResponseWriter, r *http.Request) {
		latStr := r.URL.Query().Get("lat")
		lonStr := r.URL.Query().Get("lon")

		lat, err1 := strconv.ParseFloat(latStr, 64)
		lon, err2 := strconv.ParseFloat(lonStr, 64)
		if err1 != nil || err2 != nil {
			http.Error(w, "Invalid lat/lon", http.StatusBadRequest)
			return
		}

		point := fmt.Sprintf("ST_SetSRID(ST_MakePoint(%f, %f), 4326)", lon, lat)

		address := Address{
			Latitude:  lat,
			Longitude: lon,
		}

		// Road: closest named line with highway
		db.QueryRow(`
			SELECT name
			FROM planet_osm_line
			WHERE name IS NOT NULL AND highway IS NOT NULL
			ORDER BY ST_Distance(ST_Transform(way, 4326), ` + point + `)
			LIMIT 1
		`).Scan(&address.Road)

		// Suburb
		db.QueryRow(`
			SELECT name
			FROM planet_osm_point
			WHERE place = 'suburb' AND name IS NOT NULL
			  AND ST_DWithin(ST_Transform(way, 4326), ` + point + `, 1000)
			ORDER BY ST_Distance(ST_Transform(way, 4326), ` + point + `)
			LIMIT 1
		`).Scan(&address.Suburb)

		// City
		db.QueryRow(`
			SELECT name
			FROM planet_osm_point
			WHERE place IN ('city', 'town') AND name IS NOT NULL
			  AND ST_DWithin(ST_Transform(way, 4326), ` + point + `, 3000)
			ORDER BY ST_Distance(ST_Transform(way, 4326), ` + point + `)
			LIMIT 1
		`).Scan(&address.City)

		// County
		db.QueryRow(`
			SELECT name
			FROM planet_osm_polygon
			WHERE boundary = 'administrative' AND admin_level = '6' AND name IS NOT NULL
			  AND ST_Intersects(ST_Transform(way, 4326), ` + point + `)
			LIMIT 1
		`).Scan(&address.County)

		// State
		db.QueryRow(`
			SELECT name
			FROM planet_osm_polygon
			WHERE boundary = 'administrative' AND admin_level = '4' AND name IS NOT NULL
			  AND ST_Intersects(ST_Transform(way, 4326), ` + point + `)
			LIMIT 1
		`).Scan(&address.State)

		// Country
		db.QueryRow(`
			SELECT name
			FROM planet_osm_polygon
			WHERE boundary = 'administrative' AND admin_level = '2' AND name IS NOT NULL
			  AND ST_Intersects(ST_Transform(way, 4326), ` + point + `)
			LIMIT 1
		`).Scan(&address.Country)

		// Country code (fallback — not always available)
		db.QueryRow(`
			SELECT tags -> 'ISO3166-1:alpha2'
			FROM planet_osm_polygon
			WHERE boundary = 'administrative' AND admin_level = '2'
			  AND ST_Intersects(ST_Transform(way, 4326), ` + point + `)
			LIMIT 1
		`).Scan(&address.CountryCode)

		log.Printf("Resolved address: %+v\n",address)

		// Respond
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(address)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🌍 Reverse geocoder running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
