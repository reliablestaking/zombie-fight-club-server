package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ory/hydra-client-go/client/admin"
	db "github.com/reliablestaking/zombie-fight-club-server/db"

	bfg "github.com/blockfrost/blockfrost-go"
	"github.com/reliablestaking/zombie-fight-club-server/blockfrost"

	"github.com/reliablestaking/zombie-fight-club-server/metadata"
	"github.com/reliablestaking/zombie-fight-club-server/nftstorage"

	"github.com/reliablestaking/zombie-fight-club-server/imagebuilder"
	"github.com/reliablestaking/zombie-fight-club-server/nftkeyme"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/gorilla/sessions"
)

type (
	// Server struct
	Server struct {
		BuildTime                 string
		Sha1ver                   string
		NftkeymeOauthConfig       *oauth2.Config
		Store                     db.Store
		NftkeymeClient            nftkeyme.NftkeymeClient
		ZombiePolicyId            string
		HunterPolicyId            string
		ZombieMeta                map[string]string
		ZombieMetaStruct          map[string]metadata.ZombieChain
		ZombieChainTraitStrength  metadata.ZombieChainTraitStrength
		HunterMeta                map[string]string
		HunterMetaStruct          map[string]metadata.ZombieHunter
		ZombieHunterTraitStrength metadata.ZombieHunterTraitStrength
		BaseCostAda               int
		PaymentAddress            string
		BlockfrostClient          bfg.APIClient
		ImageBuilderClient        imagebuilder.ImageBuilderClient
		BlockforstIpfsClient      blockfrost.BlockfrostClient
		NftStorageClient          nftstorage.NftstorageClient
		ZfcPolicyID               string
		AlienPolicyID             string
		BrianSplitAddress         string
		RoyaltySplitAddress       string
		LeaderCache               *cache.Cache
		HydraClient               HydraClient
	}

	// Version struct
	Version struct {
		Sha       string `json:"sha"`
		BuildTime string `json:"buildTime"`
	}
)

// Start the server
func (s Server) Start() {
	logrus.Info("Starting server...")
	e := echo.New()

	allowedOriginsCsv := make([]string, 0)
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		allowedOriginsCsv = strings.Split(allowedOrigins, ",")
	}

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     allowedOriginsCsv,
		AllowMethods:     []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
		AllowCredentials: true,
	}))

	// better secret
	e.Use(session.Middleware(sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))))

	// login/auth
	e.POST("/login", s.Login) //accept auth code from server, exchange for access/refresh token, create session using gorilla session
	e.POST("/login/check", s.LoginCheck, s.CheckCookie)

	// general fight endpoints
	e.GET("/fightnfts", s.GetNftsToFight, s.CheckCookie)     // find nfts available to fight
	e.POST("/fights", s.CreateFight, s.CheckCookie)          // create a new fight
	e.GET("/fights/:fightId", s.GetFightById, s.CheckCookie) // get fight status

	// fights
	e.GET("/user/nfts", s.GetMyNfts, s.CheckCookie)                // get all of my zombies
	e.PUT("/user/nfts/:name", s.ListNftForFight, s.CheckCookie)    // list a new zombie
	e.DELETE("/user/nfts/:name", s.DeleteListedNft, s.CheckCookie) // delist a zombie
	e.GET("/user/fights", s.GetMyFights, s.CheckCookie)            // get my fights

	// leaderboard
	e.GET("/leaders", s.GetLeaders) // get leaderboards (most wins, most loses, etc...)

	// version endpoint
	e.GET("/version", s.GetVersion)

	port := os.Getenv("NFTKEYME_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}
	e.Logger.Fatal(e.Start(":" + port))
}

// GetVersion return build version info
func (s Server) GetVersion(c echo.Context) (err error) {
	version := Version{
		Sha:       s.Sha1ver,
		BuildTime: s.BuildTime,
	}

	return c.JSON(http.StatusOK, version)
}

// RenderError renders an error page
func (s Server) RenderError(errorMsg string, c echo.Context) error {
	errorEnd := struct {
		Error string
	}{
		Error: errorMsg,
	}
	return c.JSON(http.StatusInternalServerError, errorEnd)
}

// CheckCookie checks incoming cookie to verify authed
func (s Server) CheckCookie(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// check bearer token
		bearerToken := c.Request().Header.Get("Authorization")
		if bearerToken != "" && strings.HasPrefix(bearerToken, "Bearer ") {
			// get token
			token := strings.Split(bearerToken, " ")[1]

			// get token info
			// validate token
			tokenResponse, err := s.HydraClient.adminClient.Admin.IntrospectOAuth2Token(
				admin.NewIntrospectOAuth2TokenParams().WithToken(token),
			)
			if err != nil {
				logrus.WithContext(c.Request().Context()).WithError(err).Errorf("Error making introspect request")
				return echo.NewHTTPError(http.StatusInternalServerError, err)
			}

			if *tokenResponse.Payload.Active == false {
				logrus.WithContext(c.Request().Context()).WithError(err).Errorf("Token %s no longer active or invalid", token)
				return echo.NewHTTPError(http.StatusUnauthorized, err)
			}

			subject := tokenResponse.Payload.Sub

			// check if user exists
			// check if already exsts
			logrus.Infof("Checking if user already exsists in db %s", subject)
			nftkeyUser, err := s.Store.GetUserByNftkeyID(subject)
			if err != nil {
				logrus.WithError(err).Errorf("Error getting discord user %s", subject)
				return s.RenderError("Internal server error", c)
			}
			if nftkeyUser == nil {
				logrus.Infof("User not found, creating in db %s", subject)
				err = s.Store.InsertUser(subject, "", "")
				if err != nil {
					logrus.WithError(err).Errorf("Error persisting user %s", subject)
					return s.RenderError("Internal server error", c)
				}
				//get again so won't panic later
				nftkeyUser, err = s.Store.GetUserByNftkeyID(subject)
				if err != nil {
					logrus.WithError(err).Errorf("Error getting user after create %s", subject)
					return s.RenderError("Internal server error", c)
				}
			}

			c.Set("user", nftkeyUser)

			return next(c)
		} else {

			cookie := c.Request().Header.Get("Cookie")
			if cookie == "" {
				logrus.Warn("No cookie found, unatuhorized...")
				return echo.NewHTTPError(http.StatusUnauthorized)
			}

			logrus.Debugf("Found cookie %s", cookie)
			sess, err := session.Get("session", c)
			if err != nil {
				logrus.WithError(err).Error("Error getting session")
				return echo.NewHTTPError(http.StatusInternalServerError)
			}

			if sess == nil {
				return echo.NewHTTPError(http.StatusUnauthorized)
			}

			nftkeyUserID := sess.Values["nftId"].(string)
			if nftkeyUserID == "" {
				logrus.Error("No user id found")
				return echo.NewHTTPError(http.StatusUnauthorized)
			}

			user, err := s.Store.GetUserByNftkeyID(nftkeyUserID)
			if err != nil {
				logrus.WithError(err).Error("Error getting user")
				return echo.NewHTTPError(http.StatusInternalServerError)
			}

			c.Set("user", user)

			return next(c)
		}
	}
}
