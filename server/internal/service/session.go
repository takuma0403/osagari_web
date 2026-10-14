package service

import (
	"context"
	"errors"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
)

// IssuedSession は発行したセッション。Token は `__session` Cookie に入れる値で、DB にはハッシュだけを保存している。
type IssuedSession struct {
	Token     string
	IssuedAt  time.Time
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
	return IssuedSession{Token: token, IssuedAt: now, ExpiresAt: expiresAt}, nil
}

// deviceHashOf は端末識別子（X-Device-Id）の SHA-256 を返す。識別子が空なら空（ADR 0006）。
func deviceHashOf(deviceID string) string {
	if deviceID == "" {
		return ""
	}
	return auth.HashToken(deviceID)
}

// trustDevice は端末を登録済みの端末にする（AUTHN-3）。登録済みなら最終利用日時だけを更新する。deviceHash が空なら何もしない。
func trustDevice(ctx context.Context, devices TrustedDeviceStore, userID, deviceHash string, now time.Time) error {
	if deviceHash == "" {
		return nil
	}
	return devices.Upsert(ctx, &domain.TrustedDevice{
		UserID:     userID,
		DeviceHash: deviceHash,
		LastUsedAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
}

var errUnauthenticated = apperr.Unauthorized("ログインしてください")

// SessionDeps は SessionService の依存。
type SessionDeps struct {
	Sessions SessionStore
	Users    UserStore
	Clock    clock.Clock
}

// SessionService は `__session` のセッションの検証とログアウトを行う（AUTHN-5）。
type SessionService struct {
	d SessionDeps
}

func NewSessionService(d SessionDeps) *SessionService {
	return &SessionService{d: d}
}

// Authenticate はセッションのトークンからログイン中のユーザーを返す。
// セッションが期限切れ・失効しているか、ユーザーが有効（停止・失効・削除でない）でなければ 401。
func (s *SessionService) Authenticate(ctx context.Context, token string) (*domain.User, error) {
	if token == "" {
		return nil, errUnauthenticated
	}
	now := s.d.Clock.Now()
	sess, err := s.d.Sessions.FindActiveByTokenHash(ctx, auth.HashToken(token), now)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errUnauthenticated
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	u, err := s.d.Users.FindByID(ctx, sess.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errUnauthenticated
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if u.Status != domain.UserStatusActive || u.IsExpired(now) {
		return nil, errUnauthenticated
	}
	return u, nil
}

// Logout はセッションをすぐに失効させる（AUTHN-5）。ログインしていない・既に失効している場合も成功にする。
func (s *SessionService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	err := s.d.Sessions.Revoke(ctx, auth.HashToken(token), s.d.Clock.Now())
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return apperr.Internal(err)
	}
	return nil
}
