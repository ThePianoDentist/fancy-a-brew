package storage

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type User struct {
	UserId          uuid.UUID
	FirebaseToken   string
	DefaultNickname string
	TheUsual        string
	LastKnownLong   float64
	LastKnownLat    float64
}

func (u *User) CreateUser(db *sql.DB) (uuid.UUID, error) {
	// //https://stackoverflow.com/a/47396542 for geolocation
	err := db.QueryRow(
		"INSERT INTO appusers(firebase_token, default_nickname, the_usual, last_known_location) "+
			"VALUES($1, $2, $3, $4) RETURNING user_id",
		u.FirebaseToken, u.DefaultNickname, u.TheUsual, fmt.Sprintf("POINT(%f %f)", u.LastKnownLong, u.LastKnownLat),
	).Scan(&u.UserId)
	if err != nil {
		return uuid.UUID{}, err
	}

	return u.UserId, nil
}

// Creates new user if doesn't exist (no matching firebase-token). Or just updates existing users location
func (u *User) UpsertUser(db *sql.DB) (uuid.UUID, error) {
	setLastKnowLocationFragment := "SET "
	if u.LastKnownLat != 0.0 || u.LastKnownLong != 0.0 {
		setLastKnowLocationFragment = "SET last_known_location=EXCLUDED.last_known_location,"
	}
	err := db.QueryRow(
		"INSERT INTO appusers(firebase_token, default_nickname, the_usual, last_known_location) "+
			"VALUES($1, $2, $3, $4) "+
			"ON CONFLICT(firebase_token) DO UPDATE "+
			setLastKnowLocationFragment+
			// this coalesce with nullif, will basically update the column if the update-value is non-null AND not-empty-string
			"default_nickname=COALESCE(NULLIF(EXCLUDED.default_nickname,''), appusers.default_nickname),"+
			"the_usual=COALESCE(NULLIF(EXCLUDED.the_usual,''), appusers.the_usual) "+
			"RETURNING user_id",
		u.FirebaseToken, u.DefaultNickname, u.TheUsual, fmt.Sprintf("POINT(%f %f)", u.LastKnownLong, u.LastKnownLat),
	).Scan(&u.UserId)
	if err != nil {
		return uuid.UUID{}, err
	}

	return u.UserId, nil
}

func GetUser(db *sql.DB, userId uuid.UUID) (User, error) {
	var user User
	err := db.QueryRow("SELECT user_id, firebase_token, the_usual, default_nickname FROM appusers"+
		" WHERE user_id = $1", userId).Scan(&user.UserId, &user.FirebaseToken, &user.TheUsual, &user.DefaultNickname)
	return user, err
}

func GetUserIdFromToken(db *sql.DB, firebaseToken string) (uuid.UUID, error) {
	var userId uuid.UUID
	err := db.QueryRow("SELECT user_id from appusers WHERE firebase_token = $1", firebaseToken).Scan(&userId)
	return userId, err
}

func GetUsersWithinRadius(db *sql.DB, long, lat float64, metreRadius int32) ([]User, error) {

	rows, err := db.Query(
		"SELECT user_id, firebase_token, default_nickname, the_usual FROM appusers WHERE ST_DWithin(last_known_location, ST_MakePoint($1,$2)::geography, $3)", long, lat, metreRadius,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)

	for rows.Next() {
		var r User
		if err := rows.Scan(&r.UserId, &r.FirebaseToken, &r.DefaultNickname, &r.TheUsual); err != nil {
			return nil, err
		}
		users = append(users, r)
	}

	return users, nil
}
