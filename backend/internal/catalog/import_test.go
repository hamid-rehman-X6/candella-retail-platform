package catalog

import (
	"fmt"
	"strings"
	"testing"
)

func TestImportCSVValidationAndMoney(t *testing.T) {
	rows, err := parseImportCSV(strings.NewReader("\ufeffproduct_name,variant_name,sku,barcode,selling_price,opening_quantity\nShirt,Small,,,149.50,7\nShirt,Large,,,199,0\n"))
	if err != nil || len(rows) != 2 || rows[0].Line != 2 || rows[1].Line != 3 {
		t.Fatalf("parsed rows=%+v, err=%v", rows, err)
	}
	for _, input := range []string{
		"product_name,variant_name,sku,barcode,selling_price\nShirt,Small,,,149\n",
		"product_name,variant_name,sku,sku,barcode,selling_price,opening_quantity\nShirt,Small,,,,149,0\n",
		"product_name,variant_name,sku,barcode,selling_price,opening_quantity\nShirt,Small,,,149,0\n\"unterminated\n",
	} {
		if _, err := parseImportCSV(strings.NewReader(input)); err != ErrImportFile {
			t.Fatalf("invalid CSV accepted: %q, err=%v", input, err)
		}
	}
	if value, err := parseImportMoney("149.50", "PKR"); err != nil || value != 14950 {
		t.Fatalf("PKR conversion=%d, err=%v", value, err)
	}
	if value, err := parseImportMoney("149", "JPY"); err != nil || value != 149 {
		t.Fatalf("JPY conversion=%d, err=%v", value, err)
	}
	for _, value := range []string{"-1", "1.999", "1,000", "10000000000000"} {
		if _, err := parseImportMoney(value, "PKR"); err == nil {
			t.Fatalf("invalid money accepted: %q", value)
		}
	}
	for _, value := range []string{"=1+1", " +SUM(A1)", "\t@evil"} {
		if got := spreadsheetSafe(value); !strings.HasPrefix(got, "'") {
			t.Fatalf("formula was not escaped: %q -> %q", value, got)
		}
	}
	var large strings.Builder
	large.WriteString("product_name,variant_name,sku,barcode,selling_price,opening_quantity\n")
	for i := 0; i <= maxImportRows; i++ {
		fmt.Fprintf(&large, "Product %d,Standard,,,,0\n", i)
	}
	if _, err := parseImportCSV(strings.NewReader(large.String())); err != ErrImportFile {
		t.Fatalf("oversized row count was accepted: %v", err)
	}
}

func TestImportHashBindsModeAndRows(t *testing.T) {
	rows := []ImportRow{{ProductName: "Soap", SellingPrice: "150", OpeningQuantity: "4"}}
	original := importHash("create", rows)
	if original == importHash("update", rows) {
		t.Fatal("mode did not affect import hash")
	}
	rows[0].OpeningQuantity = "5"
	if original == importHash("create", rows) {
		t.Fatal("changed opening stock did not affect import hash")
	}
}
