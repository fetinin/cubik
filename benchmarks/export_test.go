package benchmarks

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// Test data structures
type SavedAnimation struct {
	ID       string
	DeviceID string
	Name     string
	Frames   [][]Color
}

type Color struct {
	R, G, B uint8
}

// Benchmark structs for each format
type RawJSONFormat struct {
	Name   string    `json:"name"`
	Width  int       `json:"width"`
	Height int       `json:"height"`
	Frames [][][]int `json:"frames"` // [[[r,g,b], ...], ...]
}

type RowRLEEntry struct {
	Count int `json:"count"`
	Color int `json:"color"` // 0xRRGGBB
}

type RowRLEFormat struct {
	Name   string            `json:"name"`
	Width  int               `json:"width"`
	Height int               `json:"height"`
	Frames [][][]RowRLEEntry `json:"rows"` // per-frame: per-row runs
}

type SparsePixel struct {
	X int `json:"x"`
	Y int `json:"y"`
	C int `json:"c"` // 0xRRGGBB
}

type SparseFormat struct {
	Name   string          `json:"name"`
	Width  int             `json:"width"`
	Height int             `json:"height"`
	Frames [][]SparsePixel `json:"frames"`
}

// Helper: pack RGB to int
func packRGB(r, g, b uint8) int {
	return (int(r) << 16) | (int(g) << 8) | int(b)
}

// Format 1: Raw JSON encoder
func encodeRawJSON(anim *SavedAnimation) ([]byte, error) {
	frames := make([][][]int, len(anim.Frames))
	for fi, frame := range anim.Frames {
		pixels := make([][]int, len(frame))
		for pi, pixel := range frame {
			pixels[pi] = []int{int(pixel.R), int(pixel.G), int(pixel.B)}
		}
		frames[fi] = pixels
	}

	data := RawJSONFormat{
		Name:   anim.Name,
		Width:  20, // TODO: compute from frame
		Height: 5,
		Frames: frames,
	}
	return json.MarshalIndent(data, "", "  ")
}

// Format 2: Row-RLE JSON encoder
func encodeRowRLEJSON(anim *SavedAnimation) ([]byte, error) {
	width := 20 // standard matrix width
	frames := make([][][]RowRLEEntry, len(anim.Frames))

	for fi, frame := range anim.Frames {
		rows := make([][]RowRLEEntry, 5) // height
		for y := range 5 {
			var runs []RowRLEEntry
			count := 0
			lastColor := -1

			for x := range width {
				idx := y*width + x
				if idx >= len(frame) {
					break
				}
				color := packRGB(frame[idx].R, frame[idx].G, frame[idx].B)

				if color == lastColor {
					count++
				} else {
					if lastColor != -1 {
						runs = append(runs, RowRLEEntry{Count: count, Color: lastColor})
					}
					lastColor = color
					count = 1
				}
			}
			if lastColor != -1 {
				runs = append(runs, RowRLEEntry{Count: count, Color: lastColor})
			}
			rows[y] = runs
		}
		frames[fi] = rows
	}

	data := RowRLEFormat{
		Name:   anim.Name,
		Width:  width,
		Height: 5,
		Frames: frames,
	}
	return json.MarshalIndent(data, "", "  ")
}

// Format 3: Sparse JSON encoder
func encodeSparseJSON(anim *SavedAnimation) ([]byte, error) {
	width := 20
	height := 5
	black := packRGB(0, 0, 0)

	frames := make([][]SparsePixel, len(anim.Frames))
	for fi, frame := range anim.Frames {
		var pixels []SparsePixel
		for i, pixel := range frame {
			color := packRGB(pixel.R, pixel.G, pixel.B)
			if color != black {
				pixels = append(pixels, SparsePixel{
					X: i % width,
					Y: i / width,
					C: color,
				})
			}
		}
		frames[fi] = pixels
	}

	data := SparseFormat{
		Name:   anim.Name,
		Width:  width,
		Height: height,
		Frames: frames,
	}
	return json.MarshalIndent(data, "", "  ")
}

type FrameJSON struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

func deserializeAnimation(jsonStr string) (*SavedAnimation, error) {
	var jsonFrames [][][]FrameJSON
	if err := json.Unmarshal([]byte(jsonStr), &jsonFrames); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frames: %w", err)
	}

	frames := make([][]Color, len(jsonFrames))
	for i, jsonFrameWrapper := range jsonFrames {
		if len(jsonFrameWrapper) > 0 {
			jsonFrame := jsonFrameWrapper[0]
			frame := make([]Color, len(jsonFrame))
			for j, pixel := range jsonFrame {
				frame[j] = Color{R: pixel.R, G: pixel.G, B: pixel.B}
			}
			frames[i] = frame
		}
	}

	return &SavedAnimation{
		Frames: frames,
	}, nil
}

// TestExportFormats compares size of different export formats
func TestExportFormats(t *testing.T) {
	db, err := sql.Open("sqlite", "../cubik.db")
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, device_id, name, frames_json FROM saved_animations")
	if err != nil {
		t.Fatalf("Failed to query animations: %v", err)
	}
	defer rows.Close()

	type Result struct {
		Name        string
		DBSize      int
		RawJSONSize int
		RLEJSONSize int
		SparseSize  int
	}

	var results []Result

	for rows.Next() {
		var id, deviceID, name, framesJSON string
		if err := rows.Scan(&id, &deviceID, &name, &framesJSON); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		anim, err := deserializeAnimation(framesJSON)
		if err != nil {
			t.Fatalf("Failed to deserialize: %v", err)
		}
		anim.Name = name

		// Encode to each format
		rawJSON, _ := encodeRawJSON(anim)
		rleJSON, _ := encodeRowRLEJSON(anim)
		sparseJSON, _ := encodeSparseJSON(anim)

		results = append(results, Result{
			Name:        name,
			DBSize:      len(framesJSON),
			RawJSONSize: len(rawJSON),
			RLEJSONSize: len(rleJSON),
			SparseSize:  len(sparseJSON),
		})

		// Optionally write files for inspection
		outputDir := "../benchmark_output"
		os.MkdirAll(outputDir, 0755)
		_ = os.WriteFile(filepath.Join(outputDir, fmt.Sprintf("%s_raw.json", name)), rawJSON, 0644)
		_ = os.WriteFile(filepath.Join(outputDir, fmt.Sprintf("%s_rle.json", name)), rleJSON, 0644)
		_ = os.WriteFile(filepath.Join(outputDir, fmt.Sprintf("%s_sparse.json", name)), sparseJSON, 0644)
	}

	// Print comparison table
	t.Log("\n=== Size Comparison (bytes) ===")
	t.Log("Name\t\tDB\tRaw\tRLE\tSparse\tRLE%\tSparse%")
	for _, r := range results {
		rlePct := float64(r.RLEJSONSize) / float64(r.RawJSONSize) * 100
		sparsePct := float64(r.SparseSize) / float64(r.RawJSONSize) * 100
		t.Logf("%s\t%d\t%d\t%d\t%d\t%.1f%%\t%.1f%%",
			r.Name, r.DBSize, r.RawJSONSize, r.RLEJSONSize, r.SparseSize, rlePct, sparsePct)
	}
}

// Benchmark encoding speed for each format
func BenchmarkEncodeRawJSON(b *testing.B) {
	anim := loadTestAnimation()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encodeRawJSON(anim)
	}
}

func BenchmarkEncodeRowRLEJSON(b *testing.B) {
	anim := loadTestAnimation()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encodeRowRLEJSON(anim)
	}
}

func BenchmarkEncodeSparseJSON(b *testing.B) {
	anim := loadTestAnimation()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encodeSparseJSON(anim)
	}
}

func loadTestAnimation() *SavedAnimation {
	db, err := sql.Open("sqlite", "../cubik.db")
	if err != nil {
		return &SavedAnimation{}
	}
	defer db.Close()

	var framesJSON string
	err = db.QueryRow("SELECT frames_json FROM saved_animations LIMIT 1").Scan(&framesJSON)
	if err != nil {
		return &SavedAnimation{}
	}

	anim, _ := deserializeAnimation(framesJSON)
	return anim
}
