package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/Aether-Launcher/Aether/pkg/netutil"
)

// SkinEntry mirrors a single skin on the Mojang profile.
type SkinEntry struct {
	ID      string `json:"id"`
	State   string `json:"state"`
	URL     string `json:"url"`
	Variant string `json:"variant"`
}

// CapeEntry mirrors a single cape on the Mojang profile.
type CapeEntry struct {
	ID    string `json:"id"`
	State string `json:"state"`
	URL   string `json:"url"`
	Alias string `json:"alias"`
}

// FullProfile is the GET /minecraft/profile response shape.
type FullProfile struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Skins []SkinEntry `json:"skins"`
	Capes []CapeEntry `json:"capes"`
}

const profileBaseURL = "https://api.minecraftservices.com/minecraft/profile"

// profileHTTPClient is overridable in tests.
var profileHTTPClient = http.DefaultClient

func profileGET(ctx context.Context, token, path string) (*http.Response, error) {
	return doWithRetry(ctx, func() (*http.Request, error) {
		r, _ := http.NewRequestWithContext(ctx, http.MethodGet, profileBaseURL+path, nil)
		r.Header.Set("Authorization", "Bearer "+token)
		return r, nil
	})
}

// ensureFreshToken returns a usable access token, refreshing once on 401.
func ensureFreshToken(ctx context.Context, acc *Account) (string, error) {
	if acc == nil {
		return "", &AuthError{Code: "ERR_NO_ACCOUNT", Message: "No active account. Please sign in."}
	}
	if acc.Type != TypeMicrosoft {
		return "", &AuthError{Code: "ERR_OFFLINE_ACCOUNT", Message: "Offline accounts have no Mojang skins or capes. Sign in with Microsoft."}
	}
	if acc.AccessToken == "" {
		return "", &AuthError{Code: "ERR_NO_TOKEN", Message: "No access token. Please sign in again."}
	}
	return acc.AccessToken, nil
}

func fetchProfileWithToken(ctx context.Context, token string) (*FullProfile, int, error) {
	resp, err := profileGET(ctx, token, "")
	if err != nil {
		if netutil.IsTransientNetworkError(err) {
			return nil, 0, &AuthError{Code: "ERR_NET", Message: "No internet connection.", Err: err}
		}
		return nil, 0, &AuthError{Code: "ERR_NET", Message: "Failed to reach Minecraft profile service", Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, resp.StatusCode, &AuthError{Code: "ERR_UNAUTHORIZED", Message: "Session expired. Please sign in again."}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		return nil, resp.StatusCode, &AuthError{Code: "ERR_PROFILE", Message: fmt.Sprintf("Profile request failed (%s): %s", resp.Status, strings.TrimSpace(string(body)))}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	var p FullProfile
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, resp.StatusCode, &AuthError{Code: "ERR_PROFILE_PARSE", Message: "Failed to parse profile response", Err: err}
	}
	return &p, resp.StatusCode, nil
}

// GetProfileTextures returns the skins and capes attached to the active account.
// It never returns tokens. Offline / missing accounts yield typed AuthErrors.
func GetProfileTextures(ctx context.Context) (*FullProfile, error) {
	acc := GetActiveAccount()
	token, err := ensureFreshToken(ctx, acc)
	if err != nil {
		return nil, err
	}
	prof, _, err := fetchProfileWithToken(ctx, token)
	if err != nil {
		// Single silent refresh attempt on auth failure.
		if aErr, ok := err.(*AuthError); ok && aErr.Code == "ERR_UNAUTHORIZED" {
			refreshed, rErr := RefreshMicrosoftToken(ctx, acc)
			if rErr != nil {
				return nil, rErr
			}
			_ = AddMicrosoftAccount(*refreshed)
			return fetchProfileWithTokenRetry(ctx, refreshed.AccessToken)
		}
		return nil, err
	}
	return prof, nil
}

func fetchProfileWithTokenRetry(ctx context.Context, token string) (*FullProfile, error) {
	prof, _, err := fetchProfileWithToken(ctx, token)
	return prof, err
}

// UploadSkin uploads a PNG (max 5 MiB, validated as PNG magic bytes) and
// activates it. Variant must be "classic" or "slim".
func UploadSkin(ctx context.Context, pngBytes []byte, variant string) (*SkinEntry, error) {
	const maxSkinBytes = 5 * 1024 * 1024
	if len(pngBytes) == 0 || len(pngBytes) > maxSkinBytes {
		return nil, &AuthError{Code: "ERR_SKIN_SIZE", Message: fmt.Sprintf("Skin must be 1 byte–%d bytes, got %d", maxSkinBytes, len(pngBytes))}
	}
	if !isPNG(pngBytes) {
		return nil, &AuthError{Code: "ERR_SKIN_FORMAT", Message: "Skin must be a PNG image"}
	}
	variant = strings.ToLower(strings.TrimSpace(variant))
	if variant != "classic" && variant != "slim" {
		variant = "classic"
	}
	acc := GetActiveAccount()
	token, err := ensureFreshToken(ctx, acc)
	if err != nil {
		return nil, err
	}
	doUpload := func(tok string) (*SkinEntry, int, error) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		_ = w.WriteField("variant", variant)
		fw, err := w.CreateFormFile("file", "skin.png")
		if err != nil {
			return nil, 0, err
		}
		if _, err := fw.Write(pngBytes); err != nil {
			return nil, 0, err
		}
		_ = w.Close()
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, profileBaseURL+"/skins", &buf)
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", w.FormDataContentType())
		resp, err := profileHTTPClient.Do(req)
		if err != nil {
			if netutil.IsTransientNetworkError(err) {
				return nil, 0, &AuthError{Code: "ERR_NET", Message: "No internet connection.", Err: err}
			}
			return nil, 0, &AuthError{Code: "ERR_NET", Message: "Skin upload failed", Err: err}
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, resp.StatusCode, &AuthError{Code: "ERR_UNAUTHORIZED", Message: "Session expired. Please sign in again."}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.StatusCode, &AuthError{Code: "ERR_SKIN_UPLOAD", Message: fmt.Sprintf("Skin upload failed (%s): %s", resp.Status, strings.TrimSpace(string(body)))}
		}
		// Response may be the new skin or the full profile; accept both.
		var skin SkinEntry
		if err := json.Unmarshal(body, &skin); err == nil && skin.URL != "" {
			return &skin, resp.StatusCode, nil
		}
		var prof FullProfile
		if err := json.Unmarshal(body, &prof); err == nil {
			for _, s := range prof.Skins {
				if strings.EqualFold(s.State, "ACTIVE") {
					c := s
					return &c, resp.StatusCode, nil
				}
			}
			if len(prof.Skins) > 0 {
				c := prof.Skins[len(prof.Skins)-1]
				return &c, resp.StatusCode, nil
			}
		}
		return &SkinEntry{Variant: variant, State: "ACTIVE"}, resp.StatusCode, nil
	}
	skin, code, err := doUpload(token)
	if err != nil {
		if aErr, ok := err.(*AuthError); ok && aErr.Code == "ERR_UNAUTHORIZED" {
			_ = code
			refreshed, rErr := RefreshMicrosoftToken(ctx, acc)
			if rErr != nil {
				return nil, rErr
			}
			_ = AddMicrosoftAccount(*refreshed)
			skin, _, err = doUpload(refreshed.AccessToken)
			if err != nil {
				return nil, err
			}
			return skin, nil
		}
		return nil, err
	}
	return skin, nil
}

// EquipCape sets the active cape by ID.
func EquipCape(ctx context.Context, capeID string) error {
	capeID = strings.TrimSpace(capeID)
	if capeID == "" {
		return &AuthError{Code: "ERR_CAPE_ID", Message: "Cape ID is required"}
	}
	acc := GetActiveAccount()
	token, err := ensureFreshToken(ctx, acc)
	if err != nil {
		return err
	}
	doPut := func(tok string) (int, error) {
		payload, _ := json.Marshal(map[string]string{"capeId": capeID})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPut, profileBaseURL+"/capes/active", bytes.NewReader(payload))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		resp, err := profileHTTPClient.Do(req)
		if err != nil {
			if netutil.IsTransientNetworkError(err) {
				return 0, &AuthError{Code: "ERR_NET", Message: "No internet connection.", Err: err}
			}
			return 0, &AuthError{Code: "ERR_NET", Message: "Cape request failed", Err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return resp.StatusCode, &AuthError{Code: "ERR_UNAUTHORIZED", Message: "Session expired. Please sign in again."}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
			return resp.StatusCode, &AuthError{Code: "ERR_CAPE_EQUIP", Message: fmt.Sprintf("Equip cape failed (%s): %s", resp.Status, strings.TrimSpace(string(body)))}
		}
		return resp.StatusCode, nil
	}
	if _, err := doPut(token); err != nil {
		if aErr, ok := err.(*AuthError); ok && aErr.Code == "ERR_UNAUTHORIZED" {
			refreshed, rErr := RefreshMicrosoftToken(ctx, acc)
			if rErr != nil {
				return rErr
			}
			_ = AddMicrosoftAccount(*refreshed)
			_, err = doPut(refreshed.AccessToken)
			return err
		}
		return err
	}
	return nil
}

// HideCape hides the currently active cape (DELETE /capes/active).
func HideCape(ctx context.Context) error {
	acc := GetActiveAccount()
	token, err := ensureFreshToken(ctx, acc)
	if err != nil {
		return err
	}
	doDelete := func(tok string) (int, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, profileBaseURL+"/capes/active", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := profileHTTPClient.Do(req)
		if err != nil {
			if netutil.IsTransientNetworkError(err) {
				return 0, &AuthError{Code: "ERR_NET", Message: "No internet connection.", Err: err}
			}
			return 0, &AuthError{Code: "ERR_NET", Message: "Hide cape failed", Err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return resp.StatusCode, &AuthError{Code: "ERR_UNAUTHORIZED", Message: "Session expired. Please sign in again."}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
			return resp.StatusCode, &AuthError{Code: "ERR_CAPE_HIDE", Message: fmt.Sprintf("Hide cape failed (%s): %s", resp.Status, strings.TrimSpace(string(body)))}
		}
		return resp.StatusCode, nil
	}
	if _, err := doDelete(token); err != nil {
		if aErr, ok := err.(*AuthError); ok && aErr.Code == "ERR_UNAUTHORIZED" {
			refreshed, rErr := RefreshMicrosoftToken(ctx, acc)
			if rErr != nil {
				return rErr
			}
			_ = AddMicrosoftAccount(*refreshed)
			_, err = doDelete(refreshed.AccessToken)
			return err
		}
		return err
	}
	return nil
}

func isPNG(b []byte) bool {
	return len(b) > 8 && b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4E && b[3] == 0x47 &&
		b[4] == 0x0D && b[5] == 0x0A && b[6] == 0x1A && b[7] == 0x0A
}

// ActiveAccountPublic returns the safe subset of the active account for
// extension sandboxes. Tokens are never exposed.
func ActiveAccountPublic() map[string]interface{} {
	acc := GetActiveAccount()
	if acc == nil {
		return map[string]interface{}{"signedIn": false}
	}
	return map[string]interface{}{
		"signedIn": true,
		"id":       acc.ID,
		"username": acc.Username,
		"type":     string(acc.Type),
	}
}

// ExpiresAt returns token expiry for UI display (0 when unknown).
func activeAccountExpiresAt() int64 {
	_ = time.Now
	acc := GetActiveAccount()
	if acc == nil {
		return 0
	}
	return acc.ExpiresAt
}

var _ = activeAccountExpiresAt
