package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const maxImportRows = 5000

var (
	ErrImportFile     = &Error{http.StatusBadRequest, "invalid_import_file", "Use the CSV template, keep the file under 5 MB and include no more than 5,000 rows."}
	ErrImportMode     = &Error{http.StatusBadRequest, "invalid_import_mode", "Choose Create new or Update existing before previewing the file."}
	ErrImportChanged  = &Error{http.StatusConflict, "import_changed", "The file or catalog changed since preview. Preview it again before importing."}
	ErrImportKeyReuse = &Error{http.StatusConflict, "import_key_reused", "This import key was already used for a different file. Start a new import."}
)

type ImportRow struct {
	Line            int    `json:"line"`
	ProductName     string `json:"productName"`
	VariantName     string `json:"variantName"`
	SKU             string `json:"sku"`
	Barcode         string `json:"barcode"`
	SellingPrice    string `json:"sellingPrice"`
	OpeningQuantity string `json:"openingQuantity"`
}

type ImportRowResult struct {
	Line        int      `json:"line"`
	ProductName string   `json:"productName"`
	SKU         string   `json:"sku"`
	Action      string   `json:"action"`
	Errors      []string `json:"errors"`
}

type ImportPreview struct {
	Rows         []ImportRowResult `json:"rows"`
	CreateCount  int               `json:"createCount"`
	UpdateCount  int               `json:"updateCount"`
	InvalidCount int               `json:"invalidCount"`
	Hash         string            `json:"hash"`
}

type ImportResult struct {
	Created  int               `json:"created"`
	Updated  int               `json:"updated"`
	Replayed bool              `json:"replayed"`
	Rows     []ImportRowResult `json:"rows"`
}

type ImportRepository interface {
	PreviewImport(context.Context, string, string, []ImportRow) (*ImportPreview, error)
	ConfirmImport(context.Context, string, string, string, string, string, []ImportRow) (*ImportResult, error)
	ExportProducts(context.Context, string) ([][]string, error)
}

type ImportService struct{ repo ImportRepository }

func NewImportService(repo ImportRepository) *ImportService { return &ImportService{repo: repo} }

func (s *ImportService) Preview(ctx context.Context, tenantID, mode string, rows []ImportRow) (*ImportPreview, error) {
	if mode != "create" && mode != "update" {
		return nil, ErrImportMode
	}
	if len(rows) == 0 || len(rows) > maxImportRows {
		return nil, ErrImportFile
	}
	preview, err := s.repo.PreviewImport(ctx, tenantID, mode, rows)
	if err != nil {
		return nil, err
	}
	preview.Hash = importHash(mode, rows)
	return preview, nil
}

func (s *ImportService) Confirm(ctx context.Context, tenantID, userID, mode, key, previewHash string, rows []ImportRow) (*ImportResult, error) {
	if mode != "create" && mode != "update" {
		return nil, ErrImportMode
	}
	if len(rows) == 0 || len(rows) > maxImportRows {
		return nil, ErrImportFile
	}
	if len(key) < 8 || len(key) > 100 || previewHash == "" || previewHash != importHash(mode, rows) {
		return nil, ErrImportChanged
	}
	return s.repo.ConfirmImport(ctx, tenantID, userID, mode, key, previewHash, rows)
}

func (s *ImportService) Export(ctx context.Context, tenantID string) ([][]string, error) {
	return s.repo.ExportProducts(ctx, tenantID)
}

func importHash(mode string, rows []ImportRow) string {
	data, _ := json.Marshal(struct {
		Mode string
		Rows []ImportRow
	}{mode, rows})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

var importColumns = []string{"product_name", "variant_name", "sku", "barcode", "selling_price", "opening_quantity"}

func parseImportCSV(reader io.Reader) ([]ImportRow, error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	header, err := csvReader.Read()
	if err != nil {
		return nil, ErrImportFile
	}
	columns := map[string]int{}
	for index, name := range header {
		name = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(name, "\ufeff")))
		if _, exists := columns[name]; exists {
			return nil, ErrImportFile
		}
		columns[name] = index
	}
	for _, name := range importColumns {
		if _, exists := columns[name]; !exists {
			return nil, ErrImportFile
		}
	}
	rows := make([]ImportRow, 0)
	for {
		fields, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, ErrImportFile
		}
		if len(rows) == maxImportRows {
			return nil, ErrImportFile
		}
		at := func(name string) string {
			index := columns[name]
			if index >= len(fields) {
				return ""
			}
			return strings.TrimSpace(fields[index])
		}
		line, _ := csvReader.FieldPos(0)
		rows = append(rows, ImportRow{Line: line, ProductName: at("product_name"), VariantName: at("variant_name"),
			SKU: at("sku"), Barcode: at("barcode"), SellingPrice: at("selling_price"), OpeningQuantity: at("opening_quantity")})
	}
	if len(rows) == 0 {
		return nil, ErrImportFile
	}
	return rows, nil
}

func minorDigitsForCurrency(currency string) int {
	switch strings.TrimSpace(currency) {
	case "JPY", "KRW", "VND", "CLP", "PYG", "XOF", "XAF":
		return 0
	case "BHD", "JOD", "KWD", "OMR", "TND":
		return 3
	default:
		return 2
	}
}

func parseImportMoney(raw, currency string) (int64, error) {
	digits := minorDigitsForCurrency(currency)
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" || len(parts) == 2 && (digits == 0 || len(parts[1]) == 0 || len(parts[1]) > digits) {
		return 0, ErrInvalid
	}
	for _, part := range parts {
		for _, c := range part {
			if c < '0' || c > '9' {
				return 0, ErrInvalid
			}
		}
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole > 1_000_000_000_000 {
		return 0, ErrInvalid
	}
	multiplier := int64(1)
	for i := 0; i < digits; i++ {
		multiplier *= 10
	}
	if whole > 1_000_000_000_000/multiplier {
		return 0, ErrInvalid
	}
	value := whole * multiplier
	if len(parts) == 2 {
		fraction, err := strconv.ParseInt(parts[1]+strings.Repeat("0", digits-len(parts[1])), 10, 64)
		if err != nil {
			return 0, ErrInvalid
		}
		value += fraction
	}
	if value > 1_000_000_000_000 {
		return 0, ErrInvalid
	}
	return value, nil
}

func formatImportMoney(minor int64, currency string) string {
	digits := minorDigitsForCurrency(currency)
	if digits == 0 {
		return strconv.FormatInt(minor, 10)
	}
	multiplier := int64(1)
	for i := 0; i < digits; i++ {
		multiplier *= 10
	}
	return fmt.Sprintf("%d.%0*d", minor/multiplier, digits, minor%multiplier)
}
