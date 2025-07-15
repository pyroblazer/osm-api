# 🗺️ OpenStreetMap Reverse Geocoder (PostGIS + Go)

This project sets up a reverse geocoding API using OpenStreetMap data imported into PostgreSQL with PostGIS. It supports querying full location details (road, suburb, city, country, etc.) based on latitude and longitude — like Nominatim.

---

## 📦 Requirements

- PostgreSQL 14+ with PostGIS enabled
- `osm2pgsql`
- Go 1.20+
- macOS with Apple Silicon (or compatible)
- Tested with:
  - `osm2pgsql 2.1.1`
  - `postgresql@14` via Homebrew
  - Central America and Fiji `.osm.pbf` data

---

## 🗺️ Download OSM `.pbf` Data

```bash
mkdir -p ~/osm-data && cd ~/osm-data

# Fiji
curl -L -o fiji.osm.pbf https://download.geofabrik.de/australia-oceania/fiji-latest.osm.pbf

# Guatemala
curl -L -o guatemala.osm.pbf https://download.geofabrik.de/central-america/guatemala-latest.osm.pbf
````

---

## 🛠️ PostgreSQL Setup

```bash
# Login and create user and database
psql -U $(whoami) -d postgres

-- Inside psql shell:
CREATE ROLE osm_user WITH LOGIN PASSWORD 'osm_pass';
CREATE DATABASE osm_db OWNER osm_user;
\c osm_db
CREATE EXTENSION postgis;
```

> 💡 If `CREATE EXTENSION postgis` gives a permission error, make sure you are logged in as a **superuser**.

---

## 🧪 Import Data into PostGIS

### Option 1: Clean Start (Deletes Existing OSM Data)

```bash
PGPASSWORD=osm_pass osm2pgsql \
  --create \
  --drop \
  --database osm_db \
  --username osm_user \
  --host localhost \
  --slim \
  --hstore \
  --number-processes 4 \
  fiji.osm.pbf
```

Then:

```bash
PGPASSWORD=osm_pass osm2pgsql \
  --append \
  --database osm_db \
  --username osm_user \
  --host localhost \
  --slim \
  --hstore \
  --number-processes 4 \
  guatemala.osm.pbf
```

### Option 2: Re-import Without Deleting Tables

Manually clear OSM data but preserve schema:

```bash
psql -U osm_user -d osm_db -h localhost -c "TRUNCATE planet_osm_point, planet_osm_line, planet_osm_polygon, planet_osm_roads;"
```

Then re-run imports:

```bash
# Import both again (no --drop)
PGPASSWORD=osm_pass osm2pgsql --create --database osm_db --username osm_user --host localhost --slim --hstore --number-processes 4 fiji.osm.pbf
PGPASSWORD=osm_pass osm2pgsql --append --database osm_db --username osm_user --host localhost --slim --hstore --number-processes 4 guatemala.osm.pbf
```

---

## 🧑‍💻 Run Reverse Geocoder API (Go)

### Install dependencies and run:

```bash
go mod init reverse-geocoder
go get github.com/lib/pq
go run main.go
```

### `main.go`

See [`main.go`](./main.go) for full code. It returns results like:

```json
{
  "road": "Ratu Mara Road",
  "suburb": "Raiwasa",
  "city": "Suva",
  "county": "Rewa",
  "state": "Central",
  "country": "Fiji",
  "country_code": "fj",
  "latitude": -18.1241,
  "longitude": 178.4501
}
```

---

## 🌐 Example API Usage

### Fiji:

```bash
curl "http://localhost:8080/whereami?lat=-18.1241&lon=178.4501"
```

### Guatemala:

```bash
curl "http://localhost:8080/whereami?lat=14.6349&lon=-90.5069"
```

---

## 📌 Notes

* The API logic mimics [Nominatim](https://nominatim.openstreetmap.org/) with heuristics from OSM tags (`place`, `admin_level`, `highway`, etc).
* You can extend it to support:

  * GeoJSON output
  * ISO codes
  * Display names
  * Redis caching
  * Frontend with Leaflet/Mapbox

---

## 🧽 Cleanup

To remove all OSM data:

```bash
psql -U osm_user -d osm_db -h localhost -c "DROP TABLE IF EXISTS planet_osm_point, planet_osm_line, planet_osm_polygon, planet_osm_roads;"
```

To reset DB completely:

```bash
psql -U $(whoami) -d postgres -c "DROP DATABASE osm_db;"
```

---

## 🧭 License

Data © [OpenStreetMap contributors](https://www.openstreetmap.org/copyright)