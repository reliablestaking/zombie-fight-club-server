package blockfrost

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

type (
	BlockfrostClient struct {
		HttpClient http.Client
		IpfsUrl    string
		IpfsKey    string
		BaseUrl    string
		ApiKey     string
	}

	IpfsAddResponse struct {
		Name string `json:"name"`
		Hash string `json:"ipfs_hash"`
	}

	Address struct {
		Address  string `json:"address"`
		Quantity string `json:"quantity"`
	}

	Transaction struct {
	}
)

func NewClientFromEnvironment() BlockfrostClient {
	httpClient := &http.Client{
		Timeout: time.Second * 300,
	}

	//TODO: error if not found
	baseURL := os.Getenv("IPFS_URL")

	svcBaseUrl := os.Getenv("BLOCKFROST_URL")
	apiKey := os.Getenv("BLOCKFROST_PROJECT_ID")

	client := BlockfrostClient{
		HttpClient: *httpClient,
		IpfsUrl:    baseURL,
		IpfsKey:    os.Getenv("IPFS_KEY"),
		BaseUrl:    svcBaseUrl,
		ApiKey:     apiKey,
	}

	return client
}

func (client BlockfrostClient) IpfsAdd(filePath string) (*IpfsAddResponse, error) {
	var err error
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	var fw io.Writer
	file := mustOpen(filePath)
	fileName := filepath.Base(filePath)

	if fw, err = w.CreateFormFile("file", "alien.jpg"); err != nil {
		logrus.Errorf("Error creating form file", err)
		return nil, err
	}
	if _, err = io.Copy(fw, file); err != nil {
		logrus.Errorf("Error copying file", err)
		return nil, err
	}
	w.Close()

	logrus.Infof("Uploading image with name %s", fileName)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/ipfs/add", client.IpfsUrl), &b)
	req.Header.Add("project_id", client.IpfsKey)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", w.FormDataContentType())

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

	ipfsResponse := IpfsAddResponse{}
	err = json.Unmarshal(bytes, &ipfsResponse)
	if err != nil {
		return nil, err
	}

	return &ipfsResponse, nil
}

func (client BlockfrostClient) IpfsPin(hash string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/ipfs/pin/add/%s", client.IpfsUrl, hash), nil)
	req.Header.Add("project_id", client.IpfsKey)
	req.Header.Add("Accept", "application/json")

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		logrus.Errorf("Error posting request", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logrus.Errorf("Error pinning ipfs file with status code %d", resp.StatusCode)
		return fmt.Errorf("Error pinning ipfs file with status code %d", resp.StatusCode)
	}

	return nil
}

func mustOpen(f string) *os.File {
	r, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	return r
}

func (client BlockfrostClient) SubmitTransaction(cborHex string) (string, error) {
	logrus.Info("Submitting transaction")

	decoded, err := hex.DecodeString(cborHex)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/tx/submit", client.BaseUrl), bytes.NewBuffer(decoded))
	req.Header.Add("project_id", client.ApiKey)
	req.Header.Add("Content-Type", "application/cbor")

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		logrus.Errorf("Error posting request", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Error submitting tx %d with error %s", resp.StatusCode, string(bytes))
		return "", fmt.Errorf("Error submitting tx %d", resp.StatusCode)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (client BlockfrostClient) GetAddressesForAsset(asset string) ([]Address, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/assets/%s/addresses", client.BaseUrl, asset), nil)
	req.Header.Add("project_id", client.ApiKey)
	req.Header.Add("Accept", "application/json")

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error getting request")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode != 200 {
		logrus.Errorf("Error getting asset address %d", resp.StatusCode)
		return nil, fmt.Errorf("Error getting asset address %d", resp.StatusCode)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	address := make([]Address, 0)
	err = json.Unmarshal(bytes, &address)
	if err != nil {
		return nil, err
	}

	return address, nil
}

func (client BlockfrostClient) GetTransaction(tx string) (*Transaction, error) {
	logrus.Infof("Getting transaction: %s", tx)

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/txs/%s", client.BaseUrl, tx), nil)
	req.Header.Add("project_id", client.ApiKey)
	req.Header.Add("Accept", "application/json")

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		logrus.Errorf("Error posting request", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode != 200 {
		logrus.Errorf("Error getting utxos %d", resp.StatusCode)
		return nil, fmt.Errorf("Error getting utxos %d", resp.StatusCode)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	transaction := Transaction{}
	err = json.Unmarshal(bytes, &transaction)
	if err != nil {
		return nil, err
	}

	return &transaction, nil
}
