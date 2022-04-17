package strava

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

// https://developers.strava.com/docs/getting-started

// One time setup:
// In a browser open
// http://www.strava.com/oauth/authorize?client_id=[REPLACE_WITH_YOUR_CLIENT_ID]&response_type=code&redirect_uri=http://localhost/exchange_token&approval_prompt=force&scope=profile:read_all,activity:read_all
// and grab code=??? from reply after approval.
// That code is AUTHORIZATIONCODE
// Then do
// curl -X POST https://www.strava.com/oauth/token \
// -F client_id=YOURCLIENTID \
// -F client_secret=YOURCLIENTSECRET \
// -F grant_type=authorization_code
// -F code=AUTHORIZATIONCODE \

// If tokens expired
// curl -X POST https://www.strava.com/oauth/token \
// -F client_id=YOURCLIENTID \
// -F client_secret=YOURCLIENTSECRET \
// -F grant_type=authorization_code
// -F refresh_token=REFRESHTOKEN \

// curl -X GET \
// https://www.strava.com/api/v3/athlete \
// -H 'Authorization: Bearer YOURACCESSTOKEN'

const URL = "www.strava.com"

type oAuth2Response struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	// other fields are not relevant yet.
}

// apiCall wraps the ...
func apiCall(method, path string, headers, params map[string]string) (*http.Response, error) {
	url := &url.URL{
		Scheme: "https",
		Host:   URL,
		Path:   path,
	}
	req, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	// request - let caller check error and defer close
	log.Debug(req.URL.String())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		log.Debug(string(raw))
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP request not ok: %s", resp.Status)
	}
	return resp, nil
}

func GetAccessToken(apikey string) (string, error) {
	secret := base64.URLEncoding.EncodeToString([]byte(apikey))
	headers := map[string]string{
		"Authorization": "Basic " + secret,
		"Content-Type":  "application/x-www-form-urlencoded",
	}

	params := map[string]string{
		"format":     "json",
		"grant_type": "client_credentials",
	}
	resp, err := apiCall(http.MethodPost, "token", headers, params)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// decode
	var authResp oAuth2Response
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	if err != nil {
		return "", err
	}
	if authResp.ExpiresIn < 60 {
		return "", errors.New("token will expire in less than a minute")
	}
	return authResp.AccessToken, nil
}

func GetActivities(token string) error {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	params := map[string]string{
		// "format":       "json",
		// "onlyRealtime": "yes",
	}
	resp, err := apiCall(http.MethodGet, "api/v3/activities", headers, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode JSON with some anon. structs.
	activities := []struct {
		Name      string
		Distance  float64
		StartDate time.Time `json:"start_date"` // UTC
	}{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&activities)
	if err != nil {
		return err
	}

	for _, v := range activities {
		log.Infof("%+v\n", v)
	}
	return nil
}
