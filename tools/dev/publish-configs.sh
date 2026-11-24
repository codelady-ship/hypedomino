#!/bin/sh
set -eu

publish() {
  go run ./cmd/config-publish -domain "$1" -file "/repo/docs/$2"
}

publish cosmetics cosmetics/examples/catalog.json
publish daily_login economy/examples/daily_login.json
publish match_rewards economy/examples/match_rewards.json
publish shop economy/examples/shop.json
publish store_prices economy/examples/store_prices.json
publish rating ranking/examples/rating.json
publish medals pass/examples/medals.json
publish missions pass/examples/missions.json
publish pass pass/examples/pass.json
