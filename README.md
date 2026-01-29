# 🧼 bleach — scrub image metadata fast, safely, and beautifully

**bleach** is a high‑performance CLI that removes identifying metadata (EXIF, XMP, IPTC) from images.  
It uses a concurrency‑safe worker pool, magic‑byte sniffing (no extension trust), and atomic writes to keep your files safe.

---

## ✨ Highlights

- 🧵 **Concurrent worker pool** capped by CPU cores
- 🧠 **Magic‑byte detection** (JPEG/PNG/TIFF) — no reliance on file extensions
- 🧼 **Aggressive metadata stripping** for JPEG & PNG
- 🧪 **Dry‑run scan** that explains what data is present
- 🎨 **Live TUI progress** with a clean, high‑contrast palette
- 🧷 **Atomic writes** for safe, loss‑free output

---

## 🚀 Install

```bash
git clone https://github.com/noahlin34/bleach.git
cd bleach
go build -o bleach ./cmd/bleach
```

---

## 🧰 Usage

### Scan (read‑only)

```bash
bleach scan <path>
```

### Scan with insights (opt‑in, inferred)

```bash
bleach scan --insights <path>
```

### Clean (writes sanitized copies)

```bash
bleach clean <path>
```

### Clean in place

```bash
bleach clean --inplace <path>
```

### Clean to an output directory

```bash
bleach clean --output ./sanitized <path>
```

---

## 🧾 Example Output

### `bleach scan IMG_0047.png`

```
IMG_0047.png
  GPS:
    - GPSLatitudeRef=N
    - GPSLatitude=[43/1 51/1 4748/100]
    - GPSLongitudeRef=W
    - GPSLongitude=[79/1 19/1 5946/100]
  Device Model:
    - Make=Apple
    - Model=iPhone 14 Pro
  Timestamp:
    - DateTime=2024:01:03 15:56:06
    - DateTimeOriginal=2024:01:03 15:56:06
    - DateTimeDigitized=2024:01:03 15:56:06
```

### `bleach scan --insights IMG_0047.png`

```
IMG_0047.png
  ...
  Insights (inferred):
    - Location: Approx location: 43.65100, -79.34700
    - Location: Exact coordinates can reveal home, workplace, or travel patterns.
    - Device: Device: Apple iPhone 14 Pro (smartphone)
    - Timeline: Captured: 2024-01-03 15:56:06 (timezone unknown)
    - Timeline: Capture timestamps can expose routines and time zones.
```

---

## 🧠 What gets removed?

### JPEG
- EXIF (APP1)
- XMP (APP1)
- IPTC / Photoshop (APP13)
- ICC profile (APP2) unless `--preserve-icc`

### PNG
- `tEXt`, `zTXt`, `iTXt`
- `eXIf`
- `tIME`
- `iCCP` unless `--preserve-icc`

---

## 🏁 Flags

| Command | Flag | Description |
| --- | --- | --- |
| `scan` | `--insights` | Explain what metadata could reveal (inferred) |
| `clean` | `-i`, `--inplace` | Modify files in place |
| `clean` | `-o`, `--output` | Output directory for sanitized copies |
| `clean` | `--preserve-icc` | Keep ICC color profiles |

---

## 🛡️ Safety Guarantees

- **Atomic output:** writes to a `.tmp` file before replacing
- **No extension trust:** uses magic‑byte sniffing
- **Dry run:** scan mode never modifies files

---

## 🧭 Project Structure

```
/cmd                  Cobra CLI entrypoints
/internal/processor    Metadata scanning & stripping pipeline
/internal/tui          Bubble Tea models + lipgloss styling
/pkg/imgutil           Image sniffing utilities
```

---

## ✅ Roadmap

- [ ] TIFF stripping (currently not implemented)
- [ ] Optional offline geo‑insights
- [ ] Additional formats (HEIC/WebP)
- [ ] JSON output for automation pipelines

---

## 🤝 Contributing

PRs welcome!  
If you’re adding new file formats or metadata handlers, please include:

- unit tests
- sample fixtures (minimal, anonymized)
- a short explanation of the stripping approach

---

## 📄 License

MIT — see `LICENSE`.
