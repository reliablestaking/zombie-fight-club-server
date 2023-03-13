package imagebuilder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type (
	ImageBuilderClient struct {
		HttpClient http.Client
		BaseUrl    string
	}

	Alien struct {
		Background string `json:"background"`
		Skin       string `json:"skin"`
		Clothes    string `json:"clothes"`
		Hat        string `json:"hat"`
		Hand       string `json:"hand"`
		Mouth      string `json:"mouth"`
		Eyes       string `json:"eyes"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
	}

	ZombieFightImage struct {
		Background          string `json:"background"`
		ZombieChain         string `json:"zombieChain"`
		ZombieChainLifeBar  int    `json:"zcLifeBar"`
		ZombieHunter        string `json:"zombieHunter"`
		ZombieHunterLifeBar int    `json:"zhLifeBar"`
		Vs                  string `json:"vs"`
		ZombieRecord        string `json:"zombieRecord"`
		HunterRecord        string `json:"hunterRecord"`
		ZombieKO            bool   `json:"zombieKo"`
		ZombieBeatup        bool   `json:"zombieBeatup"`
		HunterKO            bool   `json:"hunterKo"`
		HunterBeatup        bool   `json:"hunterBeatup"`
		Width               int    `json:"width"`
		Height              int    `json:"height"`
	}
)

func NewClientFromEnvironment() ImageBuilderClient {
	httpClient := &http.Client{
		Timeout: time.Second * 300,
	}

	baseURL := os.Getenv("ZFC_IMAGE_BUILDER")

	client := ImageBuilderClient{
		HttpClient: *httpClient,
		BaseUrl:    baseURL,
	}

	return client
}

func (client ImageBuilderClient) BuildAlien(alien Alien) ([]byte, error) {
	logrus.Infof("Building alien %v", alien)

	body, err := json.Marshal(alien)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/images/alien", client.BaseUrl), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		logrus.WithError(err).Errorf("Error posting request")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		logrus.Errorf("Error building alien %d", resp.StatusCode)
		return nil, fmt.Errorf("Error building alien %d", resp.StatusCode)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (client ImageBuilderClient) Buildfight(fight ZombieFightImage) ([]byte, string, error) {
	logrus.Infof("Building fight %v", fight)

	body, err := json.Marshal(fight)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/images/zombiefight", client.BaseUrl), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		logrus.WithError(err).Errorf("Error posting request")
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		logrus.Errorf("Error building fight %d", resp.StatusCode)
		return nil, "", fmt.Errorf("Error building fight %d", resp.StatusCode)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return bytes, resp.Header.Get("Background"), nil
}
