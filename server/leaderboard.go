package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	db "github.com/reliablestaking/zombie-fight-club-server/db"
	"github.com/sirupsen/logrus"
)

type (
	Leaders struct {
		MostWins    []Nft `json:"mostWins"`
		MostLosses  []Nft `json:"mostLoses"`
		BestPercent []Nft `json:"bestPercent"`
	}
)

//GetLeaders get all leaders
func (s Server) GetLeaders(c echo.Context) (err error) {
	log := logrus.WithContext(c.Request().Context())

	leaderCache, found := s.LeaderCache.Get("leaders")
	if found {
		return c.JSON(http.StatusOK, leaderCache)
	}

	wins, err := s.Store.GetNftMostWins(10)
	if err != nil {
		log.WithError(err).Error("Error getting nfts")
		return s.RenderError("Error getting nfts", c)
	}
	loses, err := s.Store.GetNftMostLoses(10)
	if err != nil {
		log.WithError(err).Error("Error getting nfts")
		return s.RenderError("Error getting nfts", c)
	}
	percent, err := s.Store.GetNftHighestPercentMinimum(10, 10)
	if err != nil {
		log.WithError(err).Error("Error getting nfts")
		return s.RenderError("Error getting nfts", c)
	}

	leaders := Leaders{
		MostWins:    s.convertDbNftToNftDto(wins),
		MostLosses:  s.convertDbNftToNftDto(loses),
		BestPercent: s.convertDbNftToNftDto(percent),
	}

	c.Set("leaders", leaders)

	return c.JSON(http.StatusOK, leaders)
}

func (s Server) convertDbNftToNftDto(nfts []db.Nft) []Nft {
	dto := make([]Nft, 0)

	for _, nft := range nfts {
		zc := Nft{
			Name:  nft.NftName,
			Wins:  nft.Wins,
			Loses: nft.Loses,
		}

		zc.Type = nft.NftType

		if zc.Type == "Zombie" {
			zc.IPFS = s.ZombieMeta[nft.NftName]
		} else if zc.Type == "Hunter" {
			zc.IPFS = s.HunterMeta[nft.NftName]
		}

		dto = append(dto, zc)
	}

	return dto
}
