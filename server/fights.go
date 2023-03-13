package server

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	db "github.com/reliablestaking/zombie-fight-club-server/db"
	"github.com/sirupsen/logrus"
)

//GetMyNfts get all nfts I own
func (s Server) GetNftsToFight(c echo.Context) (err error) {
	log := logrus.WithContext(c.Request().Context())

	dbUser := c.Get("user").(*db.User)

	typeParam := c.QueryParam("type")
	if typeParam == "" {
		log.Warn("No type provided")
		return echo.NewHTTPError(http.StatusBadRequest, "No type provided")
	}

	ownedOnly := false
	ownedParamString := c.QueryParam("owned")
	if ownedParamString == "true" {
		ownedOnly = true
	}

	log.Infof("User %d is getting nfts with type %s", dbUser.ID, typeParam)
	nftDtos := make([]Nft, 0)

	if ownedOnly {
		// get nfts the user owns too
		myAssets, err := s.GetAssetsForUser(c.Request().Context(), *dbUser)
		if err != nil {
			log.WithError(err).Error("Error getting my assets")
			return s.RenderError("Error getting my assets", c)
		}

		// convert to nfts
		for _, asset := range myAssets {
			if asset.Type == typeParam {
				// add to list
				newListNft := Nft{
					Name:     asset.Name,
					Type:     asset.Type,
					UserOwns: true,
					Wins:     asset.Wins,
					Loses:    asset.Loses,
				}

				if asset.Type == "Zombie" {
					newListNft.IPFS = s.ZombieMeta[asset.Name]
				} else if asset.Type == "Hunter" {
					newListNft.IPFS = s.HunterMeta[asset.Name]
				}

				nftDtos = append(nftDtos, newListNft)
			}
		}
	} else {
		nameParam := c.QueryParam("name")
		if nameParam == "" {
			listedNfts, err := s.Store.GetListedNfts(typeParam, 100, true)
			if err != nil {
				log.WithError(err).Error("Error getting listed nfts")
				return s.RenderError("Error getting listed nfts", c)
			}

			nftDtos = s.convertNftToDto(listedNfts)
		} else {
			listedNfts, err := s.Store.GetListedNftsByName(typeParam, nameParam)
			if err != nil {
				log.WithError(err).Error("Error getting listed nfts")
				return s.RenderError("Error getting listed nfts", c)
			}

			nftDtos = s.convertNftToDto(listedNfts)
		}
	}

	return c.JSON(http.StatusOK, nftDtos)
}

//CreateFight create new fight
func (s Server) CreateFight(c echo.Context) (err error) {
	log := logrus.WithContext(c.Request().Context())

	// bind incoming object
	fight := new(db.FightDto)
	if err = c.Bind(fight); err != nil {
		log.WithError(err).Errorf("Error binding")
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	dbUser := c.Get("user").(*db.User)
	log.Infof("Creating fight between %s and %s for user %d", fight.HunterName, fight.ZombieName, dbUser.ID)

	// check that zombie is availabe to fight and still owned by user
	zombies, err := s.Store.GetListedNftByName(fight.ZombieName)
	if err != nil {
		log.WithError(err).Errorf("Error checking if user owns zombie %s", fight.ZombieName)
		return s.RenderError("Error checking if user owns zombie", c)
	}
	if len(zombies) == 0 {
		log.Warnf("Zombie %s not for listed", fight.ZombieName)
		return echo.NewHTTPError(http.StatusBadRequest, "Zombie not listed")
	}
	// if not listed, must be owned by current user
	if !zombies[0].ListAmount.Valid {
		if zombies[0].UserID == dbUser.ID {
			logrus.Infof("User owns this zombie, so okay it's not listed")
			zombies[0].ListAmount = sql.NullInt16{Int16: 0, Valid: true}
		} else {
			log.Warnf("Zombie %s not listed and user doenst own", fight.ZombieName)
			return echo.NewHTTPError(http.StatusBadRequest, "Zombie not listed")
		}
	}
	// check that owned by user system thinks it is
	ownsZombie, err := s.doesUserOwnNft(c.Request().Context(), fight.ZombieName, zombies[0].UserID)
	if err != nil {
		log.WithError(err).Errorf("Error checking user owns %s", fight.ZombieName)
		return s.RenderError("Error checking if user owns zombie", c)
	}
	if !ownsZombie {
		log.Warnf("Zombie %s not owned by user %d", fight.ZombieName, zombies[0].UserID)
		return echo.NewHTTPError(http.StatusBadRequest, "Zombie not owned by user anymore")
	}
	log.Infof("Zombie %s is valid", fight.ZombieName)

	// check that hunter is available to fight and still owned by user
	hunters, err := s.Store.GetListedNftByName(fight.HunterName)
	if err != nil {
		log.WithError(err).Errorf("Error checking if user owns hunter %s", fight.HunterName)
		return s.RenderError("Error checking if user owns hunter", c)
	}
	if len(hunters) == 0 {
		log.Warnf("Hunter %s not for listed", fight.HunterName)
		return echo.NewHTTPError(http.StatusBadRequest, "Hunter not listed")
	}
	// if not listed, must be owned by current user
	if !hunters[0].ListAmount.Valid {
		if hunters[0].UserID == dbUser.ID {
			logrus.Infof("User owns this hunter, so okay it's not listed")
			hunters[0].ListAmount = sql.NullInt16{Int16: 0, Valid: true}
		} else {
			log.Warnf("Hunter %s not listed and user doenst own", fight.HunterName)
			return echo.NewHTTPError(http.StatusBadRequest, "Zombie not listed")
		}
	}

	// check that owned by user system thinks it is
	ownsHunter, err := s.doesUserOwnNft(c.Request().Context(), fight.HunterName, hunters[0].UserID)
	if err != nil {
		log.WithError(err).Errorf("Error checking user owns %s", fight.HunterName)
		return s.RenderError("Error checking if user owns zombie", c)
	}
	if !ownsHunter {
		log.Warnf("Hunter %s not owned by user %d", fight.HunterName, hunters[0].UserID)
		return echo.NewHTTPError(http.StatusBadRequest, "Hunter not owned by user anymore")
	}
	log.Infof("Hunter %s is valid", fight.HunterName)

	// check that user owns at least one or the other (unless they are service account)
	if dbUser.NftkeymeAccessToken != "" && zombies[0].UserID != dbUser.ID && hunters[0].UserID != dbUser.ID {
		log.Warn("User doesn't own zombie or hunter, can't create fight")
		return echo.NewHTTPError(http.StatusBadRequest, "User doesn't own zombie or hunter, can't create fight")
	}

	// calculate cost
	//zombieCost := int64(zombies[0].ListAmount.Int16)
	// if zombies[0].UserID == dbUser.ID {
	// 	log.Infof("User %d owns zombie %s so not cost", dbUser.ID, fight.ZombieName)
	// 	//zombieCost = 0
	// } else {
	// figure out what address asset lives at
	assetName := s.ZombiePolicyId + hex.EncodeToString([]byte(fight.ZombieName))
	logrus.Infof("Finding address for asset %s", assetName)
	addresses, err := s.BlockforstIpfsClient.GetAddressesForAsset(assetName)
	if err != nil {
		log.WithError(err).Errorf("Error getting asset address %s", fight.ZombieName)
		return s.RenderError("Error getting asset address", c)
	} else if len(addresses) == 0 || addresses[0].Address == "" {
		log.WithError(err).Errorf("No asset address %s", fight.HunterName)
		return s.RenderError("No asset address", c)
	}
	logrus.Infof("Setting zombie send address to %d / %s / %s", len(addresses), addresses[0].Address, addresses[0].Quantity)
	fight.ZombieSendAddress = addresses[0].Address
	//}

	//hunterCost := int64(hunters[0].ListAmount.Int16)
	// if hunters[0].UserID == dbUser.ID {
	// 	log.Infof("User %d owns hunter %s so not cost", dbUser.ID, fight.HunterName)
	// 	//hunterCost = 0
	// } else {
	// figure out what address asset lives at
	assetName = s.HunterPolicyId + hex.EncodeToString([]byte(fight.HunterName))
	addresses, err = s.BlockforstIpfsClient.GetAddressesForAsset(assetName)
	if err != nil {
		log.WithError(err).Errorf("Error getting asset address %s", fight.HunterName)
		return s.RenderError("Error getting asset address", c)
	} else if len(addresses) == 0 {
		log.WithError(err).Errorf("No asset address %s", fight.HunterName)
		return s.RenderError("No asset address", c)
	}
	logrus.Infof("Setting hunter send address to %s", addresses[0].Address)
	fight.HunterSendAddress = addresses[0].Address
	// }

	paymentAmountAda := int64(s.BaseCostAda)
	logrus.Infof("Calculated a payment amoutn of %d", paymentAmountAda)

	// calculate dust
	dustTry := 0
	for {
		if dustTry == 5 {
			log.WithError(err).Errorf("Error getting unique payment amount")
			return s.RenderError("Error getting unique payment amount", c)
		}

		cost := (paymentAmountAda * 1000000) + int64(rand.Intn(500000))

		fightCheck, err := s.Store.GetFightForPaymentLastFifteen(cost)
		if err != nil {
			log.WithError(err).Errorf("Error checking fights")
			return s.RenderError("Error checking fights", c)
		}

		if fightCheck == nil {
			fight.PaymentAmountLovelace = cost
			break
		}

		dustTry++
	}

	// set payment address
	fight.PaymentAddress = s.PaymentAddress

	// set initial status
	fight.Status = "PENDING"

	// persist
	fightId, err := s.Store.CreateFight(*fight, hunters[0], zombies[0], *dbUser)
	if err != nil {
		log.WithError(err).Errorf("Error persisting fight")
		return s.RenderError("Error persisting fight", c)
	}
	fight.ID = fightId

	// set minutes til expired
	fight.MinutesUntilExpired = 15

	// set amount in ada
	fight.PaymentAmountAda = lovelaceToString(int(fight.PaymentAmountLovelace))

	return c.JSON(http.StatusOK, fight)
}

func lovelaceToString(lovelace int) string {
	lovelaceString := strconv.Itoa(lovelace)

	lovelaceLen := len(lovelaceString)

	ada := lovelaceString[0 : lovelaceLen-6]
	lace := lovelaceString[lovelaceLen-6 : lovelaceLen]

	return ada + "." + lace
}

//GetFightById gets fight by id
func (s Server) GetFightById(c echo.Context) (err error) {
	log := logrus.WithContext(c.Request().Context())

	// get param
	fightIdString := c.Param("fightId")
	if fightIdString == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "No fight id provided")
	}

	fightId, err := strconv.Atoi(fightIdString)
	if err != nil {
		log.WithError(err).Errorf("Error getting fight with id %s", fightIdString)
		return s.RenderError("Error converting fight id to int", c)
	}

	log.Infof("Getting fight id %d", fightId)

	dbUser := c.Get("user").(*db.User)
	fights, err := s.Store.GetFightsForUserAndId(*dbUser, fightId)
	if err != nil {
		log.WithError(err).Error("Error getting fights for user")
		return s.RenderError("Error getting fights", c)
	}

	fightDtos := make([]db.FightDto, 0)
	if len(fights) > 0 {
		fightDtos = s.convertFightToDto(fights)
	} else {
		return c.String(http.StatusNotFound, "Fight id not found")
	}

	return c.JSON(http.StatusOK, fightDtos[0])
}

func (s Server) convertFightToDto(fights []db.FightDb) []db.FightDto {
	dto := make([]db.FightDto, 0)

	for _, f := range fights {
		fight := db.FightDto{
			ID:                    f.ID,
			ZombieName:            f.ZombieName,
			HunterName:            f.HunterName,
			Status:                f.Status,
			CreatedDate:           &f.CreatedDate,
			PaymentAddress:        f.PaymentAddress,
			PaymentAmountLovelace: f.PaymentAmountLovelace,
		}

		if f.IPFS.Valid {
			fight.FightIPFS = f.IPFS.String
		}

		if f.IPFSAlien.Valid {
			fight.AlienIPFS = f.IPFSAlien.String
		}

		fight.PaymentAmountAda = lovelaceToString(int(fight.PaymentAmountLovelace))
		minutes := int(fight.CreatedDate.Add(15 * time.Minute).Sub(time.Now()).Minutes())

		//logrus.Infof("Comparing %s to %s with a diff of minutes %d", fight.CreatedDate, time.Now(), minutes)

		if minutes > 0 {
			fight.MinutesUntilExpired = minutes
		} else {
			fight.MinutesUntilExpired = 0
		}

		// PENDING > QUEUED > STAGED > MINTED > CONFIRMED > MINTED
		if fight.Status == "PENDING" {
			if time.Now().Sub(*fight.CreatedDate).Minutes() > 15 {
				fight.Status = "EXPIRED"
			} else {
				fight.Status = "AWAITING_PAYMENT"
			}
		} else if fight.Status == "QUEUED" || fight.Status == "STAGED" || fight.Status == "MINTED" {
			fight.Status = "PAYMENT_RECEIVED"
		} else if fight.Status == "CONFIRMED" {
			fight.Status = "MINTED"
		}

		// set winner text
		if f.HunterLifeBar.Valid && f.ZombieLifeBar.Valid {
			if f.HunterLifeBar.Int64 > f.ZombieLifeBar.Int64 {
				fight.Winner = f.HunterName
				fight.Loser = f.ZombieName
			} else {
				fight.Winner = f.ZombieName
				fight.Loser = f.HunterName
			}
		}

		// set tweet link
		if f.TweetID.Valid {
			fight.TweetLink = fmt.Sprintf("https://twitter.com/ZFCBot/status/%s", f.TweetID.String)
		}

		dto = append(dto, fight)
	}

	return dto
}
