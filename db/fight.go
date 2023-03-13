package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type (
	//FightDto dto for fight
	FightDto struct {
		ID                    int
		ZombieName            string     `json:"zombieName"`
		HunterName            string     `json:"hunterName"`
		PaymentAmountLovelace int64      `json:"paymentAmountLovelace"`
		PaymentAmountAda      string     `json:"paymentAmountAda"`
		PaymentAddress        string     `json:"paymentAddress"`
		Status                string     `json:"status"`
		CreatedDate           *time.Time `json:"createdDate"`
		MintedDate            *time.Time `json:"mintedDate,omitempty"`
		FightIPFS             string     `json:"fightIPFS,omitempty"`
		AlienIPFS             string     `json:"alienIPFS,omitempty"`
		MinutesUntilExpired   int        `json:"minutesUntilExpired,omitempty"`
		Winner                string     `json:"winner"`
		Loser                 string     `json:"loser"`
		TweetLink             string     `json:"tweetLink"`
		HunterSendAddress     string
		ZombieSendAddress     string
	}

	//FightDb struct for fight db
	FightDb struct {
		ID                    int            `db:"id"`
		ZombieName            string         `db:"zombie_name"`
		HunterName            string         `db:"hunter_name"`
		Status                string         `db:"status"`
		CreatedDate           time.Time      `db:"created_date"`
		MintedDate            sql.NullTime   `db:"minted_date"`
		PaymentAmountLovelace int64          `db:"payment_amount_lovelace"`
		PaymentAddress        string         `db:"payment_address"`
		IncomingUtxo          sql.NullString `db:"incoming_utxo"`
		IncomingUtxoInt       sql.NullInt64  `db:"incoming_utxo_index"`
		IPFS                  sql.NullString `db:"ipfs_fight"`
		IPFSAlien             sql.NullString `db:"ipfs_alien"`
		Background            sql.NullString `db:"background"`
		ZombieLifeBar         sql.NullInt64  `db:"zclifebar"`
		HunterLifeBar         sql.NullInt64  `db:"zhlifebar"`
		ZombieRecord          sql.NullString `db:"zombie_record"`
		HunterRecord          sql.NullString `db:"hunter_record"`
		ZombieKo              sql.NullBool   `db:"zombie_ko"`
		HunterKo              sql.NullBool   `db:"hunter_ko"`
		Collection            string         `db:"collection"`
		Site                  string         `db:"site"`
		Twitter               string         `db:"twitter"`
		Copyright             string         `db:"copyright"`
		HunterAmountAda       int            `db:"hunter_amount_ada"`
		HunterSendAddress     sql.NullString `db:"hunter_send_address"`
		ZombieAmountAda       int            `db:"zombie_amount_ada"`
		ZombieSendAddress     sql.NullString `db:"zombie_send_address"`
		TxID                  sql.NullString `db:"tx_id"`
		TweetID               sql.NullString `db:"tweet_id"`
	}

	//Alient struct for zfc alien
	Alien struct {
		ID           int            `db:"id"`
		FightID      sql.NullInt64  `db:"fight_id"`
		Ipfs         sql.NullString `db:"ipfs_hash"`
		Name         string         `db:"name"`
		ReadableName string         `db:"readable_name"`
		Background   string         `db:"background"`
		Skin         string         `db:"skin"`
		Clothes      string         `db:"clothes"`
		Eyes         string         `db:"eyes"`
		Mouth        string         `db:"mouth"`
		Hand         string         `db:"hand"`
		Hat          string         `db:"hat"`
		Collection   string         `db:"collection"`
		Site         string         `db:"site"`
		Twitter      string         `db:"twitter"`
		Copyright    string         `db:"copyright"`
	}
)

// CreateFight persist a new fight
func (s Store) CreateFight(fight FightDto, hunterUser UserNfts, zombieUser UserNfts, mintingUser User) (int, error) {
	var id int

	insertUserQuery := `INSERT INTO fight ( hunter_user_id,
											hunter_nft_id,
											hunter_amount_ada,
											zombie_user_id,
											zombie_nft_id,
											zombie_amount_ada,
											payment_amount_lovelace,
											payment_address,
											status,
											minting_user_id,
											hunter_send_address,
											zombie_send_address,
											created_date) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
											RETURNING id`

	err := s.Db.QueryRowx(insertUserQuery, hunterUser.UserID, hunterUser.NftID, hunterUser.ListAmount,
		zombieUser.UserID, zombieUser.NftID, zombieUser.ListAmount,
		fight.PaymentAmountLovelace, fight.PaymentAddress, fight.Status,
		mintingUser.ID, fight.HunterSendAddress, fight.ZombieSendAddress, time.Now()).Scan(&id)
	if err != nil {
		return id, err
	}

	return id, nil
}

//GetFightsForUser get fights for user
func (s Store) GetFightsForUser(user User) ([]FightDb, error) {
	fights := make([]FightDb, 0)

	userNftQuery := `SELECT f.id,
							f.status,
							f.created_date,
							f.minted_date,
							znft.name as zombie_name,
							hnft.name as hunter_name,
							f.payment_address,
							f.payment_amount_lovelace,
							f.ipfs_fight,
							a.ipfs_hash as ipfs_alien,
							f.zclifebar,
							f.zhlifebar,
							f.tweet_id
							FROM fight f
							LEFT JOIN nft znft ON znft.id = f.zombie_nft_id
							LEFT JOIN nft hnft ON hnft.id = f.hunter_nft_id
							LEFT JOIN zfc_alien a ON a.fight_id = f.id
							WHERE f.minting_user_id = $1
							ORDER BY f.created_date desc
							LIMIT 25`

	err := s.Db.Select(&fights, userNftQuery, user.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fights, nil
		}
		return nil, err
	}

	return fights, nil
}

//GetFightsForUserAndId get fights for user and id
func (s Store) GetFightsForUserAndId(user User, fightId int) ([]FightDb, error) {
	fights := make([]FightDb, 0)

	userNftQuery := `SELECT f.id,
							f.status,
							f.created_date,
							f.minted_date,
							znft.name as zombie_name,
							hnft.name as hunter_name,
							f.payment_address,
							f.payment_amount_lovelace
							FROM fight f
							LEFT JOIN nft znft ON znft.id = f.zombie_nft_id
							LEFT JOIN nft hnft ON hnft.id = f.hunter_nft_id
							WHERE f.minting_user_id = $1 and f.id = $2`

	err := s.Db.Select(&fights, userNftQuery, user.ID, fightId)
	if err != nil {
		if err == sql.ErrNoRows {
			return fights, nil
		}
		return nil, err
	}

	return fights, nil
}

//GetFightForUtxo get fight for utxo and index
func (s Store) GetFightForUtxo(utxo string, index int) ([]FightDb, error) {
	fights := make([]FightDb, 0)

	userNftQuery := `SELECT f.id,
							f.status,
							f.created_date,
							f.minted_date,
							znft.name as zombie_name,
							hnft.name as hunter_name,
							f.payment_address,
							f.payment_amount_lovelace
							FROM fight f
							LEFT JOIN nft znft ON znft.id = f.zombie_nft_id
							LEFT JOIN nft hnft ON hnft.id = f.hunter_nft_id
							WHERE f.incoming_utxo = $1 and f.incoming_utxo_index = $2`

	err := s.Db.Select(&fights, userNftQuery, utxo, index)
	if err != nil {
		if err == sql.ErrNoRows {
			return fights, nil
		}
		return nil, err
	}

	return fights, nil
}

//GetQueuedFight get fight for utxo and index
func (s Store) GetQueuedFight() ([]FightDb, error) {
	fights := make([]FightDb, 0)

	userNftQuery := `SELECT f.id,
							f.status,
							f.created_date,
							f.minted_date,
							znft.name as zombie_name,
							hnft.name as hunter_name,
							f.payment_address,
							f.payment_amount_lovelace
							FROM fight f
							LEFT JOIN nft znft ON znft.id = f.zombie_nft_id
							LEFT JOIN nft hnft ON hnft.id = f.hunter_nft_id
							WHERE f.status = 'QUEUED'`

	err := s.Db.Select(&fights, userNftQuery)
	if err != nil {
		if err == sql.ErrNoRows {
			return fights, nil
		}
		return nil, err
	}

	return fights, nil
}

//GetStagedFights get fight for utxo and index
func (s Store) GetStagedFights() ([]FightDb, error) {
	fights := make([]FightDb, 0)

	userNftQuery := `SELECT f.id,
							f.status,
							f.created_date,
							f.minted_date,
							znft.name as zombie_name,
							hnft.name as hunter_name,
							f.payment_address,
							f.payment_amount_lovelace,
							f.incoming_utxo,
							f.incoming_utxo_index,
							f.ipfs_fight,
							f.background,
							f.zombie_record,
							f.hunter_record,
							f.zombie_ko,
							f.hunter_ko,
							f.zclifebar,
							f.zhlifebar,
							f.collection,
							f.site,
							f.twitter,
							f.copyright,
							f.hunter_send_address,
							f.hunter_amount_ada,
							f.zombie_send_address,
							f.zombie_amount_ada
							FROM fight f
							LEFT JOIN nft znft ON znft.id = f.zombie_nft_id
							LEFT JOIN nft hnft ON hnft.id = f.hunter_nft_id
							WHERE f.status = 'STAGED'`

	err := s.Db.Select(&fights, userNftQuery)
	if err != nil {
		if err == sql.ErrNoRows {
			return fights, nil
		}
		return nil, err
	}

	return fights, nil
}

//GetMintedFights get minted fights
func (s Store) GetMintedFights() ([]FightDb, error) {
	fights := make([]FightDb, 0)

	userNftQuery := `SELECT f.id,
							f.status,
							f.created_date,
							f.minted_date,
							znft.name as zombie_name,
							hnft.name as hunter_name,
							f.payment_address,
							f.payment_amount_lovelace,
							f.incoming_utxo,
							f.incoming_utxo_index,
							f.ipfs_fight,
							f.background,
							f.zombie_record,
							f.hunter_record,
							f.zombie_ko,
							f.hunter_ko,
							f.zclifebar,
							f.zhlifebar,
							f.collection,
							f.site,
							f.twitter,
							f.copyright,
							f.hunter_send_address,
							f.hunter_amount_ada,
							f.zombie_send_address,
							f.zombie_amount_ada,
							f.tx_id
							FROM fight f
							LEFT JOIN nft znft ON znft.id = f.zombie_nft_id
							LEFT JOIN nft hnft ON hnft.id = f.hunter_nft_id
							WHERE f.status = 'MINTED'`

	err := s.Db.Select(&fights, userNftQuery)
	if err != nil {
		if err == sql.ErrNoRows {
			return fights, nil
		}
		return nil, err
	}

	return fights, nil
}

//GetFightForPaymentLastFifteen get fight for utxo and index
func (s Store) GetFightForPaymentLastFifteen(lovelace int64) (*FightDb, error) {
	fights := make([]FightDb, 0)

	userNftQuery := `SELECT f.id,
							f.status,
							f.created_date,
							f.minted_date,
							znft.name as zombie_name,
							hnft.name as hunter_name,
							f.payment_address,
							f.payment_amount_lovelace
							FROM fight f
							LEFT JOIN nft znft ON znft.id = f.zombie_nft_id
							LEFT JOIN nft hnft ON hnft.id = f.hunter_nft_id
							WHERE f.incoming_utxo is null and f.payment_amount_lovelace = $1 and f.created_date > NOW()::timestamp - INTERVAL '20 minutes'`

	err := s.Db.Select(&fights, userNftQuery, lovelace)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if len(fights) > 1 {
		return nil, fmt.Errorf("More than one row returned")
	} else if len(fights) == 0 {
		return nil, nil
	}

	return &fights[0], nil
}

//GetNextAvailableAlien get next available alien
func (s Store) GetNextAvailableAlien() (*Alien, error) {
	aliens := make([]Alien, 0)

	userNftQuery := `SELECT *
							FROM zfc_alien
							WHERE fight_id is null ORDER BY id asc LIMIT 1`

	err := s.Db.Select(&aliens, userNftQuery)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if len(aliens) > 1 {
		return nil, fmt.Errorf("More than one row returned")
	} else if len(aliens) == 0 {
		return nil, nil
	}

	return &aliens[0], nil
}

//GetAlienByFightId get alient by fight id
func (s Store) GetAlienByFightId(fightId int) (*Alien, error) {
	aliens := make([]Alien, 0)

	userNftQuery := `SELECT *
							FROM zfc_alien
							WHERE fight_id = $1`

	err := s.Db.Select(&aliens, userNftQuery, fightId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if len(aliens) > 1 {
		return nil, fmt.Errorf("More than one row returned")
	} else if len(aliens) == 0 {
		return nil, nil
	}

	return &aliens[0], nil
}

func (s Store) MoveFightFromPendingToQueued(ctx context.Context, fightID int, alienID int, utxo string, utxoIndex int) error {
	tx, err := s.Db.BeginTx(ctx, nil)
	if err != nil {
		logrus.New().WithError(err).Error("Beginning tx")
		return err
	}
	defer tx.Rollback()

	// update alien fk
	_, err = tx.ExecContext(ctx, "UPDATE zfc_alien SET fight_id = $1 WHERE id = $2", fightID, alienID)
	if err != nil {
		logrus.New().WithError(err).Error("Setting fight id on alien")
		return err
	}

	// update fight
	_, err = tx.ExecContext(ctx, "UPDATE fight SET status = $1, incoming_utxo = $2, incoming_utxo_index = $3 WHERE id = $4", "QUEUED", utxo, utxoIndex, fightID)
	if err != nil {
		logrus.New().WithError(err).Error("Updating fight status")
		return err
	}

	// Commit the transaction.
	if err = tx.Commit(); err != nil {
		logrus.New().WithError(err).Error("Committing tx")
		return err
	}

	return nil
}

func (s Store) MoveFightFromQueuedToStaged(ctx context.Context, alienID int, alienIpfs string, fightID int, fightIpfs, background, zombieRecord, hunterRecord string, zcLifeBar, zhLifeBar int, zombieKo, hunterKo, zombieBeahup, hunterBeatup bool, winningNft string, losingNft string) error {
	tx, err := s.Db.BeginTx(ctx, nil)
	if err != nil {
		logrus.New().WithError(err).Error("Beginning tx")
		return err
	}
	defer tx.Rollback()

	// update alien ipfs
	_, err = tx.ExecContext(ctx, "UPDATE zfc_alien SET ipfs_hash = $1 WHERE id = $2", alienIpfs, alienID)
	if err != nil {
		logrus.New().WithError(err).Error("Setting fight id on alien")
		return err
	}

	// update nft record
	_, err = tx.ExecContext(ctx, "UPDATE nft SET wins = wins +1 WHERE name = $1", winningNft)
	if err != nil {
		logrus.New().WithError(err).Error("Setting nft winner")
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE nft SET loses = loses + 1 WHERE name = $1", losingNft)
	if err != nil {
		logrus.New().WithError(err).Error("Setting nft winner")
		return err
	}

	updateFightSql := `UPDATE fight SET status = $1,
									ipfs_fight = $2,
									background = $3,
									zhLifeBar = $4,
									zcLifeBar = $5,
									hunter_record = $6,
									zombie_record = $7,
									hunter_ko = $8,
									zombie_ko = $9,
									hunter_beatup = $10,
									zombie_beatup = $11
									WHERE id = $12`

	// update fight
	_, err = tx.ExecContext(ctx, updateFightSql, "STAGED", fightIpfs, background, zhLifeBar, zcLifeBar, hunterRecord, zombieRecord, hunterKo, zombieKo, hunterBeatup, zombieBeahup, fightID)
	if err != nil {
		logrus.New().WithError(err).Error("Updating fight status")
		return err
	}

	// Commit the transaction.
	if err = tx.Commit(); err != nil {
		logrus.New().WithError(err).Error("Committing tx")
		return err
	}

	return nil
}

func (s Store) MoveFightFromStagedToMinted(ctx context.Context, fightID int, txHash string) error {
	tx, err := s.Db.BeginTx(ctx, nil)
	if err != nil {
		logrus.New().WithError(err).Error("Beginning tx")
		return err
	}
	defer tx.Rollback()

	updateFightSql := `UPDATE fight SET status = $1,
									minted_date = $2,
									tx_id = $3
									WHERE id = $4`

	// update fight
	_, err = tx.ExecContext(ctx, updateFightSql, "MINTED", time.Now(), txHash, fightID)
	if err != nil {
		logrus.New().WithError(err).Error("Updating fight status")
		return err
	}

	// Commit the transaction.
	if err = tx.Commit(); err != nil {
		logrus.New().WithError(err).Error("Committing tx")
		return err
	}

	return nil
}

func (s Store) MoveFightFromMintedToConfirmed(ctx context.Context, fightID int) error {
	tx, err := s.Db.BeginTx(ctx, nil)
	if err != nil {
		logrus.New().WithError(err).Error("Beginning tx")
		return err
	}
	defer tx.Rollback()

	updateFightSql := `UPDATE fight SET status = $1
										WHERE id = $2`

	// update fight
	_, err = tx.ExecContext(ctx, updateFightSql, "CONFIRMED", fightID)
	if err != nil {
		logrus.New().WithError(err).Error("Updating fight status")
		return err
	}

	// Commit the transaction.
	if err = tx.Commit(); err != nil {
		logrus.New().WithError(err).Error("Committing tx")
		return err
	}

	return nil
}

func (s Store) UpdateTweetID(ctx context.Context, fightID int, tweetID string) error {
	updateFightSql := `UPDATE fight SET tweet_id = $1
										WHERE id = $2`

	// update fight
	_, err := s.Db.ExecContext(ctx, updateFightSql, tweetID, fightID)
	if err != nil {
		logrus.New().WithError(err).Error("Updating tweet id")
		return err
	}

	return nil
}
