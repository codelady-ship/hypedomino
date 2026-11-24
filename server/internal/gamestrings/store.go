package gamestrings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUnknownKey = errors.New("gamestrings: unknown key")
	ErrNoBundle   = errors.New("gamestrings: no such bundle version")
)

type Published struct {
	Language Language
	Version  int
	Hash     string
	Content  []byte
	By       string
	At       time.Time
}

func (p Published) File() string {
	return "strings/" + string(p.Language) + "/" + p.Hash + ".json"
}

type database interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Store struct {
	db database
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{db: pool}
}

func (s *Store) In(tx pgx.Tx) *Store {
	return &Store{db: tx}
}

func (s *Store) SaveEntry(ctx context.Context, entry Entry, by string, now time.Time) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	placeholders, err := json.Marshal(entry.placeholders())
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `insert into game_strings (key, note, max_length, placeholders, deprecated, updated_by, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7)
		on conflict (key) do update set note = excluded.note, max_length = excluded.max_length,
			placeholders = excluded.placeholders, deprecated = excluded.deprecated,
			updated_by = excluded.updated_by, updated_at = excluded.updated_at`,
		entry.Key, entry.Note, entry.MaxLength, placeholders, entry.Deprecated, by, now.UTC())
	return err
}

func (s *Store) AddApprovedIfMissing(ctx context.Context, entry Entry, english string, by string, now time.Time) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	if err := entry.CheckValue(English, english); err != nil {
		return err
	}
	placeholders, err := json.Marshal(entry.placeholders())
	if err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `insert into game_strings (key, note, max_length, placeholders, deprecated, updated_by, updated_at)
		values ($1, $2, $3, $4, false, $5, $6) on conflict (key) do nothing`,
		entry.Key, entry.Note, entry.MaxLength, placeholders, by, now.UTC())
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	return upsertValue(ctx, s.db, entry.Key, English, Value{Text: english, Status: Approved}, by, now)
}

func (e Entry) placeholders() []Placeholder {
	if e.Placeholders == nil {
		return []Placeholder{}
	}
	return e.Placeholders
}

func (s *Store) SetValue(ctx context.Context, key string, language Language, value Value, by string, now time.Time) error {
	if !slices.Contains(Languages, language) {
		return fmt.Errorf("%w: unknown language %q", ErrInvalidValue, language)
	}
	if value.Status != Draft && value.Status != Approved {
		return fmt.Errorf("%w: status %q cannot be set", ErrInvalidValue, value.Status)
	}
	entry, err := s.Entry(ctx, key)
	if err != nil {
		return err
	}
	if value.Status == Approved {
		if err := entry.CheckValue(language, value.Text); err != nil {
			return err
		}
	}
	return upsertValue(ctx, s.db, key, language, value, by, now)
}

type executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func upsertValue(ctx context.Context, db executor, key string, language Language, value Value, by string, now time.Time) error {
	_, err := db.Exec(ctx, `insert into game_string_values (key, language, text, status, updated_by, updated_at)
		values ($1, $2, $3, $4, $5, $6)
		on conflict (key, language) do update set text = excluded.text, status = excluded.status,
			updated_by = excluded.updated_by, updated_at = excluded.updated_at`,
		key, language, value.Text, value.Status, by, now.UTC())
	return err
}

const entryColumns = `key, note, max_length, placeholders, deprecated`

func scanEntry(row pgx.CollectableRow) (Entry, error) {
	var e Entry
	var placeholders []byte
	if err := row.Scan(&e.Key, &e.Note, &e.MaxLength, &placeholders, &e.Deprecated); err != nil {
		return Entry{}, err
	}
	if err := json.Unmarshal(placeholders, &e.Placeholders); err != nil {
		return Entry{}, err
	}
	e.Values = map[Language]Value{}
	return e, nil
}

func (s *Store) Value(ctx context.Context, key string, language Language) (Value, error) {
	value := Value{Status: Missing}
	err := s.db.QueryRow(ctx, `select text, status from game_string_values where key = $1 and language = $2`, key, language).Scan(&value.Text, &value.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Value{Status: Missing}, nil
	}
	return value, err
}

func (s *Store) Unapproved(ctx context.Context, keys []string, language Language) ([]string, error) {
	rows, err := s.db.Query(ctx, `select listed.key from unnest($1::text[]) with ordinality listed(key, place)
		where not exists (select 1 from game_string_values v where v.key = listed.key and v.language = $2 and v.status = $3)
		order by listed.place`, keys, language, Approved)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (s *Store) Entry(ctx context.Context, key string) (Entry, error) {
	rows, err := s.db.Query(ctx, `select `+entryColumns+` from game_strings where key = $1`, key)
	if err != nil {
		return Entry{}, err
	}
	entry, err := pgx.CollectExactlyOneRow(rows, scanEntry)
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, fmt.Errorf("%w: %s", ErrUnknownKey, key)
	}
	return entry, err
}

func (s *Store) Entries(ctx context.Context) ([]Entry, error) {
	rows, err := s.db.Query(ctx, `select `+entryColumns+` from game_strings order by key`)
	if err != nil {
		return nil, err
	}
	return s.withValues(ctx, rows)
}

type EntryQuery struct {
	Text    string
	Missing Language
	After   string
	Limit   int
}

func (s *Store) Page(ctx context.Context, q EntryQuery) ([]Entry, error) {
	rows, err := s.db.Query(ctx, `select `+entryColumns+` from game_strings s
		where (strpos(s.key, lower($1)) > 0 or strpos(lower(s.note), lower($1)) > 0
				or exists (select 1 from game_string_values t where t.key = s.key and strpos(lower(t.text), lower($1)) > 0))
			and s.key > $2
			and ($3 = '' or (not s.deprecated and not exists (select 1 from game_string_values v where v.key = s.key and v.language = $3 and v.status = $4)))
		order by s.key limit $5`, q.Text, q.After, q.Missing, Approved, q.Limit)
	if err != nil {
		return nil, err
	}
	return s.withValues(ctx, rows)
}

func (s *Store) withValues(ctx context.Context, rows pgx.Rows) ([]Entry, error) {
	entries, err := pgx.CollectRows(rows, scanEntry)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]Entry, len(entries))
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		byKey[entry.Key] = entry
		keys = append(keys, entry.Key)
	}
	rows, err = s.db.Query(ctx, `select key, language, text, status from game_string_values where key = any($1)`, keys)
	if err != nil {
		return nil, err
	}
	var key string
	var language Language
	var value Value
	_, err = pgx.ForEachRow(rows, []any{&key, &language, &value.Text, &value.Status}, func() error {
		byKey[key].Values[language] = value
		return nil
	})
	return entries, err
}

type Coverage struct {
	Language Language
	Keys     int
	Approved int
	Percent  float64
}

func (s *Store) Coverage(ctx context.Context) ([]Coverage, error) {
	languages := make([]string, 0, len(Languages))
	for _, language := range Languages {
		languages = append(languages, string(language))
	}
	rows, err := s.db.Query(ctx, `select l.language, count(s.key), count(v.key)
		from unnest($1::text[]) with ordinality l(language, place)
		left join game_strings s on not s.deprecated
		left join game_string_values v on v.key = s.key and v.language = l.language and v.status = $2
		group by l.language, l.place order by l.place`, languages, Approved)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Coverage, error) {
		var c Coverage
		if err := row.Scan(&c.Language, &c.Keys, &c.Approved); err != nil {
			return Coverage{}, err
		}
		if c.Keys > 0 {
			c.Percent = float64(c.Approved) * 100 / float64(c.Keys)
		}
		return c, nil
	})
}
func (s *Store) Publish(ctx context.Context, language Language, by string, now time.Time) (Published, []Problem, error) {
	if !slices.Contains(Languages, language) {
		return Published{}, nil, fmt.Errorf("%w: unknown language %q", ErrInvalidValue, language)
	}
	entries, err := s.Entries(ctx)
	if err != nil {
		return Published{}, nil, err
	}
	bundle, problems := BuildBundle(entries, language)
	if problems != nil {
		return Published{}, problems, fmt.Errorf("%w: %d values fail their checks", ErrBlocked, len(problems))
	}
	content, hash, err := bundle.Encode()
	if err != nil {
		return Published{}, nil, err
	}
	published, err := s.store(ctx, Published{Language: language, Hash: hash, Content: content, By: by, At: now.UTC()})
	return published, nil, err
}

func (s *Store) store(ctx context.Context, bundle Published) (Published, error) {
	var stored Published
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtext('game_string_bundles:' || $1))`, bundle.Language); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `insert into game_string_bundles (language, version, hash, content, published_by, published_at)
			select $1, coalesce(max(version), 0) + 1, $2, $3, $4, $5 from game_string_bundles where language = $1
			on conflict (language, hash) do nothing`,
			bundle.Language, bundle.Hash, bundle.Content, bundle.By, bundle.At); err != nil {
			return err
		}
		var err error
		stored, err = scanBundle(tx.QueryRow(ctx, `select `+bundleColumns+` from game_string_bundles where language = $1 and hash = $2`, bundle.Language, bundle.Hash))
		return err
	})
	return stored, err
}

func (s *Store) Export(ctx context.Context, language Language, format Format) ([]byte, error) {
	entries, err := s.Entries(ctx)
	if err != nil {
		return nil, err
	}
	transfer, err := BuildTransfer(entries, language)
	if err != nil {
		return nil, err
	}
	return transfer.Encode(format)
}

func (s *Store) Import(ctx context.Context, language Language, format Format, data []byte, by string, now time.Time) ([]Change, []Problem, error) {
	transfer, err := ParseTransfer(data, format, language)
	if err != nil {
		return nil, nil, err
	}
	entries, err := s.Entries(ctx)
	if err != nil {
		return nil, nil, err
	}
	changes, problems := PlanImport(entries, transfer)
	if problems != nil {
		return nil, problems, fmt.Errorf("%w: %d rows fail their checks", ErrImportBlocked, len(problems))
	}
	err = pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		for _, change := range changes {
			if err := upsertValue(ctx, tx, change.Key, language, Value{Text: change.Text, Status: Draft}, by, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return changes, nil, nil
}

const bundleColumns = `language, version, hash, content, published_by, published_at`

func scanBundle(row pgx.Row) (Published, error) {
	var p Published
	if err := row.Scan(&p.Language, &p.Version, &p.Hash, &p.Content, &p.By, &p.At); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Published{}, ErrNoBundle
		}
		return Published{}, err
	}
	p.At = p.At.UTC()
	return p, nil
}

func (s *Store) Bundles(ctx context.Context, language Language, before, limit int) ([]Published, error) {
	rows, err := s.db.Query(ctx, `select language, version, hash, ''::bytea, published_by, published_at from game_string_bundles
		where language = $1 and ($2 = 0 or version < $2) order by version desc limit $3`, language, before, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Published, error) { return scanBundle(row) })
}

func (s *Store) Bundle(ctx context.Context, language Language, version int) (Published, error) {
	return scanBundle(s.db.QueryRow(ctx, `select `+bundleColumns+` from game_string_bundles where language = $1 and version = $2`, language, version))
}
