package service

import (
	"context"
	"errors"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
)

var errFeeRateRange = apperr.BadRequest("手数料率は 0〜100% の範囲で入力してください")

// SettingDeps は SettingService の依存。
type SettingDeps struct {
	Settings SettingStore
	// AdminSettings・Users・Admins・Clock は管理画面での手数料率の変更に使う（ADMIN, ADR 0034）
	AdminSettings AdminSettingStore
	Users         UserStore
	Admins        domain.AdminList
	Clock         clock.Clock
}

// SettingService は手数料率などの設定を扱う（PAY-2）。変更は管理画面（ADMIN）で行う。
type SettingService struct {
	d SettingDeps
}

func NewSettingService(d SettingDeps) *SettingService {
	return &SettingService{d: d}
}

// FeeRate は現在の手数料率（ベーシスポイント）を返す。出品画面の受け取り見込みの表示に使う。
func (s *SettingService) FeeRate(ctx context.Context) (int, error) {
	bp, err := s.d.Settings.FeeRate(ctx)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return bp, nil
}

// AdminFeeRateView は管理画面に出す手数料率。
type AdminFeeRateView struct {
	domain.FeeRateSetting
	// UpdatedByStudentNumber は最後に変更した管理者の学籍番号。初期値のままなら nil
	UpdatedByStudentNumber *string
}

// AdminFeeRate は現在の手数料率と、最後に変更した日時・管理者を返す（ADMIN, ADR 0034）。管理者でなければ 404。
func (s *SettingService) AdminFeeRate(ctx context.Context, admin *domain.User) (*AdminFeeRateView, error) {
	if err := requireAdmin(s.d.Admins, admin); err != nil {
		return nil, err
	}
	return s.adminFeeRate(ctx)
}

// UpdateFeeRate は手数料率（0〜10000 ベーシスポイント）を変える（PAY-2, ADR 0034）。管理者でなければ 404、範囲外なら 400。
// 取引には成立時点の料率を記録するので、変更は以後に成立する取引にだけ使われる（ADR 0003）。
func (s *SettingService) UpdateFeeRate(ctx context.Context, admin *domain.User, bp int) (*AdminFeeRateView, error) {
	if err := requireAdmin(s.d.Admins, admin); err != nil {
		return nil, err
	}
	if bp < 0 || bp > domain.FeeRateBasisPointsMax {
		return nil, errFeeRateRange
	}
	if err := s.d.AdminSettings.UpdateFeeRate(ctx, bp, admin.ID, s.d.Clock.Now()); err != nil {
		return nil, apperr.Internal(err)
	}
	return s.adminFeeRate(ctx)
}

func (s *SettingService) adminFeeRate(ctx context.Context) (*AdminFeeRateView, error) {
	st, err := s.d.AdminSettings.FeeRateSetting(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	v := &AdminFeeRateView{FeeRateSetting: st}
	if st.UpdatedBy != nil {
		u, err := s.d.Users.FindByID(ctx, *st.UpdatedBy)
		if err != nil {
			return nil, apperr.Internal(errors.Join(errors.New("service: 手数料率を変更した管理者が見つかりません"), err))
		}
		v.UpdatedByStudentNumber = &u.StudentNumber
	}
	return v, nil
}
