package store

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

type (
	// Store struct to store Db
	Store struct {
		Db *sqlx.DB
	}

	// User struct to store user info
	User struct {
		ID                   int          `db:"id"`
		NftkeymeID           string       `db:"nftkeyme_id"`
		NftkeymeAccessToken  string       `db:"nftkeyme_access_token"`
		NftkeymeRefreshToken string       `db:"nftkeyme_refresh_token"`
		LastAssetCheckTime   sql.NullTime `db:"last_asset_check_time"`
	}
)

// GetUserByNftkeyID Gets a user using their nftkey id
func (s Store) GetUserByNftkeyID(nftKeyUserID string) (*User, error) {
	discordUser := User{}
	err := s.Db.Get(&discordUser, "SELECT * FROM zfc_user where nftkeyme_id = $1", nftKeyUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &discordUser, nil
}

// GetUserByNftkeyID Gets a user using their nftkey id
func (s Store) GetUserByID(userID int) (*User, error) {
	discordUser := User{}
	err := s.Db.Get(&discordUser, "SELECT * FROM zfc_user where id = $1", userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &discordUser, nil
}

// InsertUser inserts a new user into the db
func (s Store) InsertUser(nftkeyID, accessToken, refreshToken string) error {
	insertUserQuery := `INSERT INTO zfc_user (nftkeyme_id,nftkeyme_access_token,nftkeyme_refresh_token) VALUES($1, $2, $3)`

	rows, err := s.Db.Query(insertUserQuery, nftkeyID, accessToken, refreshToken)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

// UpdatedUser updates a new user in the db
func (s Store) UpdatedUser(nftkeyID, accessToken, refreshToken string) error {
	insertUserQuery := `UPDATE zfc_user SET nftkeyme_access_token = $1, nftkeyme_refresh_token = $2 WHERE nftkeyme_id = $3`

	rows, err := s.Db.Query(insertUserQuery, accessToken, refreshToken, nftkeyID)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

// SetLastAssetCheckTime updates last asset check time
func (s Store) SetLastAssetCheckTime(nftkeyID string, now time.Time) error {
	insertUserQuery := `UPDATE zfc_user SET last_asset_check_time = $1 WHERE nftkeyme_id = $2`

	rows, err := s.Db.Query(insertUserQuery, now, nftkeyID)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}
