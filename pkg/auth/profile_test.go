package auth

import (
	"context"
	"testing"
)

func TestUploadSkinValidation(t *testing.T) {
	// Empty
	if _, err := UploadSkin(context.Background(), nil, "classic"); err == nil {
		t.Fatal("expected empty skin to fail")
	}
	// Non-PNG
	if _, err := UploadSkin(context.Background(), []byte("not a png"), "classic"); err == nil {
		t.Fatal("expected non-PNG to fail")
	}
	// Oversize
	big := make([]byte, 5*1024*1024+1)
	copy(big, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	if _, err := UploadSkin(context.Background(), big, "classic"); err == nil {
		t.Fatal("expected oversize skin to fail")
	}
}

func TestEquipCapeValidation(t *testing.T) {
	if err := EquipCape(context.Background(), "  "); err == nil {
		t.Fatal("expected empty cape ID to fail")
	}
}

func TestActiveAccountPublicNoTokens(t *testing.T) {
	m := ActiveAccountPublic()
	if _, ok := m["accessToken"]; ok {
		t.Fatal("tokens must never be exposed")
	}
	if _, ok := m["refreshToken"]; ok {
		t.Fatal("tokens must never be exposed")
	}
}
