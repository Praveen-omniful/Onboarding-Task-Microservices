package csv

import (
	"encoding/csv"
	"io"
)

type OrderRow struct {
	OrderID string
	SKU     string
	Hub     string
	Qty     string
}

// ParseCSV parses CSV data and returns a slice of OrderRow
func ParseCSV(r io.Reader) ([]OrderRow, error) {
	reader := csv.NewReader(r)
	var rows []OrderRow
	_, _ = reader.Read() // skip header
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(rec) < 4 {
			continue
		}
		rows = append(rows, OrderRow{
			OrderID: rec[0],
			SKU:     rec[1],
			Hub:     rec[2],
			Qty:     rec[3],
		})
	}
	return rows, nil
}
