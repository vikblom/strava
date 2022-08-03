package strava

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
}

func (app *AppClient) HandleChart(w http.ResponseWriter, r *http.Request) {

	access, err := r.Cookie("access-token")
	if err != nil {
		// If not authenticated, draw an empty chart.
		cfg := DefaultConfig
		err = charts.WriteHeatmap(cfg, w)
		if err != nil {
			log.Errorf("write empty heatmap: %v", err)
		}
		return
	}

	// Query within current year.
	now := time.Now()
	from := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	to := time.Date(now.Year(), 12, 31, 0, 0, 0, 0, now.Location())

	counts, err := GetActivities(access.Value, from, to)
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
}

func (app *AppClient) AuthInitialRedirectURL(r *http.Request) *url.URL {
	authURL := &url.URL{
		Scheme: "https",
		Host:   STRAVA_URL,
		Path:   "/oauth/authorize",
	}
	q := authURL.Query()
	// Back to this handler.
	redirect := &url.URL{Scheme: "http", Host: r.Host, Path: r.URL.Path}
	q.Add("redirect_uri", redirect.String())
	q.Add("client_id", app.ID)
	q.Add("response_type", "code")
	q.Add("scope", "profile:read_all,activity:read_all")
	authURL.RawQuery = q.Encode()

	log.Infof("auth querying: %s", authURL.String())
	return authURL
}

func (app *AppClient) AuthRetrieveTokens(code string) (*oAuth2Response, error) {

	tokenURL := &url.URL{Scheme: "https", Host: STRAVA_URL, Path: "/oauth/token"}

	form := url.Values{}
	form.Add("client_id", app.ID)
	form.Add("client_secret", app.Secret)
	form.Add("grant_type", "authorization_code")
	form.Add("code", code)

	resp, err := http.PostForm(tokenURL.String(), form)
	if err != nil {
		return nil, fmt.Errorf("token POST: %w", err)
	}

	// decode
	var tokenResp oAuth2Response
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	if tokenResp.ExpiresIn < 60 {
		log.Error("token will expire in less than a minute")
	}

	return &tokenResp, nil
}

func (app *AppClient) HandleAuth(w http.ResponseWriter, r *http.Request) {

	// No tokens, do the auth dance, then redirect back to this handler to try again.
	q := r.URL.Query()
	if !q.Has("code") {
		// No "code" in URL means start fresh: redirect user to Strava for authentication.
		http.Redirect(w, r, app.AuthInitialRedirectURL(r).String(), http.StatusTemporaryRedirect)
		return
	}

	// Populated "code" from Strava redirect, use it to get actual tokens.
	// TODO: Double check scope.
	tokenResp, err := app.AuthRetrieveTokens(q.Get("code"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

	// Back to main page with tokens in cookies.
	http.Redirect(w, r, "..", http.StatusFound)
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
	log.Infof("API %s %s", method, req.URL.String())
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

// GetActivities exhaustively in the after/before range.
func GetActivities(token string, after, before time.Time) (map[string]int, error) {

	// The output is aggregated in this map.
	counts := make(map[string]int)

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	// Requests are paged since there is a limit on activities per response.
	page := 0
	const PER_PAGE = 100 // Activities per response page, 200 is maximum in the API spec.
	for {
		page++

		params := map[string]string{
			"page":     strconv.Itoa(page),
			"per_page": strconv.Itoa(PER_PAGE),
			"after":    strconv.Itoa(int(after.Unix())),
			"before":   strconv.Itoa(int(before.Unix())),
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
		log.Debugf("Got %d activities\n", len(activities))

		// Populate dict on form that charts expect.
		for _, v := range activities {
			key := v.StartDate.Format("2006-01-02")
			// Exclude outliers.
			if v.Seconds > 60*60*24 {
				continue
			}
			// Could be multiple activities on the same day.
			counts[key] += v.Seconds / 60
		}
		if len(activities) < PER_PAGE {
			break
		}
	}

	return counts, nil
}
