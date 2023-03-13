package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/reliablestaking/zombie-fight-club-server/blockfrost"
	db "github.com/reliablestaking/zombie-fight-club-server/db"
	"github.com/reliablestaking/zombie-fight-club-server/imagebuilder"
	"github.com/reliablestaking/zombie-fight-club-server/metadata"
	"github.com/reliablestaking/zombie-fight-club-server/nftkeyme"
	"github.com/reliablestaking/zombie-fight-club-server/nftstorage"

	"github.com/reliablestaking/zombie-fight-club-server/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	_ "github.com/lib/pq"

	bfg "github.com/blockfrost/blockfrost-go"
)

var mintCmd = &cobra.Command{
	Use:   "mint",
	Short: "Run minting engine",
	Long:  "Run minting engine",
	Run:   mint,
}

func init() {
	serveCmd.AddCommand(mintCmd)
}

func mint(cmd *cobra.Command, args []string) {
	// init database
	portInt := 5432
	port := os.Getenv("DB_PORT")
	if port != "" {
		portInt, _ = strconv.Atoi(port)
	}
	sslmode := "disable"
	if os.Getenv("DB_SSL") == "true" {
		sslmode = "require"
	}
	pgCon := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_ADDR"),
		portInt,
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_NAME"),
		sslmode)
	database, err := sqlx.Connect("postgres", pgCon)
	if err != nil {
		logrus.WithError(err).Fatal("Error connecting to db...")
	}
	defer database.Close()
	store := db.Store{
		Db: database,
	}

	nftkeymeOauthConfig := &oauth2.Config{
		RedirectURL:  os.Getenv("NFTKEYME_REDIRECT_URL"),
		ClientID:     os.Getenv("NFTKEYME_CLIENT_ID"),
		ClientSecret: os.Getenv("NFTKEYME_CLIENT_SECRET"),
		Scopes:       []string{"offline assets"},
		Endpoint: oauth2.Endpoint{
			TokenURL: os.Getenv("NFTKEYME_TOKEN_URL"),
			AuthURL:  os.Getenv("NFTKEYME_AUTH_URL"),
		},
	}

	paymentAddress := os.Getenv("PAYMENT_ADDRESS")
	if paymentAddress == "" {
		logrus.Fatalf("No payment address configured")
	}

	//init blockfrost
	clientOptions := bfg.APIClientOptions{}
	if os.Getenv("TESTNET") == "true" {
		clientOptions.Server = "https://cardano-preprod.blockfrost.io/api/v0/"
	}
	api := bfg.NewAPIClient(
		clientOptions,
	)

	// load zc trait strenght
	zcTraitStrength, err := metadata.LoadZombieChainsFightStrength("metadata")
	if err != nil {
		logrus.Fatalf("Error loaindg zc strength %v", err)
	}
	zhTraitStrength, err := metadata.LoadZombieHunterFightStrength("metadata")
	if err != nil {
		logrus.Fatalf("Error loaindg zc strength %v", err)
	}

	zcMeta, err := metadata.LoadZombieChainsMeta("metadata")
	if err != nil {
		logrus.Fatalf("Error loaindg zc meta %v", err)
	}
	logrus.Infof("Found %d zc meta", len(zcMeta))

	zhMeta, err := metadata.LoadZombieHunterMeta("metadata")
	if err != nil {
		logrus.Fatalf("Error loaindg zh meta %v", err)
	}
	logrus.Infof("Found %d zh meta", len(zhMeta))

	zfcPolicyID := os.Getenv("ZFC_POLICY_ID")
	if zfcPolicyID == "" {
		logrus.Fatal("No zfc policy id found")
	}
	alienPolicyID := os.Getenv("ALIEN_POLICY_ID")
	if alienPolicyID == "" {
		logrus.Fatal("No alien policy id found")
	}

	brianSplit := os.Getenv("BRIAN_SPLIT_ADDRESS")
	if brianSplit == "" {
		logrus.Fatal("No brian split found")
	}

	royaltySplit := os.Getenv("ROYALTY_SPLIT_ADDRESS")
	if royaltySplit == "" {
		logrus.Fatal("No royalty split found")
	}

	// init server
	server := server.Server{
		Sha1ver:                   sha1ver,
		BuildTime:                 buildTime,
		NftkeymeOauthConfig:       nftkeymeOauthConfig,
		Store:                     store,
		NftkeymeClient:            nftkeyme.NewClientFromEnvironment(),
		ImageBuilderClient:        imagebuilder.NewClientFromEnvironment(),
		BlockforstIpfsClient:      blockfrost.NewClientFromEnvironment(),
		PaymentAddress:            paymentAddress,
		BlockfrostClient:          api,
		ZombieMetaStruct:          zcMeta,
		HunterMetaStruct:          zhMeta,
		ZombieChainTraitStrength:  *zcTraitStrength,
		ZombieHunterTraitStrength: *zhTraitStrength,
		ZfcPolicyID:               zfcPolicyID,
		AlienPolicyID:             alienPolicyID,
		BrianSplitAddress:         brianSplit,
		RoyaltySplitAddress:       royaltySplit,
		NftStorageClient:          nftstorage.NewClientFromEnvironment(),
	}

	// start minter
	server.RunMintingEngine()
}
