package helpscout

import (
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

var ErrorInterrupted = errors.New("")

const helpscoutApiEndpoint = "https://api.helpscout.net/v2"

type Page struct {
	Size          uint `json:"size"`
	TotalElements uint `json:"totalElements"`
	TotalPages    uint `json:"totalPages"`
	Number        uint `json:"number"`
}

type generalListApiCallReq struct {
	Embedded interface{} `json:"_embedded"`
	Page     Page        `json:"page"`
}

type Client struct {
	httpClient *httpClient
	auth       *auth
}

func NewClient(appId string, appKey string) *Client {
	httpClient := newHttpClient()

	return &Client{
		httpClient: httpClient,
		auth:       newAuth(httpClient, appId, appKey),
	}
}

func (c *Client) AuthKey(forceUpdate bool) (string, error) {
	token, err := c.auth.getToken(forceUpdate)
	if err != nil {
		return "", errors.Wrap(err, "Unable to update Auth Token")
	}

	return token, nil
}

func (c *Client) SetAuthKey(key string, expTime time.Time) {
	c.auth.token = key
	c.auth.tokenExpireTime = expTime
}

func (c *Client) doApiCall(method string, resource string, query *url.Values,
	reqData interface{}, respData interface{}) error {

	repeatAllCnt := 0
	forceTokenUpdate := false
	for {
		token, err := c.auth.getToken(forceTokenUpdate)
		if err != nil {
			return errors.Wrap(err, "Unable to update Auth Token")
		}

		url := helpscoutApiEndpoint + resource

		authHeader := make(map[string]string)
		authHeader["Authorization"] = fmt.Sprintf("Bearer %s", token)

		repeatCnt := 0
		for {
			err := c.httpClient.doRequest(url, method, authHeader, query, reqData, respData)
			if err == ErrorRateLimit {
				time.Sleep(time.Second)
				repeatCnt++
				if repeatCnt > 10 {
					return errors.New("Unable to submit a request (rate-limit)")
				}

				continue
			}

			if err == ErrorUnauthorized {
				break
			}

			return err
		}

		forceTokenUpdate = true
		repeatAllCnt++
		if repeatAllCnt > 3 {
			return errors.New("Unable to submit a request (authorization failed)")
		}
	}
}
