package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)
type UserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

var oauthConfig *oauth2.Config
var CurrentToken *oauth2.Token
var CurrentUser *UserInfo

const tokenFile = "token.json"

func Init() {
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8085/callback",
		Scopes: []string{
			"openid",
			"email",
			"profile",
			"https://www.googleapis.com/auth/drive.appdata",
		},
		Endpoint: google.Endpoint,
	}
}

func Login() (*UserInfo, error) {
	codeCh := make(chan string, 1)
	errCh  := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Addr: ":8085", Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			return
		}
		fmt.Fprintf(w, `<html><body>
			<h2 style="font-family:sans-serif;text-align:center;margin-top:80px">
			✓ Login successful! You can close this tab.</h2>
			</body></html>`)
		codeCh <- code
	})

	go srv.ListenAndServe()

	url := oauthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	openBrowser(url)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		srv.Shutdown(context.Background())
		return nil, err
	}
	srv.Shutdown(context.Background())

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, err
	}
	CurrentToken = token
	saveToken(token)

	user, err := fetchUserInfo(token)
	if err != nil {
		return nil, err
	}
	CurrentUser = user
	return user, nil
}

func TryRestoreSession() {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return
	}
	CurrentToken = &token

	// Try to fetch user info with saved token
	user, err := fetchUserInfo(&token)
	if err != nil {
		// Token expired and can't refresh — clear it
		CurrentToken = nil
		os.Remove(tokenFile)
		return
	}
	CurrentUser = user
}

func Logout() {
	CurrentToken = nil
	CurrentUser = nil
	os.Remove(tokenFile)
}

func GetClient() *http.Client {
	if CurrentToken == nil {
		return nil
	}
	return oauthConfig.Client(context.Background(), CurrentToken)
}

func fetchUserInfo(token *oauth2.Token) (*UserInfo, error) {
	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func saveToken(token *oauth2.Token) {
	data, err := json.Marshal(token)
	if err != nil {
		return
	}
	os.WriteFile(tokenFile, data, 0600)
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}