package gamestrings

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

var (
	ErrInvalidImport = errors.New("gamestrings: invalid import file")
	ErrImportBlocked = errors.New("gamestrings: import blocked")
)

type Format string

const (
	CSV  Format = "csv"
	JSON Format = "json"
)

const byteOrderMark = "\ufeff"

type TransferRow struct {
	Key          string        `json:"key"`
	Note         string        `json:"note,omitempty"`
	MaxLength    int           `json:"maxLength,omitempty"`
	Placeholders []Placeholder `json:"placeholders,omitempty"`
	Reference    string        `json:"reference,omitempty"`
	Text         string        `json:"text"`
	Status       Status        `json:"status,omitempty"`
}

type Transfer struct {
	Language Language      `json:"language"`
	Rows     []TransferRow `json:"rows"`
}

type Change struct {
	Key  string
	Text string
}

func BuildTransfer(entries []Entry, language Language) (Transfer, error) {
	if !slices.Contains(Languages, language) {
		return Transfer{}, fmt.Errorf("%w: unknown language %q", ErrInvalidValue, language)
	}
	transfer := Transfer{Language: language, Rows: []TransferRow{}}
	for _, entry := range entries {
		if entry.Deprecated {
			continue
		}
		value := entry.Values[language]
		row := TransferRow{
			Key: entry.Key, Note: entry.Note, MaxLength: entry.MaxLength, Placeholders: entry.placeholders(),
			Text: value.Text, Status: statusOf(value),
		}
		if language != English {
			row.Reference = entry.Values[English].Text
		}
		transfer.Rows = append(transfer.Rows, row)
	}
	slices.SortFunc(transfer.Rows, func(a, b TransferRow) int { return strings.Compare(a.Key, b.Key) })
	return transfer, nil
}

func statusOf(value Value) Status {
	if value.Status == "" {
		return Missing
	}
	return value.Status
}

func (t Transfer) Encode(format Format) ([]byte, error) {
	switch format {
	case JSON:
		return json.MarshalIndent(t, "", "  ")
	case CSV:
		return t.encodeCSV()
	}
	return nil, fmt.Errorf("%w: format %q", ErrInvalidImport, format)
}

func (t Transfer) header() []string {
	header := []string{"key", "note", "max_length", "placeholders"}
	if t.Language != English {
		header = append(header, string(English))
	}
	return append(header, string(t.Language), "status")
}

func (t Transfer) encodeCSV() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(byteOrderMark)
	writer := csv.NewWriter(&buffer)
	if err := writer.Write(t.header()); err != nil {
		return nil, err
	}
	for _, row := range t.Rows {
		record := []string{row.Key, row.Note, strconv.Itoa(row.MaxLength), placeholderCell(row.Placeholders)}
		if t.Language != English {
			record = append(record, row.Reference)
		}
		if err := writer.Write(append(record, row.Text, string(row.Status))); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}

func placeholderCell(placeholders []Placeholder) string {
	cells := make([]string, len(placeholders))
	for i, placeholder := range placeholders {
		cells[i] = placeholder.Name + ":" + string(placeholder.Type)
	}
	return strings.Join(cells, " ")
}

func ParseTransfer(data []byte, format Format, language Language) (Transfer, error) {
	if !slices.Contains(Languages, language) {
		return Transfer{}, fmt.Errorf("%w: unknown language %q", ErrInvalidValue, language)
	}
	switch format {
	case JSON:
		return parseJSON(data, language)
	case CSV:
		return parseCSV(data, language)
	}
	return Transfer{}, fmt.Errorf("%w: format %q", ErrInvalidImport, format)
}

func parseJSON(data []byte, language Language) (Transfer, error) {
	var transfer Transfer
	if err := json.Unmarshal(data, &transfer); err != nil {
		return Transfer{}, fmt.Errorf("%w: %w", ErrInvalidImport, err)
	}
	if transfer.Language != language {
		return Transfer{}, fmt.Errorf("%w: file is for %q, not %q", ErrInvalidImport, transfer.Language, language)
	}
	return transfer, nil
}

func parseCSV(data []byte, language Language) (Transfer, error) {
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte(byteOrderMark))))
	records, err := reader.ReadAll()
	if err != nil {
		return Transfer{}, fmt.Errorf("%w: %w", ErrInvalidImport, err)
	}
	if len(records) == 0 {
		return Transfer{}, fmt.Errorf("%w: no header", ErrInvalidImport)
	}
	keyColumn := slices.Index(records[0], "key")
	textColumn := slices.Index(records[0], string(language))
	if keyColumn < 0 || textColumn < 0 {
		return Transfer{}, fmt.Errorf("%w: header needs key and %s columns", ErrInvalidImport, language)
	}
	transfer := Transfer{Language: language, Rows: make([]TransferRow, 0, len(records)-1)}
	for _, record := range records[1:] {
		transfer.Rows = append(transfer.Rows, TransferRow{Key: record[keyColumn], Text: record[textColumn]})
	}
	return transfer, nil
}

func PlanImport(entries []Entry, transfer Transfer) ([]Change, []Problem) {
	byKey := make(map[string]Entry, len(entries))
	for _, entry := range entries {
		byKey[entry.Key] = entry
	}
	seen := map[string]bool{}
	var changes []Change
	var problems []Problem
	for _, row := range transfer.Rows {
		entry, known := byKey[row.Key]
		switch {
		case seen[row.Key]:
			problems = append(problems, Problem{Key: row.Key, Err: fmt.Errorf("%w: key repeats", ErrInvalidImport)})
			continue
		case !known:
			problems = append(problems, Problem{Key: row.Key, Err: fmt.Errorf("%w: %s", ErrUnknownKey, row.Key)})
			continue
		case entry.Deprecated:
			problems = append(problems, Problem{Key: row.Key, Err: fmt.Errorf("%w: %s is deprecated", ErrInvalidImport, row.Key)})
			continue
		}
		seen[row.Key] = true
		if row.Text == "" || row.Text == entry.Values[transfer.Language].Text {
			continue
		}
		if err := entry.CheckValue(transfer.Language, row.Text); err != nil {
			problems = append(problems, Problem{Key: row.Key, Err: err})
			continue
		}
		changes = append(changes, Change{Key: row.Key, Text: row.Text})
	}
	if len(problems) > 0 {
		return nil, problems
	}
	return changes, nil
}
