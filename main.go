package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type OAuthAtlassianResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
	ExpiresIn        int    `json:"expires_in"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type OAuthAtlassianCode struct {
	GrantType    string `json:"grant_type"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
}

type AccessibleResources []struct {
	Id     string   `json:"id"`
	Url    string   `json:"url"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

var atlassianAuthURL = "https://auth.atlassian.com/oauth/token"
var atlassianAPIURL = "https://api.atlassian.com"
var userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_2) AppleWebKit/600.8.9 (KHTML, like Gecko)"
var port string
var clientId string
var clientSecret string
var redirectURI string
var certfile string
var keyfile string

var CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
var Usage = func() {
	fmt.Printf("Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {

	flag.StringVar(&port, "port", "443", "port to serve on")
	flag.StringVar(&clientId, "client-id", "", "Atlassian ClientID")
	flag.StringVar(&clientSecret, "client-secret", "", "Atlassian Client Secret")
	flag.StringVar(&redirectURI, "redirect-uri", "", "Atlassian Redirect URL")
	flag.StringVar(&certfile, "c", "", "path to cert file")
	flag.StringVar(&keyfile, "k", "", "path to key file")
	flag.Parse()

	// Wait for callback from client
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			fmt.Fprintf(os.Stdout, "could not parse query: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		url := r.RequestURI
		state := r.FormValue("state")
		code := r.FormValue("code")
		userAgent := r.UserAgent()
		log.Printf("%s\t%s\tstate=%s\n", userAgent, url, state)

		//Thanks and goodbye, user
		w.Header().Set("Location", "https://atlassian.com/")
		w.WriteHeader(http.StatusFound)

		exchangeCodeForJWT(code)
	})

	log.Printf("Starting HTTP server at %q", "0.0.0.0:"+port)
	if certfile != "" {
		log.Fatal(http.ListenAndServeTLS("0.0.0.0:"+port, certfile, keyfile, nil))
	}
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}

func exchangeCodeForJWT(code string) {
	log.Printf("Exchanging code for JWT")
	httpClient := http.Client{}

	// Exchange code for JWT token
	data := OAuthAtlassianCode{
		GrantType:    "authorization_code",
		ClientId:     clientId,
		ClientSecret: clientSecret,
		Code:         code,
		RedirectURI:  redirectURI,
	}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, atlassianAuthURL, &buf)
	if err != nil {
		log.Printf("could not create HTTP request: %v", err)
		return
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", userAgent)
	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("could not send HTTP request: %v", err)
		return
	}
	defer res.Body.Close()
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Exchange response\n\t%s\n", string(bytes))

	var jwt OAuthAtlassianResponse
	if err := json.Unmarshal(bytes, &jwt); err != nil {
		log.Printf("could not parse JSON response: %v", err)
		return
	}

	if jwt.AccessToken == "" {
		log.Printf("Missing access token in response")
		return
	}
	verifyAccessToken(jwt)
	getCloudId(jwt)
}

func verifyAccessToken(jwt OAuthAtlassianResponse) {
	me := "/me"
	bytes, _ := makeRequest(atlassianAPIURL, me, jwt)
	log.Printf("GET %s\n%s\n", me, string(bytes))
}

func getCloudId(jwt OAuthAtlassianResponse) {
	res := "/oauth/token/accessible-resources"
	bytes, _ := makeRequest(atlassianAPIURL, res, jwt)

	log.Printf("GET %s\n%s\n", res, string(bytes))

	var resources AccessibleResources
	if err := json.Unmarshal(bytes, &resources); err != nil {
		log.Printf("could not parse JSON response: %v", err)
		return
	}
	for _, r := range resources {
		s, _ := json.MarshalIndent(r, "", "\t")
		fmt.Println(string(s))
		fmt.Printf(`Carry on...
curl "%s/ex/confluence/%s/rest/api/search?cql=type=page&limit=1" \
--header 'Accept: application/json' \
--header 'Authorization: Bearer %s' \
`, atlassianAPIURL, r.Id, jwt.AccessToken)
	}
}

func makeRequest(atlassianAPIURL, endpoint string, jwt OAuthAtlassianResponse) ([]byte, error) {
	httpClient := http.Client{}

	uri := atlassianAPIURL + endpoint
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		log.Printf("could not create HTTP request: %v", err)
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt.AccessToken)
	req.Header.Set("user-agent", userAgent)
	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("could not send HTTP request: %v", err)
		return nil, err
	}
	defer res.Body.Close()
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return bytes, nil
}
