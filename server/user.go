package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	db "github.com/reliablestaking/zombie-fight-club-server/db"
	"github.com/sirupsen/logrus"
)

type (
	//Zombie store my zombie info
	Nft struct {
		ID             int
		Name           string `json:"name"`
		Type           string `json:"type"`
		IPFS           string `json:"image"`
		ListedPriceAda *int16 `json:"listedPriceAda,omitempty"`
		UserOwns       bool   `json:"userOwns"`
		Wins           int    `json:"wins"`
		Loses          int    `json:"loses"`
	}
)

//GetMyNfts get all nfts I own
func (s Server) GetMyNfts(c echo.Context) (err error) {
	log := logrus.WithContext(c.Request().Context())
	log.Infof("Getting my nfts for user")

	dbUser := c.Get("user").(*db.User)

	zombies, err := s.GetAssetsForUser(c.Request().Context(), *dbUser)
	if err != nil {
		log.WithError(err).Error("Error getting nfts for user")
		return s.RenderError("Error getting nfts", c)
	}

	return c.JSON(http.StatusOK, zombies)
}

//ListNftForFight list or update nft for fight
func (s Server) ListNftForFight(c echo.Context) (err error) {
	log := logrus.WithContext(c.Request().Context())

	// bind incoming object
	nft := new(Nft)
	if err = c.Bind(nft); err != nil {
		log.WithError(err).Errorf("Error binding list fight")
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	nftName := c.Param("name")
	if nft.Name != nftName {
		return echo.NewHTTPError(http.StatusBadRequest, "Name in path doesn't match name in body")
	}

	dbUser := c.Get("user").(*db.User)

	log.Infof("Listing nft %s for user %d", nft.Name, dbUser.ID)

	//verify actually owns this
	nfts, err := s.GetAssetsForUser(c.Request().Context(), *dbUser)
	if err != nil {
		log.WithError(err).Error("Error getting nfts for user")
		return s.RenderError("Error getting nfts", c)
	}

	existingNft := containsNftDto(nfts, nft.Name)

	if existingNft == nil {
		log.Warn("User doesn't actually own this nft")
		return echo.NewHTTPError(http.StatusBadRequest, "User doens't own this nft")
	}

	//update in db
	//err = s.Store.UpdateNftListPrice(nft.ListedPriceAda, dbUser.ID, existingNft.ID)

	fakeListPrice := int16(5)
	err = s.Store.UpdateNftListPrice(&fakeListPrice, dbUser.ID, existingNft.ID)
	if err != nil {
		log.WithError(err).Error("Error persisting listing")
		return s.RenderError("Error persisting listing", c)
	}

	return c.JSON(http.StatusOK, nft)
}

//ListNftForFight list or update nft for fight
func (s Server) DeleteListedNft(c echo.Context) (err error) {
	log := logrus.WithContext(c.Request().Context())

	// get param
	nftName := c.Param("name")
	if nftName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Nft name in path empty")
	}

	dbUser := c.Get("user").(*db.User)

	log.Infof("Unlisting nft %s for user %d", nftName, dbUser.ID)

	//verify actually owns this
	nfts, err := s.GetAssetsForUser(c.Request().Context(), *dbUser)
	if err != nil {
		log.WithError(err).Error("Error getting nfts for user")
		return s.RenderError("Error getting nfts", c)
	}

	existingNft := containsNftDto(nfts, nftName)
	if existingNft == nil {
		log.Warn("User doesn't actually own this nft")
		return echo.NewHTTPError(http.StatusBadRequest, "User doens't own this nft")
	}

	//update in db
	err = s.Store.UpdateNftListPrice(nil, dbUser.ID, existingNft.ID)
	if err != nil {
		log.WithError(err).Error("Error persisting listing")
		return s.RenderError("Error persisting listing", c)
	}

	return c.JSON(http.StatusOK, nil)
}

//GetMyFights get all of my fight
func (s Server) GetMyFights(c echo.Context) (err error) {
	log := logrus.WithContext(c.Request().Context())
	log.Infof("Getting my fights for user")

	dbUser := c.Get("user").(*db.User)

	fights, err := s.Store.GetFightsForUser(*dbUser)
	if err != nil {
		log.WithError(err).Error("Error getting fights for user")
		return s.RenderError("Error getting fights", c)
	}

	fightDtos := s.convertFightToDto(fights)

	return c.JSON(http.StatusOK, fightDtos)
}
