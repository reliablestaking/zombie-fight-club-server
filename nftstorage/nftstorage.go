package nftstorage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type (
	// NftstorageClient struct to hold client
	NftstorageClient struct {
		HttpClient http.Client
		BaseUrl    string
		ApiKey     string
	}

	StoreResponse struct {
		Value StoreResponseValue `json:"value"`
	}

	StoreResponseValue struct {
		Pin StoreResponseValuePin `json:"pin"`
	}

	StoreResponseValuePin struct {
		CID string `json:"cid"`
	}
)

//NewClientFromEnvironment create new nftkeyme client using env vars
func NewClientFromEnvironment() NftstorageClient {
	httpClient := &http.Client{
		Timeout: time.Second * 300,
	}

	baseURL := os.Getenv("NFTSTORAGE_URL")
	key := os.Getenv("NFTSTORAGE_KEY")

	client := NftstorageClient{
		HttpClient: *httpClient,
		BaseUrl:    baseURL,
		ApiKey:     key,
	}

	return client
}

func (client NftstorageClient) IpfsAdd(filePath string) (*StoreResponse, error) {
	logrus.Infof("Uploading image with path %s", filePath)

	file := mustOpen(filePath)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/upload", client.BaseUrl), file)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", client.ApiKey))
	req.Header.Add("Accept", "application/json")
	//req.Header.Add("Content-Type", w.FormDataContentType())

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		logrus.Errorf("Error posting request", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logrus.Errorf("Error adding ipfs file with status code %d", resp.StatusCode)
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		logrus.Errorf("Response Body %s", string(bytes))

		return nil, fmt.Errorf("Error adding ipfs file with status code %d", resp.StatusCode)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ipfsResponse := StoreResponse{}
	err = json.Unmarshal(bytes, &ipfsResponse)
	if err != nil {
		return nil, err
	}

	return &ipfsResponse, nil
}

func mustOpen(f string) *os.File {
	r, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	return r
}
