// app_profile.go - Wails bindings for Mojang profile skins/capes.

package main

import (
	"encoding/base64"
	"strings"

	"github.com/Aether-Launcher/Aether/pkg/auth"
)

// GetProfileSkins returns skins attached to the active Microsoft account.
func (a *App) GetProfileSkins() ([]auth.SkinEntry, error) {
	prof, err := auth.GetProfileTextures(a.ctx)
	if err != nil {
		return nil, err
	}
	return prof.Skins, nil
}

// GetProfileCapes returns capes attached to the active Microsoft account.
func (a *App) GetProfileCapes() ([]auth.CapeEntry, error) {
	prof, err := auth.GetProfileTextures(a.ctx)
	if err != nil {
		return nil, err
	}
	return prof.Capes, nil
}

// UploadProfileSkin uploads a base64 PNG as the active skin.
func (a *App) UploadProfileSkin(base64Data, variant string) (auth.SkinEntry, error) {
	raw, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		if idx := strings.Index(base64Data, "base64,"); idx >= 0 {
			raw, err = base64.StdEncoding.DecodeString(base64Data[idx+len("base64,"):])
		}
		if err != nil {
			return auth.SkinEntry{}, err
		}
	}
	skin, err := auth.UploadSkin(a.ctx, raw, variant)
	if err != nil {
		return auth.SkinEntry{}, err
	}
	return *skin, nil
}

// EquipProfileCape sets the active cape.
func (a *App) EquipProfileCape(capeID string) error {
	return auth.EquipCape(a.ctx, capeID)
}

// HideProfileCape hides the active cape.
func (a *App) HideProfileCape() error {
	return auth.HideCape(a.ctx)
}
