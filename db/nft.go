package store

import (
	"database/sql"
)

type (
	// UserNfts struct to hold nfts owned by user
	UserNfts struct {
		UserID     int           `db:"zfc_user_id"`
		NftID      int           `db:"nft_id"`
		NftName    string        `db:"name"`
		NftType    string        `db:"nft_type"`
		ListAmount sql.NullInt16 `db:"amount_ada"`
		ListDate   sql.NullTime  `db:"listed_date"`
		Wins       int           `db:"wins"`
		Loses      int           `db:"loses"`
	}

	Nft struct {
		ID      int             `db:"id"`
		NftName string          `db:"name"`
		NftType string          `db:"nft_type"`
		Wins    int             `db:"wins"`
		Loses   int             `db:"loses"`
		Percent sql.NullFloat64 `db:"winpercent"`
	}
)

// GetNftsOwnedByUser Gets owned nfts by user
func (s Store) GetNftsOwnedByUser(userID int) ([]UserNfts, error) {
	userNfts := make([]UserNfts, 0)

	userNftQuery := `SELECT un.zfc_user_id, un.nft_id, un.amount_ada, n.name, n.nft_type, n.wins, n.loses FROM zfc_user_nft un
					LEFT JOIN nft n ON n.id = un.nft_id WHERE un.zfc_user_id = $1`

	err := s.Db.Select(&userNfts, userNftQuery, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return userNfts, nil
		}
		return nil, err
	}

	return userNfts, nil
}

// InsertZcNftOwnedByUser inserts nft owned by user
func (s Store) InsertZcNftOwnedByUser(userID int, nftID int) error {
	userNftInsert := `INSERT INTO zfc_user_nft (zfc_user_id,nft_id) VALUES($1, $2)`

	rows, err := s.Db.Query(userNftInsert, userID, nftID)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

// RemoveZcNftNotOwnedByUser removes nft not owned by user
func (s Store) RemoveZcNftNotOwnedByUser(userID int, nftID int) error {
	userNftInsert := `DELETE FROM zfc_user_nft WHERE zfc_user_id != $1 AND nft_id = $2`

	rows, err := s.Db.Query(userNftInsert, userID, nftID)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

// RemoveZcNftOwnedByUser remove nft owned by user
func (s Store) RemoveZcNftOwnedByUser(userID int, nftID int) error {
	userNftInsert := `DELETE FROM zfc_user_nft WHERE zfc_user_id = $1 AND nft_id = $2`

	rows, err := s.Db.Query(userNftInsert, userID, nftID)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

// UpdateNftListPrice update list price
func (s Store) UpdateNftListPrice(listPrice *int16, userID, nftID int) error {
	userNftInsert := `UPDATE zfc_user_nft SET amount_ada = $1, listed_date = now() WHERE zfc_user_id = $2 AND nft_id = $3`

	rows, err := s.Db.Query(userNftInsert, listPrice, userID, nftID)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

func (s Store) GetNftByName(name string) (*Nft, error) {
	nft := Nft{}

	err := s.Db.Get(&nft, "SELECT * FROM nft WHERE name = $1", name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &nft, nil
}

//GetListedNfts get all nfts listed
func (s Store) GetListedNfts(nftType string, limit int, random bool) ([]UserNfts, error) {
	userNfts := make([]UserNfts, 0)

	// get listed nfts
	userNftQuery := `SELECT un.nft_id, un.amount_ada, n.name, n.nft_type, n.wins, n.loses FROM zfc_user_nft un
					LEFT JOIN nft n ON n.id = un.nft_id WHERE un.amount_ada is not null`

	userNftQuery += " AND n.nft_type = $1"
	if random {
		userNftQuery += " ORDER BY RANDOM ()"
	}
	userNftQuery += " LIMIT $2"

	err := s.Db.Select(&userNfts, userNftQuery, nftType, limit)
	if err != nil {
		if err == sql.ErrNoRows {
			return userNfts, nil
		}
		return nil, err
	}

	return userNfts, nil
}

func (s Store) GetListedNftsByName(nftType string, name string) ([]UserNfts, error) {
	userNfts := make([]UserNfts, 0)

	// get listed nfts
	userNftQuery := `SELECT un.nft_id, un.amount_ada, n.name, n.nft_type, n.wins, n.loses FROM zfc_user_nft un
					LEFT JOIN nft n ON n.id = un.nft_id WHERE un.amount_ada is not null`
	userNftQuery += " AND n.nft_type = $1"
	userNftQuery += " AND n.name LIKE '%' || $2 || '%'"

	err := s.Db.Select(&userNfts, userNftQuery, nftType, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return userNfts, nil
		}
		return nil, err
	}

	return userNfts, nil
}

//GetListedNftByName get nft by name
func (s Store) GetListedNftByName(name string) ([]UserNfts, error) {
	userNfts := make([]UserNfts, 0)

	userNftQuery := `SELECT un.zfc_user_id, un.nft_id, un.amount_ada, n.name, n.nft_type FROM zfc_user_nft un
					LEFT JOIN nft n ON n.id = un.nft_id WHERE n.name = $1`

	err := s.Db.Select(&userNfts, userNftQuery, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return userNfts, nil
		}
		return nil, err
	}

	return userNfts, nil
}

func (s Store) GetNftMostWins(limit int) ([]Nft, error) {
	nfts := make([]Nft, 0)

	err := s.Db.Select(&nfts, "SELECT * FROM nft ORDER BY wins DESC LIMIT $1", limit)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return nfts, nil
}

func (s Store) GetNftMostLoses(limit int) ([]Nft, error) {
	nfts := make([]Nft, 0)

	err := s.Db.Select(&nfts, "SELECT * FROM nft ORDER BY loses DESC LIMIT $1", limit)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return nfts, nil
}

func (s Store) GetNftHighestPercentMinimum(minimum int, limit int) ([]Nft, error) {
	nfts := make([]Nft, 0)

	err := s.Db.Select(&nfts, "SELECT id, name, nft_type, wins, loses, (wins/(wins+loses)::float)*100 as winpercent FROM nft WHERE wins >= $1 ORDER BY winpercent DESC LIMIT $2", minimum, limit)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return nfts, nil
}
