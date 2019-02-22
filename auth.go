package helpscout

import (
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const helpscoutAuthEndpoint = "https://api.helpscout.net/v2/oauth2/token"

type authReqData struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
}

type auth struct {
	httpClient      *httpClient
	token           string
	tokenExpireTime time.Time
	appId           string
	appKey          string
}

func newAuth(httpClient *httpClient, appId string, appKey string) *auth {
	return &auth{
		httpClient:      httpClient,
		appId:           appId,
		appKey:          appKey,
		token:           "",
		tokenExpireTime: time.Time{},
	}
}

func (a *auth) getToken(forceUpdate bool) (string, error) {

	/* token exists and still valid */
	if !forceUpdate && a.token != "" && a.tokenExpireTime.After(time.Now().Add((10 * time.Minute))) {
		return a.token, nil
	}

	reqData := authReqData{
		ClientId:     a.appId,
		ClientSecret: a.appKey,
		GrantType:    "client_credentials",
	}

	var responseJson struct {
		ExpiresIn int    `json:"expires_in"`
		Token     string `json:"access_token"`
		TokenType string `json:"token_type"`
	}

	repeatCnt := 0
	for {
		err := a.httpClient.doRequest(helpscoutAuthEndpoint, http.MethodPost, nil, nil, &reqData, &responseJson)
		if err == ErrorRateLimit {
			time.Sleep(time.Second)
			repeatCnt++
			if repeatCnt > 10 {
				return "", errors.New("Unable to submit auth-token update request (rate-limit)")
			}

			continue
		}

		if err == ErrorUnauthorized {
			return "", errors.Wrap(err, "Unable to submit auth-token update request (authorization failed)")
		}

		if err != nil {
			return "", errors.Wrap(err, "Unable to submit auth-token update request")
		}

		break
	}

	if responseJson.Token == "" || responseJson.ExpiresIn <= 0 {
		return "", errors.Errorf("Authorization server returned an invalid data: %+v", responseJson)
	}

	a.token = responseJson.Token
	a.tokenExpireTime = time.Now().Add(time.Second * time.Duration(responseJson.ExpiresIn))

	return a.token, nil
}
