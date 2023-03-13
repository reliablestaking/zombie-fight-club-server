package server

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

//Login accept auth code from server, exchange for access/refresh token, create session using gorilla session
func (s Server) Login(c echo.Context) (err error) {
	logrus.Infof("Logging in with auth code")

	authCode := c.QueryParam("code")
	state := c.QueryParam("state")
	logrus.Infof("Handling auth code from nftkeyme with state id %s", state)

	//exchange code for token
	token, err := s.NftkeymeOauthConfig.Exchange(oauth2.NoContext, authCode)
	if err != nil {
		logrus.WithError(err).Error("Error exchange code for token")
		return s.RenderError("Internal server error", c)
	}

	// get user
	nftkeymeUser, err := s.NftkeymeClient.GetUserInfo(token.AccessToken)
	if err != nil {
		logrus.WithError(err).Errorf("Error getting nftkeyme info %s", state)
		return s.RenderError("Internal server error", c)
	}

	// check if already exsts
	logrus.Infof("Checking if user already exsists in db %s", nftkeymeUser.ID)
	nftkeyUser, err := s.Store.GetUserByNftkeyID(nftkeymeUser.ID)
	if err != nil {
		logrus.WithError(err).Errorf("Error getting discord user %s", state)
		return s.RenderError("Internal server error", c)
	}
	if nftkeyUser == nil {
		logrus.Infof("User not found, creating in db %s", nftkeymeUser.ID)
		err = s.Store.InsertUser(nftkeymeUser.ID, token.AccessToken, token.RefreshToken)
		if err != nil {
			logrus.WithError(err).Errorf("Error persisting user %s", state)
			return s.RenderError("Internal server error", c)
		}
		//get again so won't panic later
		nftkeyUser, err = s.Store.GetUserByNftkeyID(nftkeymeUser.ID)
		if err != nil {
			logrus.WithError(err).Errorf("Error getting user after create %s", state)
			return s.RenderError("Internal server error", c)
		}
	} else {
		logrus.Infof("User found, updating in db %s", nftkeymeUser.ID)
		err = s.Store.UpdatedUser(nftkeyUser.NftkeymeID, token.AccessToken, token.RefreshToken)
		if err != nil {
			logrus.WithError(err).Errorf("Error updating user %s", state)
			return s.RenderError("Internal server error", c)
		}
	}

	sess, err := session.Get("session", c)
	if err != nil {
		logrus.WithError(err).Error("Error getting session")
		return s.RenderError("Error getting session", c)
	}

	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		//TODO: configurable?
		//Secure:   true,
		//SameSite: http.SameSiteNoneMode,
	}

	sess.Values = make(map[interface{}]interface{})
	//TODO: fix panic
	sess.Values["nftId"] = nftkeyUser.NftkeymeID
	sess.Save(c.Request(), c.Response())

	return c.JSON(http.StatusOK, nil)
}

func (s Server) LoginCheck(c echo.Context) (err error) {
	return c.JSON(http.StatusOK, nil)
}
