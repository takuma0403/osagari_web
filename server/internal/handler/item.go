package handler

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// ItemService は出品の業務ロジック（service.ItemService）。
type ItemService interface {
	CreateImageUpload(ctx context.Context, u *domain.User, contentType string) (service.ImageUpload, error)
	Create(ctx context.Context, u *domain.User, in domain.ItemContent, imageKeys []string, publish bool) (*service.ItemView, error)
	Get(ctx context.Context, u *domain.User, id string) (*service.ItemView, error)
	Update(ctx context.Context, u *domain.User, id string, in domain.ItemContent, imageKeys []string) (*service.ItemView, error)
	Publish(ctx context.Context, u *domain.User, id string) (*service.ItemView, error)
	Withdraw(ctx context.Context, u *domain.User, id string) (*service.ItemView, error)
}

// CreateItemImageUpload は商品写真のアップロード用の署名付きURLを発行する（POST /api/items/image-uploads。ITEM, STORAGE）。
func (h *Handler) CreateItemImageUpload(ctx context.Context, req api.CreateItemImageUploadRequestObject) (api.CreateItemImageUploadResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	up, err := h.d.Item.CreateImageUpload(ctx, u, string(req.Body.ContentType))
	if err != nil {
		return nil, err
	}
	return api.CreateItemImageUpload201JSONResponse{
		ObjectKey: up.ObjectKey,
		UploadUrl: up.URL,
		Method:    api.ImageUploadMethod(up.Method),
		Headers:   up.Headers,
		ExpiresAt: up.ExpiresAt,
	}, nil
}

// CreateItem は商品を下書きか公開中として作る（POST /api/items。ITEM）。
func (h *Handler) CreateItem(ctx context.Context, req api.CreateItemRequestObject) (api.CreateItemResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	in := toItemContent(api.ItemInput{
		Title: b.Title, Category: b.Category, Condition: b.Condition, Price: b.Price, Description: b.Description,
		Campus: b.Campus, Textbook: b.Textbook, FurnitureAppliance: b.FurnitureAppliance,
	})
	v, err := h.d.Item.Create(ctx, u, in, b.ImageObjectKeys, b.Publish)
	if err != nil {
		return nil, err
	}
	return api.CreateItem201JSONResponse(toOwnItem(v)), nil
}

// GetOwnItem は出品者本人の商品を返す（GET /api/items/{itemId}。ITEM）。
func (h *Handler) GetOwnItem(ctx context.Context, req api.GetOwnItemRequestObject) (api.GetOwnItemResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	v, err := h.d.Item.Get(ctx, u, req.ItemId)
	if err != nil {
		return nil, err
	}
	return api.GetOwnItem200JSONResponse(toOwnItem(v)), nil
}

// UpdateItem は商品を編集する（PUT /api/items/{itemId}。ITEM）。
func (h *Handler) UpdateItem(ctx context.Context, req api.UpdateItemRequestObject) (api.UpdateItemResponseObject, error) {
	if req.Body == nil {
		return nil, errMissingBody
	}
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	v, err := h.d.Item.Update(ctx, u, req.ItemId, toItemContent(*req.Body), req.Body.ImageObjectKeys)
	if err != nil {
		return nil, err
	}
	return api.UpdateItem200JSONResponse(toOwnItem(v)), nil
}

// PublishItem は商品を公開する（POST /api/items/{itemId}/publish。ITEM）。
func (h *Handler) PublishItem(ctx context.Context, req api.PublishItemRequestObject) (api.PublishItemResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	v, err := h.d.Item.Publish(ctx, u, req.ItemId)
	if err != nil {
		return nil, err
	}
	return api.PublishItem200JSONResponse(toOwnItem(v)), nil
}

// WithdrawItem は商品を取り下げる（POST /api/items/{itemId}/withdraw。ITEM）。
func (h *Handler) WithdrawItem(ctx context.Context, req api.WithdrawItemRequestObject) (api.WithdrawItemResponseObject, error) {
	u, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	v, err := h.d.Item.Withdraw(ctx, u, req.ItemId)
	if err != nil {
		return nil, err
	}
	return api.WithdrawItem200JSONResponse(toOwnItem(v)), nil
}

func toItemContent(in api.ItemInput) domain.ItemContent {
	c := domain.ItemContent{
		Title:       in.Title,
		Category:    (*domain.ItemCategory)(in.Category),
		Condition:   (*domain.ItemCondition)(in.Condition),
		Price:       in.Price,
		Description: in.Description,
		Campus:      (*domain.Campus)(in.Campus),
	}
	if t := in.Textbook; t != nil {
		c.Textbook = &domain.TextbookDetails{CourseName: t.CourseName, Instructor: t.Instructor, ISBN: t.Isbn, HasWriting: t.HasWriting}
	}
	if f := in.FurnitureAppliance; f != nil {
		c.FurnitureAppliance = &domain.FurnitureApplianceDetails{Size: f.Size, OperationChecked: f.OperationChecked}
	}
	return c
}

func toOwnItem(v *service.ItemView) api.OwnItem {
	out := api.OwnItem{
		Id:          v.ID,
		ShareId:     v.ShareID,
		Status:      api.ItemStatus(v.Status),
		Title:       v.Title,
		Category:    (*api.ItemCategory)(v.Category),
		Condition:   (*api.ItemCondition)(v.Condition),
		Price:       v.Price,
		Description: v.Description,
		Campus:      (*api.Campus)(v.Campus),
		Images:      make([]api.ItemImage, len(v.Images)),
		PublishedAt: v.PublishedAt,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
	for i, img := range v.Images {
		out.Images[i] = api.ItemImage{ObjectKey: img.ObjectKey, Url: v.ImageURLs[i].URL, ExpiresAt: v.ImageURLs[i].ExpiresAt}
	}
	if t := v.Textbook; t != nil {
		out.Textbook = &api.TextbookDetails{CourseName: t.CourseName, Instructor: t.Instructor, Isbn: t.ISBN, HasWriting: t.HasWriting}
	}
	if f := v.FurnitureAppliance; f != nil {
		out.FurnitureAppliance = &api.FurnitureApplianceDetails{Size: f.Size, OperationChecked: f.OperationChecked}
	}
	return out
}
