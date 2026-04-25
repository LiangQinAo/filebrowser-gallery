# filebrowser-gallery

> A high-performance NAS media gallery, forked and enhanced from [filebrowser/filebrowser](https://github.com/filebrowser/filebrowser).

## ✨ Enhancements over upstream

- **Media Gallery View** — iOS-style fullscreen gallery with smooth PhotoSwipe gestures (pinch-to-zoom, swipe navigation)
- **RAW Image Support** — Preview RAW camera formats (`.dng`, `.raf`, `.arw`, etc.) via macOS `sips` fallback
- **EXIF Metadata Panel** — Display photo metadata (camera model, aperture, shutter speed, GPS, etc.) in the Info panel
- **Mobile-Friendly UI** — Bottom sheet filter/sort panels optimized for touch devices
- **Non-Interruptive Deletion** — Delete images directly from within the gallery viewer without closing the lightbox
- **Selective Batch Operations** — Multi-select images in gallery view for batch actions
- **Version API** — `/api/version` endpoint exposing build version info

## 🔗 Upstream

This project is based on [File Browser](https://github.com/filebrowser/filebrowser) (Apache 2.0).  
Original upstream branch is preserved as `upstream/master` for easy diff and rebase.

## 🚀 Quick Start

```bash
# Build frontend
cd frontend && pnpm install && pnpm build

# Build binary
go build -o filebrowser .

# Run
./filebrowser --config config/settings.json
```

Or use the provided Docker Compose setup in [nas-deployment](https://github.com/LiangQinAo/nas-deployment).

## License

[Apache License 2.0](LICENSE) — Original © File Browser Contributors, enhancements © LiangQinAo
