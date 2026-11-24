package adminapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/shop"
)

type shopPreviewRequest struct {
	Catalog  json.RawMessage   `json:"catalog"`
	At       *time.Time        `json:"at"`
	Variants map[string]string `json:"variants"`
}

type previewOfferView struct {
	ID      string `json:"id"`
	NameKey string `json:"name_key"`
	Variant string `json:"variant,omitempty"`
}

type previewSectionView struct {
	Section string             `json:"section"`
	EndsAt  time.Time          `json:"ends_at,omitzero"`
	Offers  []previewOfferView `json:"offers"`
}

type shopPreviewView struct {
	At       time.Time            `json:"at"`
	Sections []previewSectionView `json:"sections"`
}

func (h *Handler) previewShop(w http.ResponseWriter, r *http.Request, p principal) {
	var request shopPreviewRequest
	if !decodeUpTo(w, r, &request, MaxCatalogBytes) {
		return
	}
	if len(request.Catalog) == 0 {
		h.refuse(w, fmt.Errorf("%w: catalog", errMalformed))
		return
	}
	catalog, err := shop.Parse(request.Catalog, h.config.Shop)
	if err != nil {
		h.refuse(w, err)
		return
	}
	if err := chosenVariantsExist(catalog, request.Variants); err != nil {
		h.refuse(w, err)
		return
	}
	at := h.now().UTC()
	if request.At != nil {
		at = request.At.UTC()
	}
	view := shopPreviewView{At: at, Sections: []previewSectionView{}}
	for _, section := range catalog.Sections(h.config.Shop, at) {
		section = section.WithVariants(request.Variants)
		offers := make([]previewOfferView, 0, len(section.Offers))
		for _, offer := range section.Offers {
			shown := previewOfferView{ID: offer.ID, NameKey: offer.NameKey}
			if variant, ok := offer.Variant(request.Variants[offer.ID]); ok {
				shown.Variant = variant.ID
			}
			offers = append(offers, shown)
		}
		view.Sections = append(view.Sections, previewSectionView{Section: section.Name, EndsAt: section.EndsAt, Offers: offers})
	}
	reply(w, http.StatusOK, view)
}

func chosenVariantsExist(catalog *shop.Catalog, chosen map[string]string) error {
	for offerID, variantID := range chosen {
		offer, ok := catalog.Offer(offerID)
		if !ok {
			return fmt.Errorf("%w: variants name unknown offer %s", errMalformed, offerID)
		}
		if _, ok := offer.Variant(variantID); !ok {
			return fmt.Errorf("%w: offer %s has no variant %q", errMalformed, offerID, variantID)
		}
	}
	return nil
}
