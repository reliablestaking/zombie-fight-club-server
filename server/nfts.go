package server

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	db "github.com/reliablestaking/zombie-fight-club-server/db"
	"github.com/reliablestaking/zombie-fight-club-server/nftkeyme"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func (s Server) GetAssetsForUser(ctx context.Context, zfcUser db.User) ([]Nft, error) {
	log := logrus.WithContext(ctx)

	// get from DB
	nfts, err := s.Store.GetNftsOwnedByUser(zfcUser.ID)
	if err != nil {
		log.WithError(err).Error("Error getting nfts owned by user from db")
		return nil, err
	}

	// determine when to update
	if zfcUser.LastAssetCheckTime.Valid {
		updateDuration := time.Now().Sub(zfcUser.LastAssetCheckTime.Time)
		if updateDuration < time.Hour {
			log.Infof("Last check %v, so just return results", updateDuration.Minutes())
			return s.convertNftToDto(nfts), nil
		}
		log.Infof("Need to check for assets again %v", updateDuration.Minutes())
	}

	//get new token
	//TODO: add this to func
	t := oauth2.Token{
		RefreshToken: zfcUser.NftkeymeRefreshToken,
	}
	tokenSource := s.NftkeymeOauthConfig.TokenSource(oauth2.NoContext, &t)
	newToken, err := tokenSource.Token()
	if err != nil {
		logrus.WithError(err).Error("Error getting token")
		return nil, err
	}
	if newToken.AccessToken != zfcUser.NftkeymeAccessToken {
		logrus.Infof("Updating zfc user %s with new token", zfcUser.NftkeymeID)
		err = s.Store.UpdatedUser(zfcUser.NftkeymeID, newToken.AccessToken, newToken.RefreshToken)
		if err != nil {
			logrus.WithError(err).Error("Error updating discord user")
			return nil, err
		}
	}

	// lookup zombies from nft key me
	zombies, err := s.NftkeymeClient.GetAssetsForUser(newToken.AccessToken, s.ZombiePolicyId)
	if err != nil {
		log.WithError(err).Error("Error getting assets from nft key for user")
		return nil, err
	}

	hunters, err := s.NftkeymeClient.GetAssetsForUser(newToken.AccessToken, s.HunterPolicyId)
	if err != nil {
		log.WithError(err).Error("Error getting assets from nft key for user")
		return nil, err
	}

	//combine into one slice
	zcs := make([]nftkeyme.Asset, 0)
	zcs = append(zcs, zombies...)
	zcs = append(zcs, hunters...)

	//persist owned zombie
	ownedAssetNames := make([]string, 0)
	for _, zc := range zcs {
		//convert asset name to string
		nftName, err := hex.DecodeString(zc.AssetName)
		if err != nil {
			return nil, err
		}

		logrus.Infof("User %d owns zombie %s", zfcUser.ID, nftName)
		ownedAssetNames = append(ownedAssetNames, string(nftName))

		// get nft from DB
		nft, err := s.Store.GetNftByName(string(nftName))
		if err != nil {
			log.WithError(err).Errorf("Error getting nft with name %s", nftName)
			return nil, err
		}
		if nft == nil {
			log.WithError(err).Errorf("No nft with name %s", nftName)
			return nil, fmt.Errorf("No asset with name %s", nftName)
		}

		// if not already in db, then persist
		if !containsNft(nfts, string(nftName)) {
			err = s.Store.InsertZcNftOwnedByUser(zfcUser.ID, nft.ID)
			if err != nil {
				log.WithError(err).Error("Error inserting zfc nft owned")
				return nil, err
			}
		}

		// remove from db in case someone else owned it too
		err = s.Store.RemoveZcNftNotOwnedByUser(zfcUser.ID, nft.ID)
		if err != nil {
			log.WithError(err).Error("Error inserting zfc nft owned")
			return nil, err
		}
	}

	// delete any not owned anymore
	for _, nft := range nfts {
		//log.Infof("Checking if user still owns %s", nft.NftName)
		if !containsString(ownedAssetNames, nft.NftName) {
			log.Infof("User no longer owns %s, remove from db", nft.NftName)
			nft, err := s.Store.GetNftByName(nft.NftName)
			if err != nil {
				log.WithError(err).Errorf("Error getting nft with name %s", nft.NftName)
				return nil, err
			}

			err = s.Store.RemoveZcNftOwnedByUser(zfcUser.ID, nft.ID)
			if err != nil {
				log.WithError(err).Errorf("Error removing nft with name %s", nft.NftName)
				return nil, err
			}
		}
	}

	// set update date
	err = s.Store.SetLastAssetCheckTime(zfcUser.NftkeymeID, time.Now())
	if err != nil {
		log.WithError(err).Error("Error updating asset check time")
		return nil, err
	}

	// lookup again?
	nfts, err = s.Store.GetNftsOwnedByUser(zfcUser.ID)
	if err != nil {
		return nil, err
	}

	return s.convertNftToDto(nfts), nil

}

func (s Server) doesUserOwnNft(ctx context.Context, nftName string, userID int) (bool, error) {
	dbUser, err := s.Store.GetUserByID(userID)
	if err != nil {
		return false, err
	}

	assets, err := s.GetAssetsForUser(ctx, *dbUser)
	if err != nil {
		return false, err
	}

	for _, asset := range assets {
		if asset.Name == nftName {

			return true, nil
		}
	}

	return false, nil
}

func (s Server) convertNftToDto(nfts []db.UserNfts) []Nft {
	dto := make([]Nft, 0)

	for _, nft := range nfts {
		zc := Nft{
			ID:    nft.NftID,
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

		if nft.ListAmount.Valid {
			listPrice := nft.ListAmount.Int16
			zc.ListedPriceAda = &listPrice
		}

		dto = append(dto, zc)
	}

	return dto
}

func containsNft(nfts []db.UserNfts, nftName string) bool {
	for _, nft := range nfts {
		if nft.NftName == nftName {
			return true
		}
	}
	return false
}

func containsNftDto(nfts []Nft, nftName string) *Nft {
	for _, nft := range nfts {
		if nft.Name == nftName {
			return &nft
		}
	}
	return nil
}

func containsString(s []string, z string) bool {
	for _, a := range s {
		if a == z {
			return true
		}
	}
	return false
}
