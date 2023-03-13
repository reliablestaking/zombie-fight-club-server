package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	bfg "github.com/blockfrost/blockfrost-go"
	"github.com/jmoiron/sqlx"
	"github.com/reliablestaking/zombie-fight-club-server/blockfrost"
	db "github.com/reliablestaking/zombie-fight-club-server/db"
	"github.com/reliablestaking/zombie-fight-club-server/imagebuilder"
	"github.com/reliablestaking/zombie-fight-club-server/nftkeyme"

	"github.com/patrickmn/go-cache"
	"github.com/reliablestaking/zombie-fight-club-server/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	_ "github.com/lib/pq"
)

var (
	sha1ver   string // sha1 revision used to build the program
	buildTime string // when the executable was built
)

func Execute() {
	if err := serveCmd.Execute(); err != nil {
		logrus.WithError(err).Error("Error in serve")
		os.Exit(1)
	}
}

var serveCmd = &cobra.Command{
	Use:   "server",
	Short: "Run server",
	Long:  "Run server",
	Run:   serve,
}

func serve(cmd *cobra.Command, args []string) {
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

	zombiePolicyId := os.Getenv("ZOMBIE_POLICY_ID")
	if zombiePolicyId == "" {
		logrus.Fatal("No zombie policy id found")
	}

	hunterPolicyId := os.Getenv("HUNTER_POLICY_ID")
	if hunterPolicyId == "" {
		logrus.Fatal("No hunter policy id found")
	}

	baseCostString := os.Getenv("BASE_COST_ADA")
	baseCostInt, err := strconv.Atoi(baseCostString)
	if err != nil {
		logrus.WithError(err).Fatalf("Couldn't parse base cost string %s", baseCostString)
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

	cache := cache.New(30*time.Minute, 60*time.Minute)

	hydraClient, err := server.NewHydraClientFromEnv()
	if err != nil {
		logrus.WithError(err).Fatal("Error initializing hydra clients")
	}

	// init server
	server := server.Server{
		Sha1ver:              sha1ver,
		BuildTime:            buildTime,
		NftkeymeOauthConfig:  nftkeymeOauthConfig,
		Store:                store,
		NftkeymeClient:       nftkeyme.NewClientFromEnvironment(),
		ImageBuilderClient:   imagebuilder.NewClientFromEnvironment(),
		BlockforstIpfsClient: blockfrost.NewClientFromEnvironment(),
		ZombiePolicyId:       zombiePolicyId,
		HunterPolicyId:       hunterPolicyId,
		ZombieMeta:           loadZcCsv(),
		HunterMeta:           loadHunterCsv(),
		BaseCostAda:          baseCostInt,
		PaymentAddress:       paymentAddress,
		BlockfrostClient:     api,
		LeaderCache:          cache,
		HydraClient:          *hydraClient,
	}

	// start server
	server.Start()
}

func loadZcCsv() map[string]string {
	zcs := make(map[string]string)

	// open file
	f, err := os.Open("metadata/zombie-meta-final.csv")
	if err != nil {
		logrus.Fatal(err)
	}

	// remember to close the file at the end of the program
	defer f.Close()

	// read csv values using csv.Reader
	csvReader := csv.NewReader(f)
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		// do something with read line
		zcs[rec[0]] = rec[2]
	}

	return zcs
}

func loadHunterCsv() map[string]string {
	zcs := make(map[string]string)

	// open file
	f, err := os.Open("metadata/hunter-meta-final.csv")
	if err != nil {
		logrus.Fatal(err)
	}

	// remember to close the file at the end of the program
	defer f.Close()

	// read csv values using csv.Reader
	csvReader := csv.NewReader(f)
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		// do something with read line
		zcs[rec[0]] = rec[2]
	}

	return zcs
}
