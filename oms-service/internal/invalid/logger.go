package invalid

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// InvalidRecord represents an invalid row from CSV processing
type InvalidRecord struct {
	RowNumber    int                    `json:"row_number"`
	Data         map[string]interface{} `json:"data"`
	Errors       []string               `json:"errors"`
	Timestamp    time.Time              `json:"timestamp"`
	OriginalFile string                 `json:"original_file"`
}

// Logger handles logging of invalid CSV records
type Logger struct {
	outputDir string
	mu        sync.RWMutex
	files     map[string]*os.File
}

// NewLogger creates a new invalid record logger
func NewLogger(outputDir string) (*Logger, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &Logger{
		outputDir: outputDir,
		files:     make(map[string]*os.File),
	}, nil
}

// LogInvalidRecord logs an invalid record to a CSV file
func (l *Logger) LogInvalidRecord(ctx context.Context, record *InvalidRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Generate filename based on original file and date
	now := time.Now()
	filename := fmt.Sprintf("invalid_%s_%s.csv",
		sanitizeFilename(record.OriginalFile),
		now.Format("2006-01-02"))

	filepath := filepath.Join(l.outputDir, filename)

	// Check if file exists or needs to be created
	file, exists := l.files[filename]
	var needsHeader bool

	if !exists {
		var err error
		file, err = os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open/create invalid records file: %w", err)
		}
		l.files[filename] = file

		// Check if file is empty (needs header)
		stat, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat file: %w", err)
		}
		needsHeader = stat.Size() == 0
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if needed
	if needsHeader {
		header := []string{
			"row_number",
			"timestamp",
			"original_file",
			"errors",
			"order_id",
			"customer_name",
			"customer_email",
			"product_name",
			"sku",
			"hub_id",
			"quantity",
			"unit_price",
			"total_amount",
			"order_date",
			"shipping_address",
			"status",
		}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write CSV header: %w", err)
		}
	}

	// Write the invalid record
	row := []string{
		fmt.Sprintf("%d", record.RowNumber),
		record.Timestamp.Format(time.RFC3339),
		record.OriginalFile,
		fmt.Sprintf("%v", record.Errors),
		getStringValue(record.Data, "order_id"),
		getStringValue(record.Data, "customer_name"),
		getStringValue(record.Data, "customer_email"),
		getStringValue(record.Data, "product_name"),
		getStringValue(record.Data, "sku"),
		getStringValue(record.Data, "hub_id"),
		getStringValue(record.Data, "quantity"),
		getStringValue(record.Data, "unit_price"),
		getStringValue(record.Data, "total_amount"),
		getStringValue(record.Data, "order_date"),
		getStringValue(record.Data, "shipping_address"),
		getStringValue(record.Data, "status"),
	}

	if err := writer.Write(row); err != nil {
		return fmt.Errorf("failed to write CSV row: %w", err)
	}

	log.Printf("📝 Logged invalid record to: %s (row %d)", filepath, record.RowNumber)
	return nil
}

// ListInvalidFiles returns a list of available invalid record files
func (l *Logger) ListInvalidFiles() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(l.outputDir, "invalid_*.csv"))
	if err != nil {
		return nil, fmt.Errorf("failed to list invalid files: %w", err)
	}

	// Return just the filenames, not full paths
	var filenames []string
	for _, file := range files {
		filenames = append(filenames, filepath.Base(file))
	}

	return filenames, nil
}

// GetInvalidFilePath returns the full path to an invalid records file
func (l *Logger) GetInvalidFilePath(filename string) string {
	return filepath.Join(l.outputDir, filename)
}

// Close closes all open files
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for filename, file := range l.files {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file %s: %v", filename, err)
		}
	}
	l.files = make(map[string]*os.File)
	return nil
}

// ListFiles returns a list of all invalid CSV files in the output directory
func (l *Logger) ListFiles() ([]string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries, err := os.ReadDir(l.outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".csv" {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// GetFilePath returns the full path to a specific invalid file
func (l *Logger) GetFilePath(filename string) (string, error) {
	// Ensure the filename has .csv extension
	if filepath.Ext(filename) != ".csv" {
		filename += ".csv"
	}

	fullPath := filepath.Join(l.outputDir, filename)

	// Check if file exists
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", filename)
		}
		return "", fmt.Errorf("error accessing file: %w", err)
	}

	return fullPath, nil
}

// Helper functions
func sanitizeFilename(filename string) string {
	// Remove file extension and invalid characters
	if ext := filepath.Ext(filename); ext != "" {
		filename = filename[:len(filename)-len(ext)]
	}
	// Use only the base name to avoid path issues
	filename = filepath.Base(filename)
	if filename == "" {
		filename = "unknown"
	}
	return filename
}

func getStringValue(data map[string]interface{}, key string) string {
	if val, exists := data[key]; exists && val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}
