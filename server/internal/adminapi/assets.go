package adminapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/medal"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

const (
	AssetPageSize     = 50
	AssetPageMax      = 200
	assetSlotStatic   = "static"
	assetAwaitingFile = "awaiting_file"
	assetPassed       = "passed"
	assetRejected     = "rejected"
	astcBlockSide     = 6
	astcBlockBytes    = 16
	assetColumns      = `id::text, item_id, slot, address, status, format, width, height, bytes, frames, content_hash, checks, uploaded_by, created_at, checked_at, object_key`
)

var staticTextureLimits = map[string]int64{
	cosmetics.TypeEmoji: 100_000,
	cosmetics.TypeFrame: 125_000,
	cosmetics.TypeTable: 1_000_000,
}

var (
	errAssetStoreOff = errors.New("adminapi: no asset store is configured")
	errAssetItem     = errors.New("adminapi: not a cosmetic item or medal id")
	errAssetSlot     = errors.New("adminapi: this slot has no agreed file format yet")
	errAssetUnknown  = errors.New("adminapi: no such asset")
	errAssetChecked  = errors.New("adminapi: the asset was already checked")
)

type AssetStore interface {
	SignedPut(key string, now time.Time) string
	SignedGet(key string, now time.Time) string
	Open(ctx context.Context, key string, now time.Time) (io.ReadCloser, error)
}

type assetView struct {
	ID          string          `json:"id"`
	ItemID      string          `json:"item_id"`
	Slot        string          `json:"slot"`
	Address     string          `json:"address"`
	Status      string          `json:"status"`
	Format      *string         `json:"format"`
	Width       *int            `json:"width"`
	Height      *int            `json:"height"`
	Bytes       *int64          `json:"bytes"`
	Frames      *int            `json:"frames"`
	ContentHash *string         `json:"content_hash"`
	Checks      json.RawMessage `json:"checks"`
	UploadedBy  string          `json:"uploaded_by"`
	CreatedAt   time.Time       `json:"created_at"`
	CheckedAt   *time.Time      `json:"checked_at"`
	ViewURL     string          `json:"view_url,omitempty"`
	TextureSize *int64          `json:"texture_bytes"`
	SizeLimit   *int64          `json:"texture_limit"`
	objectKey   string
}

func scanAsset(row pgx.Row) (assetView, error) {
	var a assetView
	err := row.Scan(&a.ID, &a.ItemID, &a.Slot, &a.Address, &a.Status, &a.Format, &a.Width, &a.Height, &a.Bytes, &a.Frames, &a.ContentHash, &a.Checks, &a.UploadedBy, &a.CreatedAt, &a.CheckedAt, &a.objectKey)
	return a, err
}

func (h *Handler) listAssets(w http.ResponseWriter, r *http.Request, _ principal) {
	query := r.URL.Query()
	before, exact := query.Get("before"), query.Get("exact")
	limit, err := optionalNumber(query.Get("limit"))
	if err != nil || (before != "" && uuid.Validate(before) != nil) || (exact != "" && exact != "true") {
		h.refuse(w, errMalformed)
		return
	}
	if limit == 0 {
		limit = AssetPageSize
	}
	rows, err := h.pool.Query(r.Context(), `select `+assetColumns+` from cosmetic_assets
		where ($1 = '' or status = $1) and (item_id = $2 or (not $5 and starts_with(item_id, $2))) and ($6 = '' or slot = $6)
			and ($3 = '' or (created_at, id) < (select created_at, id from cosmetic_assets where id = nullif($3, '')::uuid))
		order by created_at desc, id desc limit $4`, query.Get("status"), query.Get("item"), before, min(limit, AssetPageMax), exact == "true", query.Get("slot"))
	if err != nil {
		h.refuse(w, err)
		return
	}
	assets, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (assetView, error) { return scanAsset(row) })
	if err != nil {
		h.refuse(w, err)
		return
	}
	for at := range assets {
		assets[at] = h.viewable(assets[at])
	}
	reply(w, http.StatusOK, map[string]any{"store_configured": h.config.Assets != nil, "assets": assets})
}

func (h *Handler) viewable(asset assetView) assetView {
	if h.config.Assets != nil && asset.Status == assetPassed {
		asset.ViewURL = h.config.Assets.SignedGet(asset.objectKey, h.now())
	}
	if asset.Width != nil && asset.Height != nil {
		estimate := astcEstimate(*asset.Width, *asset.Height)
		asset.TextureSize = &estimate
	}
	if limit, ok := staticTextureLimits[cosmetics.TypeOf(asset.ItemID)]; ok {
		asset.SizeLimit = &limit
	}
	return asset
}

func astcEstimate(width, height int) int64 {
	blocks := func(side int) int64 { return int64((side + astcBlockSide - 1) / astcBlockSide) }
	return blocks(width) * blocks(height) * astcBlockBytes
}

func (h *Handler) startAssetUpload(w http.ResponseWriter, r *http.Request, p principal) {
	var request struct {
		ItemID string `json:"item_id"`
		Slot   string `json:"slot"`
	}
	if !decode(w, r, &request) {
		return
	}
	if err := h.acceptsUpload(request.ItemID, request.Slot); err != nil {
		h.refuse(w, err)
		return
	}
	id := uuid.NewString()
	address := assetAddress(request.ItemID, request.Slot)
	key := address + "/" + id + ".png"
	var asset assetView
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		var err error
		asset, err = scanAsset(tx.QueryRow(r.Context(), `insert into cosmetic_assets (id, item_id, slot, address, object_key, status, uploaded_by, created_at)
			values ($1, $2, $3, $4, $5, $6, $7, $8) returning `+assetColumns, id, request.ItemID, request.Slot, address, key, assetAwaitingFile, p.account.ID, now))
		if err != nil {
			return nil, err
		}
		return []audit.Entry{assetEntry(string(rbac.Create), asset)}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"asset": asset, "upload_url": h.config.Assets.SignedPut(key, h.now())})
}

func (h *Handler) acceptsUpload(itemID, slot string) error {
	switch {
	case h.config.Assets == nil:
		return errAssetStoreOff
	case !cosmetics.ValidItemID(itemID) && !medal.ValidID(itemID):
		return fmt.Errorf("%w: %q", errAssetItem, itemID)
	case slot != assetSlotStatic:
		return fmt.Errorf("%w: %q", errAssetSlot, slot)
	}
	return nil
}

func assetAddress(itemID, slot string) string {
	if medal.ValidID(itemID) {
		return medal.StaticAddress(itemID)
	}
	return cosmetics.AddressOf(itemID, slot)
}

func (h *Handler) checkAsset(w http.ResponseWriter, r *http.Request, p principal) {
	if h.config.Assets == nil {
		h.refuse(w, errAssetStoreOff)
		return
	}
	asset, err := h.awaitingAsset(r.Context(), r.PathValue("id"))
	if err != nil {
		h.refuse(w, err)
		return
	}
	measured, err := h.measureAsset(r.Context(), asset)
	if err != nil {
		h.refuse(w, err)
		return
	}
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		var err error
		asset, err = scanAsset(tx.QueryRow(r.Context(), `update cosmetic_assets
			set status = $2, format = $3, width = $4, height = $5, bytes = $6, frames = $7, content_hash = $8, checks = $9, checked_at = $10
			where id = $1 and status = $11 returning `+assetColumns,
			asset.ID, measured.status(), measured.Format, measured.Width, measured.Height, measured.Bytes, measured.Frames, measured.ContentHash, measured.checksJSON(), now, assetAwaitingFile))
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errAssetChecked
		}
		if err != nil {
			return nil, err
		}
		return []audit.Entry{assetEntry("check", asset)}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"asset": h.viewable(asset)})
}

func (h *Handler) awaitingAsset(ctx context.Context, id string) (assetView, error) {
	if uuid.Validate(id) != nil {
		return assetView{}, errAssetUnknown
	}
	asset, err := scanAsset(h.pool.QueryRow(ctx, `select `+assetColumns+` from cosmetic_assets where id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return assetView{}, errAssetUnknown
	}
	if err != nil {
		return assetView{}, err
	}
	if asset.Status != assetAwaitingFile {
		return assetView{}, errAssetChecked
	}
	return asset, nil
}

func (h *Handler) measureAsset(ctx context.Context, asset assetView) (assetMeasure, error) {
	body, err := h.config.Assets.Open(ctx, asset.objectKey, h.now())
	if err != nil {
		return assetMeasure{}, err
	}
	defer body.Close()
	measured, err := measureStatic(body)
	if err != nil || measured.Width == nil {
		return measured, err
	}
	if limit, ok := staticTextureLimits[cosmetics.TypeOf(asset.ItemID)]; ok {
		measured.Checks["size"] = astcEstimate(*measured.Width, *measured.Height) < limit
	}
	return measured, nil
}

func assetEntry(action string, asset assetView) audit.Entry {
	after, _ := json.Marshal(asset)
	return audit.Entry{Action: action, Object: audit.Object{Type: rbac.Assets, ID: asset.ID}, After: after}
}

type assetMeasure struct {
	Format      *string
	Width       *int
	Height      *int
	Frames      *int
	Bytes       int64
	ContentHash string
	Checks      map[string]bool
}

func (m assetMeasure) status() string {
	for _, passed := range m.Checks {
		if !passed {
			return assetRejected
		}
	}
	return assetPassed
}

func (m assetMeasure) checksJSON() []byte {
	encoded, _ := json.Marshal(m.Checks)
	return encoded
}

func measureStatic(file io.Reader) (assetMeasure, error) {
	digest := sha256.New()
	counted := &countingWriter{}
	read := io.TeeReader(file, io.MultiWriter(digest, counted))
	config, decodeErr := png.DecodeConfig(read)
	if _, err := io.Copy(io.Discard, read); err != nil {
		return assetMeasure{}, fmt.Errorf("%w: %v", objectstore.ErrUnavailable, err)
	}
	measured := assetMeasure{Bytes: counted.total, ContentHash: hex.EncodeToString(digest.Sum(nil)), Checks: map[string]bool{"format": decodeErr == nil}}
	if decodeErr != nil {
		return measured, nil
	}
	format, frames := "png", 1
	measured.Format, measured.Width, measured.Height, measured.Frames = &format, &config.Width, &config.Height, &frames
	measured.Checks["alpha"] = config.ColorModel == color.NRGBAModel || config.ColorModel == color.NRGBA64Model
	return measured, nil
}

type countingWriter struct {
	total int64
}

func (c *countingWriter) Write(chunk []byte) (int, error) {
	c.total += int64(len(chunk))
	return len(chunk), nil
}
