package strava

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/nikolaydubina/calendarheatmap/charts"
	log "github.com/sirupsen/logrus"
)

// https://developers.strava.com/docs/getting-started

// One time setup:
// In a browser open
// https://www.strava.com/oauth/authorize?client_id=[REPLACE_WITH_YOUR_CLIENT_ID]&response_type=code&redirect_uri=http://localhost/exchange_token&approval_prompt=force&scope=profile:read_all,activity:read_all
// and grab code=??? from reply after approval.
// That code is AUTHORIZATIONCODE
// Then do
// curl -X POST https://www.strava.com/oauth/token \
// -F client_id=YOURCLIENTID \
// -F client_secret=YOURCLIENTSECRET \
// -F grant_type=authorization_code \
// -F code=AUTHORIZATIONCODE

// If tokens expired
// curl -X POST https://www.strava.com/oauth/token \
// -F client_id=YOURCLIENTID \
// -F client_secret=YOURCLIENTSECRET \
// -F grant_type=refresh_token \
// -F refresh_token=REFRESHTOKEN \

// curl -X GET \
// https://www.strava.com/api/v3/athlete \
// -H 'Authorization: Bearer YOURACCESSTOKEN'

const (
	STRAVA_URL = "www.strava.com"
)

type oAuth2Response struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	// other fields are not relevant yet.
}

type AppClient struct {
	ID     string
	Secret string
	URL    string
}

func (app *AppClient) WriteChart(w http.ResponseWriter, r *http.Request) {

}

func (app *AppClient) HandleAuthApproval(w http.ResponseWriter, r *http.Request) {

	srvAddr := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	log.Infof("srv addr: %v", srvAddr.String())
	log.Infof("is tls: %v", r.TLS == nil)
	log.Infof("uri: %v", r.RequestURI)

	access, err := r.Cookie("access-token")
	if err != nil {
		log.Errorf("err: %v", err)
	} else {
		log.Infof("access: %v", access.Value)

		// TMP HACK
		counts, err := GetActivities(access.Value)
		if err != nil {
			log.Errorf("GetActivities: %v", err)
			return
		}

		cfg := DefaultConfig
		cfg.Counts = counts

		err = charts.WriteHeatmap(cfg, w)
		if err != nil {
			log.Errorf("WriteHeatmap: %v", err)
		}
		return
	}

	q := r.URL.Query()
	if q.Has("code") {
		log.Infof("oauth code: %v", q.Get("code"))
		// TODO: Double check scope.

		tokenURL := &url.URL{Scheme: "https", Host: STRAVA_URL, Path: "/oauth/token"}

		form := url.Values{}
		form.Add("client_id", app.ID)
		form.Add("client_secret", "00738b12bcc2ccd283a7aa11902aa8fe1a722514") // FIXME
		form.Add("grant_type", "authorization_code")
		form.Add("code", q.Get("code"))

		resp, err := http.PostForm(tokenURL.String(), form)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		_ = resp
		log.Info(resp)

		// decode
		var tokenResp oAuth2Response
		err = json.NewDecoder(resp.Body).Decode(&tokenResp)
		if err != nil {
			log.Error(err.Error())
		}
		if tokenResp.ExpiresIn < 60 {
			log.Error("token will expire in less than a minute")
		}
		log.Infof("access token: %s", tokenResp.AccessToken)
		log.Infof("refresh token: %s", tokenResp.RefreshToken)

		// TODO: Use gorilla securecookie or similar.
		http.SetCookie(w, &http.Cookie{
			Name:     "access-token",
			Value:    tokenResp.AccessToken,
			HttpOnly: true,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh-token",
			Value:    tokenResp.RefreshToken,
			HttpOnly: true,
		})

		// TMP HACK
		counts, err := GetActivities(tokenResp.AccessToken)
		if err != nil {
			log.Errorf("GetActivities: %v", err)
			return
		}

		cfg := DefaultConfig
		cfg.Counts = counts

		err = charts.WriteHeatmap(cfg, w)
		if err != nil {
			log.Errorf("WriteHeatmap: %v", err)
		}
		return

	} else {
		log.Infof("auth will redirect back to: %s", app.URL)

		authURL := &url.URL{
			Scheme: "https",
			Host:   STRAVA_URL,
			Path:   "/oauth/authorize",
		}
		q := authURL.Query()
		q.Add("redirect_uri", app.URL) // Back to this handler.
		q.Add("client_id", app.ID)
		q.Add("response_type", "code")
		//q.Add("approval_prompt", "force")
		q.Add("scope", "profile:read_all,activity:read_all")
		authURL.RawQuery = q.Encode()

		log.Infof("auth querying: %s", authURL.String())
		http.Redirect(w, r, authURL.String(), http.StatusTemporaryRedirect)
		// Will populate http://localhost:8080/foo?state=&code=692842f838edcad616a957a2eac80945bb97cae1&scope=read,activity:read_all,profile:read_all
	}
}

// apiCall wraps the ...
func apiCall(method, path string, headers, params map[string]string) (*http.Response, error) {
	url := &url.URL{
		Scheme: "https",
		Host:   STRAVA_URL,
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
	log.Info(req.URL.String())
	log.Info(req.Header)
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

func GetActivities(token string) (map[string]int, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	params := map[string]string{
		"per_page": "100",
	}
	resp, err := apiCall(http.MethodGet, "/api/v3/athlete/activities", headers, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decode JSON with some anon. structs.
	activities := []struct {
		Name string
		// Avoid elapsed_time since a training can be paused
		// and resumed hours later.
		Seconds   int `json:"moving_time"`
		Distance  float64
		StartDate time.Time `json:"start_date"` // UTC
		Type      string
	}{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&activities)
	if err != nil {
		return nil, err
	}

	// Populate dict on form that charts expect.
	counts := make(map[string]int, len(activities))
	for _, v := range activities {
		log.Debugf("%+v\n", v)

		key := v.StartDate.Format("2006-01-02")
		pre := counts[key]
		counts[key] = pre + v.Seconds/60
	}
	return counts, nil
}
