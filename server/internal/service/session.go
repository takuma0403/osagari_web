package service

import (
	"context"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// IssuedSession は発行したセッション。Token は `__session` Cookie に入れる値で、DB にはハッシュだけを保存している。
type IssuedSession struct {
	Token     string
	ExpiresAt time.Time
}

// issueSession はユーザーのセッション（30日有効）を発行して保存する（AUTHN-3, AUTHN-5）。
func issueSession(ctx context.Context, sessions SessionStore, userID string, now time.Time) (IssuedSession, error) {
	token, err := auth.GenerateToken()
	if err != nil {
		return IssuedSession{}, err
	}
	expiresAt := now.Add(auth.SessionTTL)
	err = sessions.Create(ctx, &domain.Session{
		UserID:    userID,
		TokenHash: auth.HashToken(token),
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return IssuedSession{}, err
	}
	return IssuedSession{Token: token, ExpiresAt: expiresAt}, nil
}

// trustDevice は、メールのコードで確認した端末を登録済みの端末にする（AUTHN-3）。deviceID が空なら何もしない。
func trustDevice(ctx context.Context, devices TrustedDeviceStore, userID, deviceID string, now time.Time) error {
	if deviceID == "" {
		return nil
	}
	return devices.Upsert(ctx, &domain.TrustedDevice{
		UserID:     userID,
		DeviceHash: auth.HashToken(deviceID),
		LastUsedAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
}
