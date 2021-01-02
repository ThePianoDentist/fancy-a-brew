package storage

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Kettle struct {
	KettleId     uuid.UUID `json:"kettleId"`
	WirelessId   string    `json:"wirelessId"`
	Name         string    `json:"name"`
	CurrentMaker uuid.UUID `json:"currentMaker"`
	Long         float64
	Lat          float64
}

func GetKettle(db *sql.DB, kettleId uuid.UUID) (Kettle, error) {
	var k Kettle
	// Is there a nice way to map sturct-fields to rows. i.e. like `json="lowercasedname"`?
	err := db.QueryRow("SELECT kettle_id, wireless_id, name, current_maker, ST_X(location::geometry), ST_Y(location::geometry) "+
		"FROM kettles WHERE kettle_id = $1", kettleId).Scan(&k.KettleId, &k.WirelessId, &k.Name, &k.CurrentMaker, &k.Long, &k.Lat)
	if err != nil {
		return Kettle{}, err
	}

	return k, nil
}

func (k *Kettle) UpsertKettle(db *sql.DB) (uuid.UUID, error) {
	setLocationFragment := "SET "
	if k.Long != 0.0 || k.Lat != 0.0 {
		setLocationFragment = "SET location=EXCLUDED.location, "
	}
	err := db.QueryRow(
		"INSERT INTO kettles(wireless_id, name, location) "+
			"VALUES($1, $2, $3) "+
			"ON CONFLICT(wireless_id) DO UPDATE "+
			setLocationFragment+
			// this coalesce with nullif, will basically update the column if the update-value is non-null AND not-empty-string
			"wireless_id=COALESCE(NULLIF(EXCLUDED.wireless_id,''), kettles.wireless_id),"+
			"name=COALESCE(NULLIF(EXCLUDED.name,''), kettles.name) "+
			"RETURNING kettle_id",
		k.WirelessId, k.Name, fmt.Sprintf("POINT(%f %f)", k.Long, k.Lat),
	).Scan(&k.KettleId)
	if err != nil {
		return uuid.UUID{}, err
	}

	return k.KettleId, nil
}

// could prob also achieve this with upsert but meh
// (actually upsert is hard because we SPECIFICALLY want to blank/NULL it. so hard to differentiate between
// a) we actually want to null.
// b) we want to leave it as it is.
func SetCurrentMaker(db *sql.DB, kettleId, userId uuid.UUID) error {
	var kid uuid.UUID
	if (userId == uuid.UUID{}) {
		return db.QueryRow(
			"UPDATE kettles SET current_maker = null WHERE kettle_id = $1 returning kettle_id",
			kettleId,
		).Scan(&kid)
	}
	return db.QueryRow(
		"UPDATE kettles SET current_maker = $2 WHERE kettle_id = $1 returning kettle_id",
		kettleId, userId,
	).Scan(&kid)
}

func GetKettlesWithinRadius(db *sql.DB, long, lat float64, metreRadius int32) ([]Kettle, error) {
	// get all kettles in surrounding area.
	// "join" a kettle means.....?
	// maybe have a connected_kettle_id in users. and just update it
	// when offer to make a drink:
	// look up connected_kettle. list all users with that as connected_kettle_id.
	// if a user moves out of range, how does their connected-kettle get nuked?
	// Maybe can just check geolocation before send the notification,
	// however a) is it possible to trigger a location sync without notifying user.
	// b) would be nice to list who is going to be available/notified for kettle-round.
	rows, err := db.Query(
		"SELECT kettle_id, wireless_id, name FROM kettles "+
			"WHERE ST_DWithin(location, ST_MakePoint($1,$2)::geography, $3) "+
			"ORDER BY ST_Distance(location, ST_MakePoint($1,$2)::geography)", long, lat, metreRadius,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	kettles := make([]Kettle, 0)

	for rows.Next() {
		var k Kettle
		if err := rows.Scan(&k.KettleId, &k.WirelessId, &k.Name); err != nil {
			return nil, err
		}
		kettles = append(kettles, k)
	}

	return kettles, nil
}
