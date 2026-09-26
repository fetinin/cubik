# Animation Export Format Research

## Problem Statement

We need a way to share animations between users and devices while minimizing file size. The format should be:

1. **Compact** - Small file size for easy sharing and storage
2. **Human-readable** - Easy to understand and modify manually if needed
3. **Simple** - Easy to parse and implement in different clients

## Background

Cubik animations consist of frames displayed on a 20x5 LED matrix (100 pixels per frame). Each pixel is an RGB color with values 0-255.

Typical animation patterns show:
- Lots of black/off pixels (LED matrix sparsity)
- Horizontal runs of repeated colors (text, shapes)
- Frame-by-frame color changes for animation effects

The current database storage uses a nested JSON structure:
```json
[[[{r:0,g:0,b:0}, {r:255,g:0,b:0}, ...]], ...]
```

## Formats Evaluated

### 1. Raw JSON (Baseline)

Straightforward representation of each pixel's RGB values.

```json
{
  "name": "Animation",
  "width": 20,
  "height": 5,
  "frames": [
    [[0,0,0], [0,0,0], [255,0,0], ...],
    ...
  ]
}
```

**Pros**: Simple, no compression logic needed
**Cons**: Large file size, lots of redundancy

### 2. Row-RLE JSON

Run-length encoding per row. Each row is a list of `[count, color]` runs.

```json
{
  "name": "Animation",
  "width": 20,
  "height": 5,
  "rows": [
    [[{"count": 10, "color": 0}, {"count": 5, "color": 0xFF0000}], ...],
    ...
  ]
}
```

**Pros**: Excellent for horizontal runs of same color
**Cons**: Overhead per run, larger for sparse animations with isolated pixels

### 3. Sparse JSON (Selected)

Only stores non-black pixels with their coordinates.

```json
{
  "name": "Animation",
  "width": 20,
  "height": 5,
  "frames": [
    [{"x": 9, "y": 0, "c": 0xDBF032}, {"x": 8, "y": 1, "c": 0xF03906}, ...],
    ...
  ]
}
```

**Pros**: Most compact for typical LED matrix usage (lots of black pixels)
**Cons**: Slightly more complex parsing

## Benchmark Results

Tested with two existing animations from the database:

| Animation | Raw JSON | Row-RLE | Sparse | RLE % | Sparse % |
|-----------|----------|---------|--------|-------|----------|
| Christmas tree | 15,030 | 5,082 | 4,132 | 33.8% | 27.5% |
| Red Flash Test | 306 | 505 | 379 | 165.0% | 123.9% |

### Encoding Performance

| Format | ns/op | Allocations |
|--------|-------|-------------|
| Raw JSON | 52,049 | 311 |
| Row-RLE | 19,593 | 61 |
| Sparse | 18,149 | 24 |

## Analysis

### Sparse Format Wins for Typical Usage

The **Sparse format** provides the best compression (27.5% of raw size) for complex animations like "Christmas tree". This is because:

1. **Black pixels dominate**: LED matrix animations typically have most pixels off
2. **No structural overhead**: Only stores actual lit pixels
3. **Fastest encoding**: Fewest allocations during serialization

### Row-RLE is Mixed

Row-RLE performs well for animations with horizontal runs but can **increase** file size for:
- Sparse animations with isolated pixels
- Animations with frequent color changes per row

The "Red Flash Test" demonstrates this - it has only isolated pixels with no horizontal runs, so RLE's structural overhead (objects with keys like `count`, `color`) makes it 65% larger than raw JSON.

### Raw JSON is Too Verbose

While simplest to implement, raw JSON produces the largest files. For sharing animations over networks or storing many animations, this is impractical.

## Decision

**Selected Format: Sparse JSON**

Rationale:
- Best compression for typical LED matrix animations
- Fastest encoding performance
- Still human-readable and easy to parse
- Compact even for animations without horizontal runs

## Running Benchmarks

To reproduce these results:

```bash
cd benchmarks
go test -v -bench=. -benchmem
```

Sample output files are generated in `benchmark_output/` for inspection.

## Future Considerations

- Add gzip compression on top of sparse format for network transfer
- Consider binary format for embedded devices with limited JSON parsing
- Evaluate frame-delta encoding for smooth animations
