package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	bfg "github.com/blockfrost/blockfrost-go"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	cli "github.com/reliablestaking/zombie-fight-club-server/cardanocli"
	store "github.com/reliablestaking/zombie-fight-club-server/db"
	"github.com/reliablestaking/zombie-fight-club-server/imagebuilder"
	"github.com/reliablestaking/zombie-fight-club-server/metadata"
	"github.com/reliablestaking/zombie-fight-club-server/twitter"
	"github.com/sirupsen/logrus"
)

type (
	SignedTx struct {
		Hex string `json:"cborHex"`
	}

	NFTMetadata struct {
		Name         string            `json:"name"`
		Image        []string          `json:"image"`
		Collection   *string           `json:"Project,omitempty"`
		Traits       map[string]string `json:"traits,omitempty"`
		Artist       *string           `json:"artist,omitempty"`
		Site         *string           `json:"website,omitempty"`
		Twitter      *string           `json:"twitter,omitempty"`
		Copyright    *string           `json:"copyright,omitempty"`
		Type         *string           `json:"type,omitempty"`
		FileMetadata []FileMetadata    `json:"files,omitempty"`
	}

	FileMetadata struct {
		MediaType string   `json:"mediaType"`
		Name      string   `json:"name"`
		Src       []string `json:"src"`
	}

	NFTReturn struct {
		ReturnToAddress string `json:"ReturnToAddress"`
		FromUtxo        string `json:"FromUtxo"`
		FromUtxoAmount  int    `json:"FromUtxoAmount"`
	}
)

func (s Server) RunMintingEngine() {
	processRefunds := false
	processRefundsString := os.Getenv("PROCESS_REFUNDS")
	if processRefundsString == "true" {
		processRefunds = true
	}

	errorOnCheck := false
	refundCheck := 0

	for {
		logrus.Infof("Running minting check for address %s", s.PaymentAddress)

		returns := make([]NFTReturn, 0)

		// sleep if an error
		if errorOnCheck {
			logrus.Error("Error on monitoring check, sleeping for 30 seconds...")
			time.Sleep(30 * time.Second)
			errorOnCheck = false
		}

		// get all utxos
		utxos, err := getAllUtxos(s.PaymentAddress, s.BlockfrostClient)
		if err != nil {
			logrus.WithError(err).Errorf("Error getting address utxos")
			errorOnCheck = true
			continue
		}
		logrus.Infof("Found %d utxos for address", len(utxos))

		// loop through them
		for _, utxo := range utxos {
			// have we seen this before?
			existingFight, err := s.Store.GetFightForUtxo(utxo.TxHash, utxo.OutputIndex)
			if err != nil {
				logrus.WithError(err).Errorf("Error getting fight for utxo and index")
				errorOnCheck = true
				break
			}

			if len(existingFight) == 1 {
				logrus.Infof("Found fight with id %d, nothing to do", existingFight[0].ID)
				continue
			}

			logrus.Infof("Utxo %s with index %d not seen before, check if valid for minting...", utxo.TxHash, utxo.OutputIndex)

			// make sure amount is lovelace
			// make sure valid
			if len(utxo.Amount) > 1 {
				logrus.Errorf("Utxo %s has more than 1 amount, ignoring...", utxo.TxHash)
				continue
			} else if utxo.Amount[0].Unit != "lovelace" { // make sure is lovelace
				logrus.WithError(err).Errorf("Utxo %s isn't lovelace", utxo.TxHash)
				continue
			}

			// convert to int
			utxoQuantity, err := strconv.Atoi(utxo.Amount[0].Quantity)
			if err != nil {
				logrus.WithError(err).Errorf("Error converting quantity")
				errorOnCheck = true
				break
			}

			// does it match an existing mint
			matchingFight, err := s.Store.GetFightForPaymentLastFifteen(int64(utxoQuantity))
			if err != nil {
				logrus.WithError(err).Errorf("Error finding payment for quantity %d", utxoQuantity)
				errorOnCheck = true
				break
			}

			if matchingFight == nil {
				logrus.Warnf("No matching fight found for utxo %s and amount %d, returning...", utxo.TxHash, utxoQuantity)
				if len(returns) < 10 {
					logrus.Infof("Returning utxo %s with quantity %d", utxo.TxHash, utxoQuantity)
					// find from address
					txUtxos, err := s.BlockfrostClient.TransactionUTXOs(context.Background(), utxo.TxHash)
					if err != nil {
						logrus.WithError(err).Errorf("Error getting transaction utxo %s", utxo.TxHash)
						errorOnCheck = true
						break
					}
					returnAddress := txUtxos.Inputs[0].Address

					returnNft := NFTReturn{
						FromUtxo:        fmt.Sprintf("%s#%d", utxo.TxHash, utxo.OutputIndex),
						FromUtxoAmount:  utxoQuantity,
						ReturnToAddress: returnAddress,
					}
					returns = append(returns, returnNft)
				}
				continue
			}

			logrus.Infof("Utxo %s is valid and we should mint for fight %d", utxo.TxHash, matchingFight.ID)

			// update alien and fight with utxo and fight status from PENDING to QUEUED in atomic tx
			err = s.moveFightFromPendingToQueued(*matchingFight, utxo.TxHash, utxo.OutputIndex)
			if err != nil {
				logrus.WithError(err).Errorf("Error moving fight to queued %d", matchingFight.ID)
				errorOnCheck = true
				break
			}
		}

		// loop over status of fights, mint a max of 10
		queuedFights, err := s.Store.GetQueuedFight()
		if err != nil {
			logrus.WithError(err).Errorf("Error getting queued fights")
			errorOnCheck = true
			break
		}

		i := 0
		logrus.Infof("Found %d queued fights", len(queuedFights))

		for _, fight := range queuedFights {
			if i == 10 {
				logrus.Info("Already minted 10")
				break
			}

			logrus.Infof("Minting fight for id %d", fight.ID)

			// create working directory
			dirName := "work/" + uuid.New().String()
			err := os.Mkdir(dirName, 0755)
			if err != nil {
				logrus.WithError(err).Errorf("Error creating directory")
				errorOnCheck = true
				break
			}

			// build alien image and upload to ipfs
			alien, err := s.Store.GetAlienByFightId(fight.ID)
			if err != nil {
				logrus.WithError(err).Errorf("Error getting alient for fight id %d", fight.ID)
				errorOnCheck = true
				break
			}
			logrus.Infof("Fight has alien %s", alien.Name)

			alienBytes, err := s.ImageBuilderClient.BuildAlien(imagebuilder.Alien{
				Background: alien.Background,
				Skin:       alien.Skin,
				Clothes:    alien.Clothes,
				Hat:        alien.Hat,
				Hand:       alien.Hand,
				Mouth:      alien.Mouth,
				Eyes:       alien.Eyes,
				Width:      640,
				Height:     640,
			})
			if err != nil {
				logrus.WithError(err).Errorf("Error building alien")
				errorOnCheck = true
				break
			}

			// write file
			err = os.WriteFile(dirName+"/alien.jpg", alienBytes, 0644)
			if err != nil {
				logrus.WithError(err).Errorf("Error writing alien file")
				errorOnCheck = true
				break
			}

			// add to ipfs
			alienIpfsResponse, err := s.NftStorageClient.IpfsAdd(dirName + "/alien.jpg")
			if err != nil {
				logrus.WithError(err).Errorf("Error uploading alien to ipfs")
				errorOnCheck = true
				break
			}
			logrus.Infof("Alien ipfs %s", alienIpfsResponse.Value.Pin.CID)

			// alienIpfsResponse, err := s.BlockforstIpfsClient.IpfsAdd(dirName + "/alien.jpg")
			// if err != nil {
			// 	logrus.WithError(err).Errorf("Error uploading alien to ipfs")
			// 	errorOnCheck = true
			// 	break
			// }
			// logrus.Infof("Updload alien to ipfs %s, now pinnings", alienIpfsResponse.Hash)
			// err = s.BlockforstIpfsClient.IpfsPin(alienIpfsResponse.Hash)
			// if err != nil {
			// 	logrus.WithError(err).Errorf("Error pinning alien to ipfs")
			// 	errorOnCheck = true
			// 	break
			// }

			// build fight image (random background, message) and upload to ipfs
			w, l, err := s.determineFightWinner(dirName, fight.ZombieName, fight.HunterName, fight.ID, alien.ID, alienIpfsResponse.Value.Pin.CID)
			if err != nil {
				logrus.WithError(err).Errorf("Error determining fight winner")
				errorOnCheck = true
				break
			}

			tweetId, err := twitter.TweetFight(dirName+"/alien.jpg", dirName+"/fight.jpg", fmt.Sprintf("%s defeated %s and revealed %s!", w, l, alien.ReadableName))
			if err != nil {
				//don't make this a real error, just fail silently
				logrus.WithError(err).Errorf("Error tweeting, failing silently...")
			} else {
				err = s.Store.UpdateTweetID(context.Background(), fight.ID, tweetId)
				if err != nil {
					logrus.WithError(err).Errorf("Error updating tweet id")
					errorOnCheck = true
					break
				}
			}

			// move images to backup dir
			err = moveImagesToBackup(dirName, alien.Name)
			if err != nil {
				logrus.WithError(err).Errorf("Error moving images")
				errorOnCheck = true
				break
			}

			//remove folder
			err = os.RemoveAll(dirName)
			if err != nil {
				logrus.WithError(err).Errorf("Error removing directory")
				errorOnCheck = true
				break
			}

			i++
		}

		// don't continue
		if errorOnCheck {
			continue
		}

		//find staged fights
		stagedFights, err := s.Store.GetStagedFights()
		if err != nil {
			logrus.WithError(err).Errorf("Error getting staged fights")
			errorOnCheck = true
			continue
		}

		i = 0
		logrus.Infof("Found %d staged fights", len(stagedFights))
		for _, fight := range stagedFights {
			if i == 10 {
				logrus.Info("Already minted 10")
				break
			}

			logrus.Infof("Minting fight for id %d", fight.ID)
			// call method to mint both fight and alien
			// build new dir
			dirName := "work/" + uuid.New().String()

			// find return address
			txUtxos, err := s.BlockfrostClient.TransactionUTXOs(context.Background(), fight.IncomingUtxo.String)
			if err != nil {
				logrus.WithError(err).Errorf("Error getting transaction utxo %s", fight.IncomingUtxo.String)
				errorOnCheck = true
				break
			}
			returnAddress := txUtxos.Inputs[0].Address

			//cando: should probably store this on first loop
			// utxoQuantity, err := strconv.Atoi(txUtxos.Outputs[0].Amount[0].Quantity)
			// if err != nil {
			// 	logrus.WithError(err).Errorf("Error converting quantity to int %s", txUtxos.Outputs[0].Amount[0].Quantity)
			// 	errorOnCheck = true
			// 	break
			// }

			// just use amount requested since already matches
			utxoQuantity := int(fight.PaymentAmountLovelace)

			//build metadata
			alien, err := s.Store.GetAlienByFightId(fight.ID)
			if err != nil {
				logrus.WithError(err).Errorf("Error getting alient for fight id %d", fight.ID)
				errorOnCheck = true
				break
			}
			alienMeta, err := buildAlienMetaString(*alien)
			if err != nil {
				logrus.WithError(err).Errorf("Error building alien meta for id %d", alien.ID)
				errorOnCheck = true
				break
			}

			alienNumberString := strings.Replace(alien.Name, "Alien", "", 1)
			fightNumber, err := strconv.Atoi(alienNumberString)
			if err != nil {
				logrus.WithError(err).Errorf("Error converting fight number %s", alienNumberString)
				errorOnCheck = true
				break
			}

			fightMeta, err := buildFightMetaString(fight, fightNumber)
			if err != nil {
				logrus.WithError(err).Errorf("Error building fight meta for id %d", fight.ID)
				errorOnCheck = true
				break
			}

			// determine splits
			txsOut := make([]string, 0)
			// if fight.ZombieSendAddress.Valid && fight.ZombieSendAddress.String != "" {
			// 	zombiePaymentAmount := fight.ZombieAmountAda * 1000000
			// 	txsOut = append(txsOut, fmt.Sprintf("%s+%d", fight.ZombieSendAddress.String, zombiePaymentAmount))
			// }
			// if fight.HunterSendAddress.Valid && fight.HunterSendAddress.String != "" {
			// 	hunterPaymentAmount := fight.HunterAmountAda * 1000000
			// 	txsOut = append(txsOut, fmt.Sprintf("%s+%d", fight.HunterSendAddress.String, hunterPaymentAmount))
			// }
			// split brian/royalty
			//TODO: determine if 12 is correct amount
			txsOut = append(txsOut, fmt.Sprintf("%s+%d", s.BrianSplitAddress, 5000000))
			//txsOut = append(txsOut, fmt.Sprintf("%s+%d", s.RoyaltySplitAddress, 4500000))

			alienSendAddress := ""
			if fight.ZombieLifeBar.Int64 > fight.HunterLifeBar.Int64 {
				alienSendAddress = fight.ZombieSendAddress.String
			} else {
				alienSendAddress = fight.HunterSendAddress.String
			}

			txHash, err := s.mintAlienAndZfcNfts(dirName, txsOut, s.RoyaltySplitAddress, returnAddress, fight.IncomingUtxo.String, int(fight.IncomingUtxoInt.Int64), utxoQuantity, s.ZfcPolicyID, s.AlienPolicyID, fightMeta, alienMeta, fmt.Sprintf("Fight%d", fightNumber), alien.Name, alienSendAddress)
			if err != nil {
				logrus.WithError(err).Errorf("Error minting, continueing...")
				errorOnCheck = true
				continue
			}

			// update tx
			logrus.Infof("Moving fight %d to minted for hash %s", fight.ID, txHash)
			err = s.Store.MoveFightFromStagedToMinted(context.Background(), fight.ID, txHash)
			if err != nil {
				logrus.WithError(err).Errorf("Error moving to minted")
				errorOnCheck = true
				break
			}

			//remove folder
			err = os.RemoveAll(dirName)
			if err != nil {
				logrus.WithError(err).Errorf("Error removing directory")
				errorOnCheck = true
				break
			}

		}

		// handle any returns
		refundCheck++
		if processRefunds && refundCheck == 10 {
			// check if any returns
			if len(returns) > 0 {
				logrus.Infof("Returning %d utxos...", len(returns))
				err = s.ReturnStuff(returns)
				if err != nil {
					logrus.WithError(err).Errorf("Error returning utxos")
				}
			}

			refundCheck = 0
		}

		// check and validate submitted txes
		unconfirmedFights, err := s.Store.GetMintedFights()
		if err != nil {
			logrus.WithError(err).Errorf("Error getting unconfirmed fights")
			errorOnCheck = true
			continue
		}
		logrus.Infof("Found %d unconfirmed fights", len(unconfirmedFights))

		for _, fight := range unconfirmedFights {
			txID := strings.ReplaceAll(fight.TxID.String, "\"", "")

			logrus.Infof("Getting tx hash %s", txID)
			transaction, err := s.BlockforstIpfsClient.GetTransaction(txID)
			if err != nil {
				logrus.WithError(err).Errorf("Error verifying tx...")
			}

			if transaction != nil {
				logrus.Infof("Transaction %s found", txID)
				err = s.Store.MoveFightFromMintedToConfirmed(context.Background(), fight.ID)
				if err != nil {
					logrus.WithError(err).Errorf("Error updating fight to confirmed")
					errorOnCheck = true
					continue
				}
			} else {
				logrus.Info("Tx not found")
			}
		}

		logrus.Info("Sleeping for 30 seconds")
		time.Sleep(30 * time.Second)
	}
}

func (s Server) moveFightFromPendingToQueued(fight store.FightDb, utxo string, utxoIndex int) error {
	// get next alien
	alien, err := s.Store.GetNextAvailableAlien()
	if err != nil {
		return err
	}

	logrus.Infof("Found alien %s for fight %d", alien.Name, fight.ID)

	// update alien fk and fight status
	logrus.Infof("Moving fight %d to alien %d", alien.ID, fight.ID)
	err = s.Store.MoveFightFromPendingToQueued(context.Background(), fight.ID, alien.ID, utxo, utxoIndex)
	if err != nil {
		return err
	}

	logrus.Infof("Fight %d is now QUEUED", fight.ID)

	return nil
}

func getAllUtxos(address string, client bfg.APIClient) ([]bfg.AddressUTXO, error) {
	allUtxos := make([]bfg.AddressUTXO, 0)

	page := 1
	for true {
		utxos, err := client.AddressUTXOs(context.Background(), address, bfg.APIQueryParams{Count: 100, Page: page})
		page++
		if err != nil {
			return nil, err
		}

		if len(utxos) == 0 {
			return allUtxos, nil
		}

		allUtxos = append(allUtxos, utxos...)
	}

	return allUtxos, nil
}

func (s Server) determineFightWinner(dirName string, zombieName string, hunterName string, fightId int, alienId int, alienIpfs string) (string, string, error) {
	// determine if this is right randomness
	zcStrength, zhStrength := metadata.FightZombieAndHunterReturnStrength(zombieName, hunterName, s.ZombieMetaStruct, s.HunterMetaStruct, s.ZombieChainTraitStrength, s.ZombieHunterTraitStrength, 120)

	// build fight image
	// {
	// 	"bacdkground": "Boxing-Ring",
	// 	"zombieChain": "ZombieChains00401",
	// 	"zombieHunter": "ZombieHunter05240",
	// 	"zhLifeBar": 0,
	// 	"zcLifeBar":60,
	// 	"vs": "VS",
	// 	"zombieRecord": "3-0",
	// 	"hunterRecord": "2-6",
	// 	"zombieKo": false,
	// 	"hunterKo": false,
	// 	"width": 1200,
	// 	"height": 675,
	// 	"zombieBeatup":false,
	// 	"hunterBeatup":true
	// }
	zombieFightImage := imagebuilder.ZombieFightImage{
		ZombieChain:  zombieName,
		ZombieHunter: hunterName,
		Vs:           "VS",
	}

	// zombie winner
	if zcStrength >= zhStrength {
		pointDifference := zcStrength - zhStrength
		winnerLb, loserLb, knockOut := metadata.DetermineLifeBar(pointDifference)
		zombieFightImage.ZombieChainLifeBar = winnerLb
		zombieFightImage.ZombieHunterLifeBar = loserLb
		if knockOut {
			zombieFightImage.HunterKO = true
		}
		if pointDifference > 20 {
			zombieFightImage.HunterBeatup = true
		}
	} else {
		pointDifference := zhStrength - zcStrength
		winnerLb, loserLb, knockOut := metadata.DetermineLifeBar(pointDifference)
		zombieFightImage.ZombieHunterLifeBar = winnerLb
		zombieFightImage.ZombieChainLifeBar = loserLb
		if knockOut {
			zombieFightImage.ZombieKO = true
		}
		if pointDifference > 20 {
			zombieFightImage.ZombieBeatup = true
		}
	}

	// determine record
	// lookup current record
	zombieNft, err := s.Store.GetNftByName(zombieName)
	if err != nil {
		return "", "", err
	}
	hunterNft, err := s.Store.GetNftByName(hunterName)
	if err != nil {
		return "", "", err
	}
	logrus.Infof("Current zombie record %d-%d, current hunter record %d-%d", zombieNft.Wins, zombieNft.Loses, hunterNft.Wins, hunterNft.Loses)
	winningNft := ""
	losingNft := ""

	//verify beatup works
	if zcStrength >= zhStrength {
		zombieFightImage.ZombieRecord = fmt.Sprintf("%03d-%03d", zombieNft.Wins+1, zombieNft.Loses)
		zombieFightImage.HunterRecord = fmt.Sprintf("%03d-%03d", hunterNft.Wins, hunterNft.Loses+1)
		winningNft = zombieName
		losingNft = hunterName
	} else {
		zombieFightImage.ZombieRecord = fmt.Sprintf("%03d-%03d", zombieNft.Wins, zombieNft.Loses+1)
		zombieFightImage.HunterRecord = fmt.Sprintf("%03d-%03d", hunterNft.Wins+1, hunterNft.Loses)
		winningNft = hunterName
		losingNft = zombieName
	}

	//TODO: record max?

	// build image and upload to ipfs
	fightBytes, zfcBackground, err := s.ImageBuilderClient.Buildfight(zombieFightImage)
	if err != nil {
		return "", "", err
	}

	// write file
	err = os.WriteFile(dirName+"/fight.jpg", fightBytes, 0644)
	if err != nil {
		return "", "", err
	}

	// add to ipfs
	// fightIpfsResponse, err := s.BlockforstIpfsClient.IpfsAdd(dirName + "/fight.jpg")
	// if err != nil {
	// 	return "", "", err
	// }
	fightIpfsResponse, err := s.NftStorageClient.IpfsAdd(dirName + "/fight.jpg")
	if err != nil {
		return "", "", err
	}
	logrus.Infof("Fight to ipfs %s", fightIpfsResponse.Value.Pin.CID)

	// logrus.Infof("Updload fight to ipfs %s, now pinnings", fightIpfsResponse.Hash)
	// err = s.BlockforstIpfsClient.IpfsPin(fightIpfsResponse.Hash)
	// if err != nil {
	// 	return "", "", err
	// }

	// update fight and record in one tx
	logrus.Infof("Moving fight %d form queued to staged", fightId)
	err = s.Store.MoveFightFromQueuedToStaged(context.Background(), alienId, alienIpfs, fightId, fightIpfsResponse.Value.Pin.CID, zfcBackground, zombieFightImage.ZombieRecord, zombieFightImage.HunterRecord, zombieFightImage.ZombieChainLifeBar, zombieFightImage.ZombieHunterLifeBar, zombieFightImage.ZombieKO, zombieFightImage.HunterKO, zombieFightImage.ZombieBeatup, zombieFightImage.HunterBeatup, winningNft, losingNft)
	if err != nil {
		return "", "", err
	}

	return winningNft, losingNft, nil
}

func (s Server) mintAlienAndZfcNfts(dirName string, baseTxsOut []string, royaltyAddress, toAddress string, fromUtxo string, fromUtxoIndex int, fromUtxoAmount int, zfcPolicyId string, alienPolicyId string, zfcMetaString string, alienMetaString string, fightName string, alienName string, alienSendAddress string) (string, error) {
	logrus.Infof("Minting nft to address %s from %s#%d with amount %d and alien to %s", toAddress, fromUtxo, fromUtxoIndex, fromUtxoAmount, alienSendAddress)

	//TODO: verify that this utxo is still valid

	// makedir for tx files
	err := os.Mkdir(dirName, 0755)
	if err != nil {
		logrus.WithError(err).Errorf("Error creating directory")
		return "", err
	}

	// build metadata file
	metadataFile := fmt.Sprintf("%s/metadata.json", dirName)
	f, err := os.Create(metadataFile)
	if err != nil {
		logrus.WithError(err).Errorf("Error creating meatdata file")
		return "", err
	}

	_, err = f.WriteString(fmt.Sprintf("{\"721\":{\"%s\":{%s},\"%s\":{%s}}}", zfcPolicyId, zfcMetaString, alienPolicyId, alienMetaString))
	f.Sync()
	f.Close()

	//build mint i.e. 1 056cd1282696748708db782c5af5f3ff1834a96b1cb198ab1e937af3.TestNFTName3
	mints := make([]string, 0)

	zfcMint := fmt.Sprintf("1 %s.%s", zfcPolicyId, fightName)
	mints = append(mints, zfcMint)

	alienMint := fmt.Sprintf("1 %s.%s", alienPolicyId, alienName)
	mints = append(mints, alienMint)

	// build tx in
	txsIn := make([]string, 0)
	txsIn = append(txsIn, fmt.Sprintf("%s#%d", fromUtxo, fromUtxoIndex))

	// build draft tx out and mints
	txsOut := make([]string, 0)
	txsOut = append(txsOut, baseTxsOut...)
	txsOut = append(txsOut, fmt.Sprintf("%s+%d", royaltyAddress, 4500000))

	//build send mint i.e. --tx-out addr_test1qqf3x5gu3g3c7p6dnw3qaqarhzs5rryp0t2qlf09mlvx2ugkjsr90u8d0qpqagp4lts2y7jq8s0c2z6dep2tmgr6s0wqnejpag+2000000+1 056cd1282696748708db782c5af5f3ff1834a96b1cb198ab1e937af3.TestNFTName3

	mintOutTx := fmt.Sprintf("%s+%d+%s", toAddress, 1250000, zfcMint)
	alienMintOutTx := fmt.Sprintf("%s+%d+%s", alienSendAddress, 1250000, alienMint)

	txsOut = append(txsOut, mintOutTx)
	txsOut = append(txsOut, alienMintOutTx)

	draftTxFile := fmt.Sprintf("%s/%s", dirName, "tx.draft")
	err = cli.BuildTransaction(draftTxFile, txsIn, txsOut, 0, 0, metadataFile, mints, "keys/zfc-policy.txt", "keys/alien-policy.txt")
	if err != nil {
		logrus.WithError(err).Errorf("Error building draft transaction")
		return "", err
	}
	fee, err := cli.CalculateFee(draftTxFile, len(txsIn), len(txsOut), 3)
	if err != nil {
		logrus.WithError(err).Errorf("Error calculating fee")
		return "", err
	}
	logrus.Infof("Calculated a fee of %d", fee)

	// build tx out again with fee
	txsOut = make([]string, 0)
	txsOut = append(txsOut, baseTxsOut...)
	txsOut = append(txsOut, fmt.Sprintf("%s+%d", royaltyAddress, fromUtxoAmount-7500000-fee))

	mintOutTx = fmt.Sprintf("%s+%d+%s", toAddress, 1250000, zfcMint)
	alienMintOutTx = fmt.Sprintf("%s+%d+%s", alienSendAddress, 1250000, alienMint)

	txsOut = append(txsOut, mintOutTx)
	txsOut = append(txsOut, alienMintOutTx)

	// old code
	//mintOutTx = fmt.Sprintf("%s+%d+%s+%s", toAddress, fromUtxoAmount-8000000-fee, zfcMint, alienMint) //TODO: need to change this if change min amount
	//txsOut = append(txsOut, mintOutTx)

	// get ttl
	block, err := s.BlockfrostClient.BlockLatest(context.Background())
	if err != nil {
		logrus.WithError(err).Errorf("Error getting latet block")
		return "", err
	}
	logrus.Infof("Found slot of %d", block.Slot)

	// build actual transaction
	actualTxFile := fmt.Sprintf("%s/%s", dirName, "mint.tx")

	err = cli.BuildTransaction(actualTxFile, txsIn, txsOut, block.Slot+1000, fee, metadataFile, mints, "keys/zfc-policy.txt", "keys/alien-policy.txt")
	if err != nil {
		logrus.WithError(err).Errorf("Error building transaction")
		return "", err
	}

	// sign file
	signedTxFile := fmt.Sprintf("%s/%s", dirName, "mint.signed")

	err = cli.SignTransaction(actualTxFile, "keys/payment.skey", "keys/zfc-mint.skey", "keys/alien-mint.skey", signedTxFile)
	if err != nil {
		logrus.WithError(err).Errorf("Error signing transaction")
		return "", err
	}

	// get cbor hex
	jsonFile, err := os.Open(signedTxFile)
	if err != nil {
		logrus.WithError(err).Errorf("Error opening file")
		return "", err
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var signedTx SignedTx
	json.Unmarshal(byteValue, &signedTx)

	//txHash := ""
	//txHex := ""

	txHex, err := s.BlockforstIpfsClient.SubmitTransaction(signedTx.Hex)
	logrus.Infof("Submitted tx: %s", txHex)
	if err != nil {
		logrus.WithError(err).Errorf("Error submitting tx")
		return "", echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return txHex, nil
}

func (s Server) ReturnStuff(returns []NFTReturn) error {
	logrus.Infof("Processing %d returns", len(returns))

	//build and submit tx
	dirName := "work/" + uuid.New().String()

	// makedir for tx files
	err := os.Mkdir(dirName, 0755)
	if err != nil {
		logrus.WithError(err).Errorf("Error creating directory")
		return err
	}

	txsIn := make([]string, 0)
	txsOut := make([]string, 0)
	for _, returnTx := range returns {
		//build txIn i.e. de44bc164500a05fa49d095e299d1a8b4d706fd971b22995f8cba60737ee5552#0
		txIn := returnTx.FromUtxo
		txsIn = append(txsIn, txIn)

		//build send mint i.e. --tx-out addr_test1qqf3x5gu3g3c7p6dnw3qaqarhzs5rryp0t2qlf09mlvx2ugkjsr90u8d0qpqagp4lts2y7jq8s0c2z6dep2tmgr6s0wqnejpag+2000000+1 056cd1282696748708db782c5af5f3ff1834a96b1cb198ab1e937af3.TestNFTName3
		returnTxOut := fmt.Sprintf("%s+%d", returnTx.ReturnToAddress, returnTx.FromUtxoAmount)
		txsOut = append(txsOut, returnTxOut)
	}

	draftTxFile := fmt.Sprintf("%s/%s", dirName, "tx.draft")
	err = cli.BuildTransaction(draftTxFile, txsIn, txsOut, 0, 0, "", nil, "", "")
	if err != nil {
		logrus.WithError(err).Errorf("Error building draft transaction")
		return err
	}
	fee, err := cli.CalculateFee(draftTxFile, len(txsIn), len(txsOut), 1)
	if err != nil {
		logrus.WithError(err).Errorf("Error calculating fee")
		return err
	}
	logrus.Infof("Calculated a fee of %d", fee)

	//incorporate fee
	txsIn = make([]string, 0)
	txsOut = make([]string, 0)
	for i, returnTx := range returns {
		//build txIn i.e. de44bc164500a05fa49d095e299d1a8b4d706fd971b22995f8cba60737ee5552#0
		txIn := returnTx.FromUtxo
		txsIn = append(txsIn, txIn)

		//build send mint i.e. --tx-out addr_test1qqf3x5gu3g3c7p6dnw3qaqarhzs5rryp0t2qlf09mlvx2ugkjsr90u8d0qpqagp4lts2y7jq8s0c2z6dep2tmgr6s0wqnejpag+2000000+1 056cd1282696748708db782c5af5f3ff1834a96b1cb198ab1e937af3.TestNFTName3
		returnAmount := returnTx.FromUtxoAmount
		if i == 0 {
			returnAmount = returnAmount - fee
		}

		returnTxOut := fmt.Sprintf("%s+%d", returnTx.ReturnToAddress, returnAmount)
		txsOut = append(txsOut, returnTxOut)
	}

	// get ttl
	block, err := s.BlockfrostClient.BlockLatest(context.Background())
	if err != nil {
		logrus.WithError(err).Errorf("Error getting latet block")
		return err
	}
	logrus.Infof("Found slot of %d", block.Slot)

	// build actual transaction
	actualTxFile := fmt.Sprintf("%s/%s", dirName, "mint.tx")

	//vaid for 1 hour
	err = cli.BuildTransaction(actualTxFile, txsIn, txsOut, block.Slot+10800, fee, "", nil, "", "")
	if err != nil {
		logrus.WithError(err).Errorf("Error building transaction")
		return err
	}

	// sign file
	signedTxFile := fmt.Sprintf("%s/%s", dirName, "return.signed")

	err = cli.SignTransaction(actualTxFile, "keys/payment.skey", "", "", signedTxFile)
	if err != nil {
		logrus.WithError(err).Errorf("Error signing transaction")
		return err
	}

	// get cbor hex
	jsonFile, err := os.Open(signedTxFile)
	if err != nil {
		logrus.WithError(err).Errorf("Error opening file")
		return err
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var signedTx SignedTx
	json.Unmarshal(byteValue, &signedTx)

	//submit transaction
	txHex, err := s.BlockforstIpfsClient.SubmitTransaction(signedTx.Hex)
	logrus.Infof("Submitted tx: %s", txHex)
	if err != nil {
		logrus.WithError(err).Errorf("Error submitting tx")
		return err
	}
	logrus.Infof("Submittted refunds with tx %s", txHex)

	jsonFile.Close()
	//remove folder
	err = os.RemoveAll(dirName)
	if err != nil {
		logrus.WithError(err).Errorf("Error removing directory")
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func buildAlienMetaString(alien store.Alien) (string, error) {
	typeMeta := "image/png"
	metaDto := NFTMetadata{
		Name: alien.ReadableName,
		//Image:      "ipfs://" + alien.Ipfs.String,
		Image:      buildIpfsImageMeta("ipfs://" + alien.Ipfs.String),
		Collection: &alien.Collection,
		Site:       &alien.Site,
		Twitter:    &alien.Twitter,
		Copyright:  &alien.Copyright,
		Type:       &typeMeta,
	}

	// remove dashes
	traits := make(map[string]string, 0)
	traits["Background"] = normalizeName(alien.Background)
	traits["Clothing"] = normalizeName(alien.Clothes)
	traits["Eyes"] = normalizeName(alien.Eyes)
	traits["Hat"] = normalizeName(alien.Hat)
	traits["Hand"] = normalizeName(alien.Hand)
	traits["Mouth"] = normalizeName(alien.Mouth)
	traits["Skin"] = normalizeName(alien.Skin)

	metaDto.Traits = traits

	metaDto.FileMetadata = make([]FileMetadata, 0)

	front := FileMetadata{
		MediaType: "image/png",
		//Src:       alien.Ipfs.String,
		Src:  buildIpfsImageMeta("ipfs://" + alien.Ipfs.String),
		Name: "Alien",
	}
	metaDto.FileMetadata = append(metaDto.FileMetadata, front)

	b, err := json.Marshal(metaDto)
	if err != nil {
		logrus.WithError(err).Errorf("Error converting meta to json")
		return "", err
	}
	metaString := fmt.Sprintf("\"%s\":%s", alien.Name, string(b))
	return metaString, nil
}

func normalizeName(name string) string {
	return strings.ReplaceAll(name, "-", "")
}

func buildIpfsImageMeta(image string) []string {
	imageArray := make([]string, 0)

	if len(image) > 64 {
		imageArray = append(imageArray, image[0:63])
		imageArray = append(imageArray, image[63:])
	} else {
		imageArray = append(imageArray, image)
	}

	return imageArray
}

func buildFightMetaString(fight store.FightDb, number int) (string, error) {
	typeMeta := "image/png"
	metaDto := NFTMetadata{
		Name: fmt.Sprintf("Fight #%d", number),
		//Image:      "ipfs://" + fight.IPFS.String,
		Image:      buildIpfsImageMeta("ipfs://" + fight.IPFS.String),
		Collection: &fight.Collection,
		Site:       &fight.Site,
		Twitter:    &fight.Twitter,
		Copyright:  &fight.Copyright,
		Type:       &typeMeta,
	}

	//add traits
	traits := make(map[string]string, 0)
	traits["Zombie"] = fight.ZombieName
	traits["Hunter"] = fight.HunterName
	traits["Background"] = normalizeName(fight.Background.String)
	traits["Zombie Life Bar"] = strconv.FormatInt(fight.ZombieLifeBar.Int64, 10)
	traits["Hunter Life Bar"] = strconv.FormatInt(fight.HunterLifeBar.Int64, 10)
	traits["Zombie Record"] = fight.ZombieRecord.String
	traits["Hunter Record"] = fight.HunterRecord.String
	traits["Zombie KO"] = strconv.FormatBool(fight.ZombieKo.Bool)
	traits["Hunter KO"] = strconv.FormatBool(fight.HunterKo.Bool)
	if fight.ZombieLifeBar.Int64 > fight.HunterLifeBar.Int64 {
		traits["Fight Winner"] = "Zombie"
	} else {
		traits["Fight Winner"] = "Hunter"
	}

	metaDto.Traits = traits

	metaDto.FileMetadata = make([]FileMetadata, 0)

	front := FileMetadata{
		MediaType: "image/png",
		//Src:       fight.IPFS.String,
		Src:  buildIpfsImageMeta("ipfs://" + fight.IPFS.String),
		Name: "Fight",
	}
	metaDto.FileMetadata = append(metaDto.FileMetadata, front)

	b, err := json.Marshal(metaDto)
	if err != nil {
		logrus.WithError(err).Errorf("Error converting meta to json")
		return "", err
	}
	metaString := fmt.Sprintf("\"%s\":%s", fmt.Sprintf("Fight%d", number), string(b))
	return metaString, nil
}

func moveImagesToBackup(dirName string, alienName string) error {
	pathToBackup := os.Getenv("BACKUP_IMAGE_PATH")

	err := moveFile(dirName+"/alien.jpg", pathToBackup+"/aliens/"+alienName+".jpg")
	if err != nil {
		return err
	}

	alienNumberString := strings.Replace(alienName, "Alien", "", 1)
	_, err = strconv.Atoi(alienNumberString)
	if err != nil {
		return err
	}

	err = moveFile(dirName+"/fight.jpg", pathToBackup+"/fights/fight"+alienNumberString+".jpg")
	if err != nil {
		return err
	}

	return nil
}

func moveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("Couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("Couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("Writing to output file failed: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("Failed removing original file: %s", err)
	}
	return nil
}

// PENDING > QUEUED > STAGED > MINTED > CONFIRMED > MINTED
