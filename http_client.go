package helpscout

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var ErrorRateLimit = errors.New("")
var ErrorUnauthorized = errors.New("")

type httpClient struct {
	http.Client
}

func newHttpClient() *httpClient {
	return &httpClient{
		http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).Dial,
				TLSHandshakeTimeout: 5 * time.Second,
			},
		},
	}
}

func (h *httpClient) doRequest(url string, method string,
	headers map[string]string, query *url.Values,
	reqData interface{}, respData interface{}) error {

	var err error
	var req *http.Request

	if reqData != nil {
		var jsonRaw []byte
		if jsonRaw, err = json.Marshal(reqData); err != nil {
			return errors.Wrap(err, "Unable to marshal request data")
		}

		reqDataBuffer := bytes.NewBuffer(jsonRaw)
		req, err = http.NewRequest(method, url, reqDataBuffer)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return errors.Wrap(err, "Unable to prepare new request")
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	if query != nil {
		req.URL.RawQuery = query.Encode()
	}

	response, err := h.Do(req)
	if err != nil {
		return errors.Wrap(err, "Unable to process request")
	}

	defer response.Body.Close()

	if response.StatusCode == 201 {
		return nil
	}

	if response.StatusCode != 200 {
		if response.StatusCode == 429 {
			return ErrorRateLimit
		}

		if response.StatusCode == 401 {
			return ErrorUnauthorized
		}

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return errors.Wrap(err, "Unable to read response body to decode error")
		}

		if len(body) != 0 {
			var errResp struct {
				Message  string `json:"message"`
				Embedded struct {
					Errors []struct {
						Path    string `json:"path"`
						Message string `json:"message"`
						Source  string `json:"source"`
					} `json:"errors"`
				} `json:"_embedded"`
			}

			if err := json.Unmarshal(body, &errResp); err != nil {
				return errors.Wrap(err, "Unable to parse error response-body as json")
			}

			return errors.Errorf("Remote server returned an error: %d [%+v]", response.StatusCode, errResp)
		}

		return errors.Errorf("Remote server returned an error: %d", response.StatusCode)
	}

	if !strings.Contains(response.Header.Get("Content-Type"), "application/json") &&
		!strings.Contains(response.Header.Get("Content-Type"), "application/hal+json") {
		return errors.Errorf("Remote server returned an invalid content type: %s",
			response.Header.Get("Content-Type"))
	}

	if respData == nil {
		return nil
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.Wrap(err, "Unable to read response body")
	}

	if err := json.Unmarshal(body, respData); err != nil {
		return errors.Wrap(err, "Unable to parse response-body as json")
	}

	return nil
}
