package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// ── Constants ────────────────────────────────────────────────────────────────

const (
	keyringSvc  = "noti-app"
	keyringUser = "token"
	redirectURI = "http://localhost:8085/callback"
)

// ── Types ────────────────────────────────────────────────────────────────────

type UserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// ── Package state ────────────────────────────────────────────────────────────

var (
	oauthConfig  *oauth2.Config
	CurrentToken *oauth2.Token
	CurrentUser  *UserInfo
	mu           sync.RWMutex

	clientID     string
	clientSecret string
)

// ── Init ─────────────────────────────────────────────────────────────────────

func Init() {
	clientID = os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")

	if clientID == "" {
		panic("GOOGLE_CLIENT_ID is missing")
	}

	oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes: []string{
			"openid",
			"email",
			"profile",
			"https://www.googleapis.com/auth/drive.appdata",
		},
		Endpoint: google.Endpoint,
	}
}

// ── Login ────────────────────────────────────────────────────────────────────

func Login() (*UserInfo, error) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generate verifier: %w", err)
	}

	state, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	code, err := openBrowserAndWait(state, verifier)
	if err != nil {
		return nil, fmt.Errorf("browser auth: %w", err)
	}

	token, err := exchangeCode(code, verifier)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	if err := saveToken(token); err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}

	user, err := fetchUserInfo(token)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}

	mu.Lock()
	CurrentToken = token
	CurrentUser = user
	mu.Unlock()

	return user, nil
}

// openBrowserAndWait opens the Google auth URL in the system browser,
// starts a local server to catch the redirect, and returns the auth code.
func openBrowserAndWait(state, verifier string) (string, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	srv := &http.Server{Addr: ":8085", Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Validate state to prevent CSRF
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch — possible CSRF attack")
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errCh <- fmt.Errorf("google auth error: %s", errParam)
			http.Error(w, "auth error", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("missing code in callback")
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body style="font-family:sans-serif;text-align:center;margin-top:80px">
			<h2>&#10003; Login successful!</h2><p>You can close this tab.</p>
			</body></html>`)
		codeCh <- code
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	
	// REMOVED: defer srv.Shutdown(...)

	authURL := oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)

	fmt.Println("AUTH URL:", authURL)

	if err := openBrowser(authURL); err != nil {
		srv.Close() // Clean up if browser fails to launch
		return "", fmt.Errorf("open browser: %w", err)
	}

	select {
	case code := <-codeCh:
		srv.Close() // <--- FORCE CLOSE ON SUCCESS: Severs HTTP connection immediately
		return code, nil
	case err := <-errCh:
		srv.Close() // <--- FORCE CLOSE ON ERROR: Severs HTTP connection immediately
		return "", err
	}
}

// ── Session ───────────────────────────────────────────────────────────────────

// TryRestoreSession loads a saved token from the OS keychain and validates it.
func TryRestoreSession() {
	token, err := loadToken()
	if err != nil {
		return
	}

	user, err := fetchUserInfo(token)
	if err != nil {
		// Token is invalid or expired with no refresh token — clear it
		_ = deleteToken()
		return
	}

	mu.Lock()
	CurrentToken = token
	CurrentUser = user
	mu.Unlock()
}

func Logout() {
	mu.Lock()
	CurrentToken = nil
	CurrentUser = nil
	mu.Unlock()

	_ = deleteToken()
}

// GetClient returns an HTTP client that automatically refreshes the token.
func GetClient() *http.Client {
	mu.RLock()
	token := CurrentToken
	mu.RUnlock()

	if token == nil {
		return nil
	}
	return oauthConfig.Client(context.Background(), token)
}

// ── Token storage (OS keychain) ───────────────────────────────────────────────

func saveToken(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return keyring.Set(keyringSvc, keyringUser, string(data))
}

func loadToken() (*oauth2.Token, error) {
	data, err := keyring.Get(keyringSvc, keyringUser)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func deleteToken() error {
	return keyring.Delete(keyringSvc, keyringUser)
}

// ── User info ─────────────────────────────────────────────────────────────────

func fetchUserInfo(token *oauth2.Token) (*UserInfo, error) {
	req, err := http.NewRequest(
		"GET",
		"https://www.googleapis.com/oauth2/v2/userinfo",
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"Authorization",
		"Bearer "+token.AccessToken,
	)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Println("USERINFO STATUS:", resp.Status)

	var user UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// ── PKCE helpers ──────────────────────────────────────────────────────────────

// generateCodeVerifier creates a cryptographically secure 64-byte random string.
// 64 bytes → 86 base64 chars, well within the 43–128 char PKCE spec.
func generateCodeVerifier() (string, error) {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ── Browser ───────────────────────────────────────────────────────────────────

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	// Detach from parent process so closing the app doesn't kill the browser
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // reap the child process
	return nil
}

func exchangeCode(code, verifier string) (*oauth2.Token, error) {
	token, err := oauthConfig.Exchange(
		context.Background(),
		code,
		oauth2.VerifierOption(verifier),
	)

	if err != nil {
		if rErr, ok := err.(*oauth2.RetrieveError); ok {
			fmt.Println("STATUS:", rErr.Response.Status)
			fmt.Println("BODY:", string(rErr.Body))
		}

		return nil, err
	}

	return token, nil
}