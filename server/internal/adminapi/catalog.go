package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

const MaxCatalogBytes = 1 << 20

var errOfferRemoved = errors.New("adminapi: a published offer stays in the catalog, its window ends it")

type catalogView struct {
	ConfigVersion int64           `json:"config_version"`
	Catalog       json.RawMessage `json:"catalog"`
}

type catalogVersionView struct {
	ConfigVersion int64           `json:"config_version"`
	PublishedAt   time.Time       `json:"published_at"`
	Catalog       json.RawMessage `json:"catalog"`
}

type catalogHistoryView struct {
	Versions []catalogVersionView `json:"versions"`
}

type publishCatalogRequest struct {
	Catalog json.RawMessage `json:"catalog"`
	changeNote
}

func (h *Handler) catalogDomain(publish bool) configDomain {
	prepare := h.restorableCatalog
	if publish {
		prepare = h.publishableCatalog
	}
	return configDomain{name: shop.Domain, object: rbac.ShopOffers, prepare: prepare, accept: func(before, after []byte) error {
		next, err := shop.Parse(after, h.config.Shop)
		if err != nil || !publish || before == nil {
			return err
		}
		var current shop.Catalog
		if err := json.Unmarshal(before, &current); err != nil {
			return err
		}
		for _, offer := range current.Offers {
			if _, kept := next.Offer(offer.ID); !kept {
				return fmt.Errorf("%w: %s", errOfferRemoved, offer.ID)
			}
		}
		return nil
	}}
}

func viewCatalog(version int64, raw []byte) catalogView {
	if raw == nil {
		raw = []byte("null")
	}
	return catalogView{ConfigVersion: version, Catalog: raw}
}

type rotationRulesView struct {
	DailySlots   int `json:"daily_slots"`
	WeeklySlots  int `json:"weekly_slots"`
	ResetHourUTC int `json:"reset_hour_utc"`
}

type shopCatalogView struct {
	catalogView
	Rotation           rotationRulesView `json:"rotation"`
	ReferencePriceDays int               `json:"reference_price_days"`
}

func (h *Handler) shopCatalog(w http.ResponseWriter, r *http.Request, p principal) {
	version, raw, err := latestSnapshot(r.Context(), h.pool, shop.Domain)
	if err != nil {
		h.refuse(w, err)
		return
	}
	rules := rotationRulesView{DailySlots: h.config.Shop.DailySlots, WeeklySlots: h.config.Shop.WeeklySlots, ResetHourUTC: h.config.Shop.ResetHourUTC}
	reply(w, http.StatusOK, shopCatalogView{catalogView: viewCatalog(version, raw), Rotation: rules, ReferencePriceDays: h.config.Shop.ReferencePriceDays})
}

func (h *Handler) publishShop(w http.ResponseWriter, r *http.Request, p principal) {
	var request publishCatalogRequest
	if !decodeUpTo(w, r, &request, MaxCatalogBytes) {
		return
	}
	if len(request.Catalog) == 0 {
		h.refuse(w, fmt.Errorf("%w: catalog", errMalformed))
		return
	}
	version, _, err := h.replaceSnapshot(r.Context(), p, h.catalogDomain(true), "publish", request.Catalog, request.changeNote, nil)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, viewCatalog(version, request.Catalog))
}

func (h *Handler) rollbackShop(w http.ResponseWriter, r *http.Request, p principal) {
	var request rollbackRequest
	if !decode(w, r, &request) {
		return
	}
	raw, err := h.earlierSnapshot(r.Context(), shop.Domain, request.Version)
	if err != nil {
		h.refuse(w, err)
		return
	}
	version, _, err := h.replaceSnapshot(r.Context(), p, h.catalogDomain(false), "rollback", raw, request.changeNote, h.mayRollBack)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, viewCatalog(version, raw))
}

func (h *Handler) restorableCatalog(ctx context.Context, tx pgx.Tx, _, after []byte) ([]byte, error) {
	_, err := h.listedAndNamedCatalog(ctx, tx, after)
	return after, err
}

func (h *Handler) publishableCatalog(ctx context.Context, tx pgx.Tx, _, after []byte) ([]byte, error) {
	catalog, err := h.listedAndNamedCatalog(ctx, tx, after)
	if err != nil {
		return nil, err
	}
	inEffect, err := snapshot.InEffectWithinDays(ctx, tx, shop.Domain, h.config.Shop.ReferencePriceDays)
	if err != nil {
		return nil, err
	}
	return after, catalog.CheckReferencePrices(inEffect)
}

func (h *Handler) listedAndNamedCatalog(ctx context.Context, tx pgx.Tx, raw []byte) (*shop.Catalog, error) {
	catalog, err := shop.Parse(raw, h.config.Shop)
	if err != nil {
		return nil, err
	}
	items, err := cosmetics.Published(ctx, tx)
	if err != nil {
		return nil, err
	}
	if err := catalog.CheckSources(items); err != nil {
		return nil, err
	}
	nameKeys := make([]string, 0, len(catalog.Offers))
	for _, offer := range catalog.Offers {
		nameKeys = append(nameKeys, offer.NameKey)
		for _, variant := range offer.Variants {
			if variant.NameKey != "" {
				nameKeys = append(nameKeys, variant.NameKey)
			}
		}
	}
	return catalog, h.keysApprovedInEnglish(ctx, tx, shop.ErrInvalid, nameKeys)
}

type cosmeticItemView struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Rarity       string             `json:"rarity"`
	Availability string             `json:"availability"`
	Sources      []cosmetics.Source `json:"sources"`
}

type cosmeticItemsView struct {
	Items []cosmeticItemView `json:"items"`
}

func (h *Handler) cosmeticItems(w http.ResponseWriter, r *http.Request, p principal) {
	catalog, err := cosmetics.Published(r.Context(), h.pool)
	if err != nil {
		h.refuse(w, err)
		return
	}
	items := catalog.Items()
	view := cosmeticItemsView{Items: make([]cosmeticItemView, 0, len(items))}
	for _, item := range items {
		view.Items = append(view.Items, cosmeticItemView{ID: item.ID, Type: item.Type, Rarity: item.Rarity, Availability: item.Availability, Sources: item.Sources})
	}
	reply(w, http.StatusOK, view)
}
