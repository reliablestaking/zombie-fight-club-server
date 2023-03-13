package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/dghubble/oauth1"
	"github.com/sirupsen/logrus"

	twitterv2 "github.com/g8rswimmer/go-twitter/v2"
)

type MediaUpload struct {
	MediaId int64 `json:"media_id"`
}

type authorizer struct{}

func (a *authorizer) Add(req *http.Request) {}

func TweetFight(alienPath string, fightPath string, tweetText string) (string, error) {
	rKey := os.Getenv("RESOURCE_KEY")
	rSecret := os.Getenv("RESOURCE_SECRET")
	tokenKey := os.Getenv("TOKEN_KEY")
	tokenSecret := os.Getenv("TOKEN_SECRET")

	// authenticate
	config := oauth1.NewConfig(rKey, rSecret)
	token := oauth1.NewToken(tokenKey, tokenSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	//client := twitter.NewClient(httpClient)
	client := &twitterv2.Client{
		Authorizer: &authorizer{},
		Client:     httpClient,
		Host:       "https://api.twitter.com",
	}

	alienMediaId, err := uploadMedia(httpClient, alienPath)
	if err != nil {
		logrus.Warn("Error uploading alien to twitter")
		return "", err
	}

	fightMediaId, err := uploadMedia(httpClient, fightPath)
	if err != nil {
		logrus.Warn("Error uploading fight to twitter")
		return "", err
	}

	logrus.Infof("Tweeting '%s' with ids of %d and %d", tweetText, alienMediaId, fightMediaId)

	mediaIds := make([]string, 0)
	mediaIds = append(mediaIds, strconv.FormatInt(alienMediaId, 10))
	mediaIds = append(mediaIds, strconv.FormatInt(fightMediaId, 10))

	req := twitterv2.CreateTweetRequest{
		Text:  tweetText,
		Media: &twitterv2.CreateTweetMedia{IDs: mediaIds},
	}

	tweetResp, err := client.CreateTweet(context.Background(), req)
	if err != nil {
		logrus.WithError(err).Error("Error tweeting")
		return "", err
	}

	return tweetResp.Tweet.ID, nil
}

func uploadMedia(httpClient *http.Client, path string) (int64, error) {
	// create body form
	b := &bytes.Buffer{}
	form := multipart.NewWriter(b)

	// create media paramater
	fw, err := form.CreateFormFile("media", "file.jpg")
	if err != nil {
		return 0, err
	}

	// open file
	opened, err := os.Open(path)
	if err != nil {
		return 0, err
	}

	// copy to form
	_, err = io.Copy(fw, opened)
	if err != nil {
		return 0, err
	}

	// close form
	form.Close()

	// upload media
	resp, err := httpClient.Post("https://upload.twitter.com/1.1/media/upload.json?media_category=tweet_image", form.FormDataContentType(), bytes.NewReader(b.Bytes()))
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	defer resp.Body.Close()

	// decode response and get media id
	m := &MediaUpload{}
	err = json.NewDecoder(resp.Body).Decode(m)
	if err != nil {
		return 0, err
	}
	//mid := strconv.Itoa(m.MediaId)
	return m.MediaId, nil
}
