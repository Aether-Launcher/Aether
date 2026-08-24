package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"Aether/pkg/netutil"
	"github.com/pkg/browser"
	"github.com/zalando/go-keyring"
)

const (
	AzureClientID  = "b824b4ea-8fbd-4c59-8f7c-6156378f14b8"
	KeyringService = "AetherLauncher"
)

// AuthError provides structured error messages for UI presentation
type AuthError struct {
	Code    string
	Message string
	Err     error
}

func (e *AuthError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// doWithRetry executes an HTTP request with retries for transient network failures.
func doWithRetry(ctx context.Context, newReq func() (*http.Request, error)) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt-1) * 500 * time.Millisecond):
			}
		}
		req, err := newReq()
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !netutil.IsTransientNetworkError(err) {
			break
		}
	}
	return nil, lastErr
}

// PKCE Helper Functions
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// Token structs for API responses
type MicrosoftTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

type XboxLiveAuthResponse struct {
	IssueInstant  string                            `json:"IssueInstant"`
	NotAfter      string                            `json:"NotAfter"`
	Token         string                            `json:"Token"`
	DisplayClaims map[string][]map[string]string `json:"DisplayClaims"`
}

type XSTSAuthResponse struct {
	IssueInstant  string                            `json:"IssueInstant"`
	NotAfter      string                            `json:"NotAfter"`
	Token         string                            `json:"Token"`
	DisplayClaims map[string][]map[string]string `json:"DisplayClaims"`
	XErr          uint32                            `json:"XErr"`
	Message       string                            `json:"Message"`
}

type MinecraftAuthResponse struct {
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"`
}

type EntitlementItem struct {
	Name string `json:"name"`
}

type EntitlementsResponse struct {
	Items []EntitlementItem `json:"items"`
}

type MinecraftProfile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Skins []struct {
		ID      string `json:"id"`
		State   string `json:"state"`
		URL     string `json:"url"`
		Variant string `json:"variant"`
	} `json:"skins"`
}

// StartPKCEAuthFlow initiates the browser PKCE OAuth flow on a dynamic local port
func StartPKCEAuthFlow(ctx context.Context) (*Account, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, &AuthError{Code: "ERR_NET_LISTEN", Message: "Failed to open local port for authentication callback", Err: err}
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d", port)

	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, &AuthError{Code: "ERR_PKCE", Message: "Failed to generate PKCE verifier", Err: err}
	}
	challenge := generateCodeChallenge(verifier)

	authURL := fmt.Sprintf(
		"https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize?"+
			"client_id=%s&response_type=code&redirect_uri=%s&scope=%s&code_challenge=%s&code_challenge_method=S256&prompt=select_account",
		AzureClientID,
		url.QueryEscape(redirectURI),
		url.QueryEscape("XboxLive.signin offline_access"),
		challenge,
	)

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		errStr := r.URL.Query().Get("error")

		if errStr != "" {
			desc := r.URL.Query().Get("error_description")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "<html><body><h2>Authentication Cancelled or Failed</h2><p>%s</p></body></html>", desc)
			errChan <- &AuthError{Code: "ERR_AUTH_CANCELLED", Message: desc}
			return
		}

		if code != "" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "<html><body style='font-family:sans-serif;text-align:center;padding-top:50px;'>"+
				"<h2>Authentication Successful!</h2><p>You can close this tab and return to Aether Launcher.</p></body></html>")
			codeChan <- code
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Invalid callback request")
	})

	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	if err := browser.OpenURL(authURL); err != nil {
		fmt.Printf("Please open this link in your browser: %s\n", authURL)
	}

	select {
	case <-ctx.Done():
		server.Shutdown(context.Background())
		return nil, &AuthError{Code: "ERR_TIMEOUT", Message: "Authentication timed out"}
	case err := <-errChan:
		server.Shutdown(context.Background())
		return nil, err
	case code := <-codeChan:
		server.Shutdown(context.Background())
		return exchangeCodeAndLogin(ctx, code, verifier, redirectURI)
	}
}

func exchangeCodeAndLogin(ctx context.Context, code, verifier, redirectURI string) (*Account, error) {
	formData := url.Values{
		"client_id":     {AzureClientID},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}

	resp, err := doWithRetry(ctx, func() (*http.Request, error) {
		r, _ := http.NewRequestWithContext(ctx, "POST", "https://login.microsoftonline.com/consumers/oauth2/v2.0/token", strings.NewReader(formData.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return r, nil
	})
	if err != nil {
		if netutil.IsTransientNetworkError(err) {
			return nil, &AuthError{Code: "ERR_NET", Message: "No internet connection. Please check your network and try again, or use Offline Login.", Err: err}
		}
		return nil, &AuthError{Code: "ERR_NET", Message: "Failed to connect to Microsoft token server", Err: err}
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var msTokens MicrosoftTokenResponse
	if err := json.Unmarshal(bodyBytes, &msTokens); err != nil {
		return nil, &AuthError{Code: "ERR_JSON", Message: "Invalid Microsoft token response", Err: err}
	}

	if msTokens.Error != "" {
		return nil, &AuthError{Code: "ERR_MS_TOKEN", Message: msTokens.ErrorDesc}
	}

	return authenticateXboxAndMinecraft(ctx, msTokens.AccessToken, msTokens.RefreshToken)
}

func authenticateXboxAndMinecraft(ctx context.Context, msAccessToken, msRefreshToken string) (*Account, error) {
	// 1. Authenticate with Xbox Live
	xblReqBody := map[string]interface{}{
		"Properties": map[string]interface{}{
			"AuthMethod": "RPS",
			"SiteName":   "user.auth.xboxlive.com",
			"RpsTicket":  "d=" + msAccessToken,
		},
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    "JWT",
	}

	xblJSON, _ := json.Marshal(xblReqBody)
	resp, err := doWithRetry(ctx, func() (*http.Request, error) {
		r, _ := http.NewRequestWithContext(ctx, "POST", "https://user.auth.xboxlive.com/user/authenticate", bytes.NewBuffer(xblJSON))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json")
		return r, nil
	})
	if err != nil {
		if netutil.IsTransientNetworkError(err) {
			return nil, &AuthError{Code: "ERR_NET", Message: "No internet connection. Please check your network and try again.", Err: err}
		}
		return nil, &AuthError{Code: "ERR_NET", Message: "Failed to reach Xbox Live auth endpoint", Err: err}
	}
	defer resp.Body.Close()

	xblBytes, _ := io.ReadAll(resp.Body)
	var xblResp XboxLiveAuthResponse
	if err := json.Unmarshal(xblBytes, &xblResp); err != nil {
		return nil, &AuthError{Code: "ERR_XBL_PARSE", Message: "Failed to parse Xbox Live response", Err: err}
	}

	if xblResp.Token == "" || len(xblResp.DisplayClaims["xui"]) == 0 {
		return nil, &AuthError{Code: "ERR_XBL_EMPTY", Message: "Xbox Live authentication returned no token"}
	}

	userHash := xblResp.DisplayClaims["xui"][0]["uhs"]

	// 2. Authorize with XSTS
	xstsReqBody := map[string]interface{}{
		"Properties": map[string]interface{}{
			"SandboxId":  "RETAIL",
			"UserTokens": []string{xblResp.Token},
		},
		"RelyingParty": "rp://api.minecraftservices.com/",
		"TokenType":    "JWT",
	}

	xstsJSON, _ := json.Marshal(xstsReqBody)
	resp, err = doWithRetry(ctx, func() (*http.Request, error) {
		r, _ := http.NewRequestWithContext(ctx, "POST", "https://xsts.auth.xboxlive.com/xsts/authorize", bytes.NewBuffer(xstsJSON))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json")
		return r, nil
	})
	if err != nil {
		if netutil.IsTransientNetworkError(err) {
			return nil, &AuthError{Code: "ERR_NET", Message: "No internet connection. Please check your network and try again.", Err: err}
		}
		return nil, &AuthError{Code: "ERR_NET", Message: "Failed to reach XSTS authorize endpoint", Err: err}
	}
	defer resp.Body.Close()

	xstsBytes, _ := io.ReadAll(resp.Body)
	var xstsResp XSTSAuthResponse
	_ = json.Unmarshal(xstsBytes, &xstsResp)

	if xstsResp.XErr != 0 {
		switch xstsResp.XErr {
		case 2148916233:
			return nil, &AuthError{Code: "ERR_NO_XBOX", Message: "This Microsoft account has no Xbox account. Please create one at xbox.com."}
		case 2148916238:
			return nil, &AuthError{Code: "ERR_CHILD_ACCOUNT", Message: "Child account error. Please add this account to a Microsoft Family group."}
		case 2148916235:
			return nil, &AuthError{Code: "ERR_XBOX_RESTRICTED", Message: "Xbox Live is not available in your region."}
		default:
			return nil, &AuthError{Code: "ERR_XSTS_FAILED", Message: fmt.Sprintf("XSTS auth failed with error code %d", xstsResp.XErr)}
		}
	}

	// 3. Login to Minecraft
	identityToken := fmt.Sprintf("XBL3.0 x=%s;%s", userHash, xstsResp.Token)
	mcReqBody := map[string]string{"identityToken": identityToken}
	mcJSON, _ := json.Marshal(mcReqBody)

	resp, err = doWithRetry(ctx, func() (*http.Request, error) {
		r, _ := http.NewRequestWithContext(ctx, "POST", "https://api.minecraftservices.com/authentication/login_with_xbox", bytes.NewBuffer(mcJSON))
		r.Header.Set("Content-Type", "application/json")
		return r, nil
	})
	if err != nil {
		if netutil.IsTransientNetworkError(err) {
			return nil, &AuthError{Code: "ERR_NET", Message: "No internet connection. Please check your network and try again.", Err: err}
		}
		return nil, &AuthError{Code: "ERR_NET", Message: "Failed to reach Minecraft auth endpoint", Err: err}
	}
	defer resp.Body.Close()

	mcBytes, _ := io.ReadAll(resp.Body)
	var mcAuth MinecraftAuthResponse
	if err := json.Unmarshal(mcBytes, &mcAuth); err != nil || mcAuth.AccessToken == "" {
		return nil, &AuthError{Code: "ERR_MC_AUTH", Message: "Failed to authenticate with Minecraft services"}
	}

	// 4. Verify Minecraft Game Ownership
	resp, err = doWithRetry(ctx, func() (*http.Request, error) {
		r, _ := http.NewRequestWithContext(ctx, "GET", "https://api.minecraftservices.com/entitlements/mcstore", nil)
		r.Header.Set("Authorization", "Bearer "+mcAuth.AccessToken)
		return r, nil
	})
	if err != nil {
		if netutil.IsTransientNetworkError(err) {
			return nil, &AuthError{Code: "ERR_NET", Message: "No internet connection. Please check your network and try again.", Err: err}
		}
		return nil, &AuthError{Code: "ERR_NET", Message: "Failed to verify Minecraft entitlements", Err: err}
	}
	defer resp.Body.Close()

	entBytes, _ := io.ReadAll(resp.Body)
	var entResp EntitlementsResponse
	_ = json.Unmarshal(entBytes, &entResp)

	hasMinecraft := false
	for _, item := range entResp.Items {
		if item.Name == "product_minecraft" || item.Name == "game_minecraft" {
			hasMinecraft = true
			break
		}
	}

	if !hasMinecraft {
		return nil, &AuthError{Code: "ERR_NO_GAME", Message: "This Microsoft account does not own Minecraft Java Edition."}
	}

	// 5. Fetch Profile
	resp, err = doWithRetry(ctx, func() (*http.Request, error) {
		r, _ := http.NewRequestWithContext(ctx, "GET", "https://api.minecraftservices.com/minecraft/profile", nil)
		r.Header.Set("Authorization", "Bearer "+mcAuth.AccessToken)
		return r, nil
	})
	if err != nil {
		if netutil.IsTransientNetworkError(err) {
			return nil, &AuthError{Code: "ERR_NET", Message: "No internet connection. Please check your network and try again.", Err: err}
		}
		return nil, &AuthError{Code: "ERR_NET", Message: "Failed to fetch Minecraft profile", Err: err}
	}
	defer resp.Body.Close()

	profBytes, _ := io.ReadAll(resp.Body)
	var profile MinecraftProfile
	if err := json.Unmarshal(profBytes, &profile); err != nil || profile.ID == "" {
		return nil, &AuthError{Code: "ERR_PROFILE", Message: "Failed to load Minecraft profile details"}
	}

	// Formatted UUID with hyphens
	formattedUUID := profile.ID
	if len(profile.ID) == 32 {
		formattedUUID = fmt.Sprintf("%s-%s-%s-%s-%s", profile.ID[0:8], profile.ID[8:12], profile.ID[12:16], profile.ID[16:20], profile.ID[20:32])
	}

	account := &Account{
		ID:           formattedUUID,
		Type:         TypeMicrosoft,
		Username:     profile.Name,
		AccessToken:  mcAuth.AccessToken,
		RefreshToken: msRefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(mcAuth.ExpiresIn) * time.Second).Unix(),
	}

	// Securely save Refresh Token in OS Keyring
	if msRefreshToken != "" {
		_ = keyring.Set(KeyringService, account.ID, msRefreshToken)
	}

	return account, nil
}

// RefreshMicrosoftToken silently refreshes the Microsoft token chain
func RefreshMicrosoftToken(ctx context.Context, acc *Account) (*Account, error) {
	refreshToken := acc.RefreshToken
	if refreshToken == "" {
		// Try loading from keyring
		var err error
		refreshToken, err = keyring.Get(KeyringService, acc.ID)
		if err != nil || refreshToken == "" {
			return nil, &AuthError{Code: "ERR_NO_REFRESH", Message: "No refresh token available. Please sign in again."}
		}
	}

	formData := url.Values{
		"client_id":     {AzureClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	resp, err := doWithRetry(ctx, func() (*http.Request, error) {
		r, _ := http.NewRequestWithContext(ctx, "POST", "https://login.microsoftonline.com/consumers/oauth2/v2.0/token", strings.NewReader(formData.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return r, nil
	})
	if err != nil {
		if netutil.IsTransientNetworkError(err) {
			return nil, &AuthError{Code: "ERR_NET", Message: "No internet connection. Please check your network and try again.", Err: err}
		}
		return nil, &AuthError{Code: "ERR_NET", Message: "Failed to reach Microsoft refresh endpoint", Err: err}
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var msTokens MicrosoftTokenResponse
	if err := json.Unmarshal(bodyBytes, &msTokens); err != nil || msTokens.AccessToken == "" {
		return nil, &AuthError{Code: "ERR_REFRESH_FAILED", Message: "Refresh token expired or invalid. Please sign in again."}
	}

	newRefreshToken := msTokens.RefreshToken
	if newRefreshToken == "" {
		newRefreshToken = refreshToken
	}

	return authenticateXboxAndMinecraft(ctx, msTokens.AccessToken, newRefreshToken)
}
