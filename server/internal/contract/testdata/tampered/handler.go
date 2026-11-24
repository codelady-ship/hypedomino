package tampered

import (
	"encoding/json"
	"net/http"
)

type buyRequest struct {
	Offer      string `json:"offer"`
	PriceCoins int64  `json:"price_coins"`
	Grant      grant  `json:"grant"`
}

type grant struct {
	RewardXP int `json:"rewardXp"`
}

func buy(w http.ResponseWriter, r *http.Request) {
	var request buyRequest
	json.NewDecoder(r.Body).Decode(&request)
}

func claim(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Balance int64
	}
	json.NewDecoder(r.Body).Decode(&request)
}
