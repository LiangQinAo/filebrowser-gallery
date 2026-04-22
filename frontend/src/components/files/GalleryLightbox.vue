<template>
  <!-- No template needed, PhotoSwipe manages its own DOM appended to body -->
</template>

<script setup lang="ts">
import { watch, onUnmounted } from "vue";
import PhotoSwipe from "photoswipe";
import "photoswipe/style.css";
import { files as api } from "@/api";
import { createURL } from "@/api/utils";
import { useFileStore } from "@/stores/file";
import { useAuthStore } from "@/stores/auth";

interface LightboxImage {
  name: string;
  path: string;
  modified: string;
  index?: number; // global index in req.items, used for multi-select
  resolution?: { width: number; height: number };
}

const props = defineProps<{
  images: LightboxImage[];
  startIndex: number;
  visible: boolean;
  multipleMode?: boolean;
  selectedIndices?: number[];
}>();

const emit = defineEmits<{
  (e: "close"): void;
  (e: "toggle-select", index: number): void;
}>();

const fileStore = useFileStore();
const authStore = useAuthStore();
let pswp: any = null;

// Browser-renderable image formats (can be shown natively without download)
const BROWSER_IMAGE_EXTS = new Set([
  ".jpg",
  ".jpeg",
  ".png",
  ".gif",
  ".webp",
  ".avif",
  ".bmp",
  ".svg",
]);
const isBrowserRenderable = (path: string) => {
  const ext = path.substring(path.lastIndexOf(".")).toLowerCase();
  return BROWSER_IMAGE_EXTS.has(ext);
};

// Update the select button UI to reflect the current slide's selection state.
// Uses DOM query so it can be called from outside createPhotoSwipe.
const updateSelectButtonUI = () => {
  if (!pswp) return;
  const item = pswp.currSlide?.data;
  if (!item || item.itemIndex === undefined) {
    console.log("[Lightbox] updateSelectButtonUI: no item or itemIndex", item);
    return;
  }
  const isSelected = props.selectedIndices?.includes(item.itemIndex) ?? false;
  console.log(
    `[Lightbox] updateSelectButtonUI: itemIndex=${item.itemIndex}, selectedIndices=${JSON.stringify(props.selectedIndices)}, isSelected=${isSelected}`
  );
  const btn = document.querySelector(
    ".pswp__button--select-button"
  ) as HTMLElement | null;
  if (!btn) {
    console.log("[Lightbox] updateSelectButtonUI: select button DOM not found");
    return;
  }
  const icon = btn.querySelector(".material-icons");
  if (icon)
    icon.textContent = isSelected ? "check_circle" : "check_circle_outline";
  btn.style.color = isSelected ? "#4caf50" : "";
};

// React when parent updates selectedIndices (after toggle-select emit)
watch(
  () => props.selectedIndices,
  (newVal) => {
    console.log("[Lightbox] selectedIndices changed:", JSON.stringify(newVal));
    updateSelectButtonUI();
  },
  { deep: true }
);

const createPhotoSwipe = () => {
  if (pswp) pswp.close();
  if (!props.images.length) return;

  // Do NOT pass width/height from resolution metadata: server returns sensor pixel
  // dimensions (pre-EXIF-rotation), while preview/big applies rotation → aspect mismatch.
  // Dimensions are resolved after load via the loadComplete event below.
  console.log(
    "[Lightbox] createPhotoSwipe: images=",
    props.images.map((i) => ({ name: i.name, index: i.index }))
  );
  console.log(
    `[Lightbox] createPhotoSwipe: startIndex=${props.startIndex}, multipleMode=${props.multipleMode}, selectedIndices=${JSON.stringify(props.selectedIndices)}`
  );

  const dataSource = props.images.map((img) => ({
    src: api.getPreviewURL(
      { path: img.path, modified: img.modified } as Resource,
      "big"
    ),
    msrc: api.getPreviewURL(
      { path: img.path, modified: img.modified } as Resource,
      "thumb"
    ),
    alt: img.name,
    path: img.path,
    itemIndex: img.index, // carry global index for selection
  }));

  pswp = new PhotoSwipe({
    dataSource,
    index: props.startIndex,
    bgOpacity: 0.92,
    showHideAnimationType: "fade",
    preload: [1, 3],
    wheelToZoom: true,
    pinchToClose: true,
    closeOnVerticalDrag: true,
  });

  pswp.on("uiRegister", () => {
    // ── Select toggle (only visible in multi-select mode) ──────────────────
    if (props.multipleMode) {
      pswp.ui.registerElement({
        name: "select-button",
        ariaLabel: "Select",
        order: 7,
        isButton: true,
        html: '<span class="material-icons pswp-custom-icon">check_circle_outline</span>',
        onInit: () => {
          updateSelectButtonUI();
        },
        onClick: () => {
          const item = pswp.currSlide.data;
          console.log(
            `[Lightbox] select-button clicked: itemIndex=${item.itemIndex}, alt=${item.alt}, selectedIndices=${JSON.stringify(props.selectedIndices)}`
          );
          if (item.itemIndex !== undefined) {
            emit("toggle-select", item.itemIndex);
          } else {
            console.warn(
              "[Lightbox] select-button: itemIndex is undefined!",
              item
            );
          }
        },
      });
    }

    // ── View original / full-resolution ───────────────────────────────────
    if (authStore.user?.perm.download) {
      pswp.ui.registerElement({
        name: "original-button",
        ariaLabel: "原图",
        order: 8,
        isButton: true,
        html: '<span class="material-icons pswp-custom-icon">hd</span>',
        onClick: () => {
          const slide = pswp.currSlide;
          const item = slide.data;
          const path = item.path as string;

          if (!isBrowserRenderable(path)) {
            // RAW / unsupported format → trigger download
            window.open(createURL("api/raw" + path, {}));
            return;
          }

          // For browser-renderable formats, swap the slide to the original file
          const originalUrl = createURL("api/raw" + path, {});
          const img = slide.content?.element as HTMLImageElement | null;
          if (!img) return;

          img.style.opacity = "0.5"; // loading indicator

          // Pre-load to get natural dimensions before swapping
          const probe = new Image();
          probe.onload = () => {
            slide.content.width = probe.naturalWidth;
            slide.content.height = probe.naturalHeight;
            img.src = originalUrl;
            img.addEventListener(
              "load",
              () => {
                img.style.opacity = "1";
                slide.updateContentSize(true);
              },
              { once: true }
            );
          };
          probe.onerror = () => {
            img.style.opacity = "1";
          };
          probe.src = originalUrl;
        },
      });

      // ── Download ───────────────────────────────────────────────────────
      pswp.ui.registerElement({
        name: "download-button",
        ariaLabel: "Download",
        order: 9,
        isButton: true,
        html: '<span class="material-icons pswp-custom-icon">file_download</span>',
        onClick: () => {
          const item = pswp.currSlide.data;
          window.open(createURL("api/raw" + item.path, {}));
        },
      });
    }

    // ── Delete ─────────────────────────────────────────────────────────────
    if (authStore.user?.perm.delete) {
      pswp.ui.registerElement({
        name: "delete-button",
        ariaLabel: "Delete",
        order: 10,
        isButton: true,
        html: '<span class="material-icons pswp-custom-icon pswp-custom-icon--danger">delete</span>',
        onClick: async () => {
          const item = pswp.currSlide.data;
          if (confirm(`删除图片 ${item.alt}？`)) {
            try {
              await api.remove(item.path);
              fileStore.reload = true;
              if (pswp.getNumItems() > 1) {
                pswp.next();
              } else {
                pswp.close();
              }
            } catch {
              alert("删除失败");
            }
          }
        },
      });
    }
  });

  // FIX: When no width/height is provided in dataSource, PS5 initializes
  // slide.width/slide.height = 0 (from content.width/height = 0), then falls back to
  // viewport size for zoom calculation. Even if we update content.width later, the
  // slide.width used by calculateSize() is still 0 → zoom = 1 → image fills viewport.
  //
  // Correct fix: update BOTH content AND slide dimensions, then recalculate zoom levels
  // and reset to fit position via calculateSize() + zoomAndPanToInitial().
  pswp.on("loadComplete", ({ content }: any) => {
    console.log(
      "[Lightbox] loadComplete: content.width=",
      content?.width,
      "content.height=",
      content?.height,
      "naturalWidth=",
      (content?.element as HTMLImageElement)?.naturalWidth,
      "naturalHeight=",
      (content?.element as HTMLImageElement)?.naturalHeight
    );
    if (!content || content.type !== "image") return;
    const img = content.element;
    if (!(img instanceof HTMLImageElement) || !img.naturalWidth) return;

    const w = img.naturalWidth;
    const h = img.naturalHeight;

    content.width = w;
    content.height = h;

    const slide = content.slide;
    if (!slide) return;

    // PS5 full re-layout sequence (mirrors what slide.setContent does internally):
    // 1. update slide dimensions (used by calculateSize + updateContentSize)
    slide.width = w;
    slide.height = h;
    // 2. recalculate zoom levels (fit/fill/initial) using correct dimensions
    slide.calculateSize();
    // 3. reset currentResolution so updateContentSize uses zoomLevels.initial
    slide.currentResolution = 0;
    // 4. resize the <img> element to the correct pixel size
    slide.updateContentSize(true);
    // 5. reset pan/zoom state
    slide.zoomAndPanToInitial();
    // 6. apply CSS transform to DOM
    slide.applyCurrentZoomPan();

    console.log(
      "[Lightbox] loadComplete fixed: w=",
      w,
      "h=",
      h,
      "isActive=",
      slide.isActive,
      "initialZoom=",
      slide.zoomLevels?.initial
    );
  });

  // Update select button state when active slide changes
  pswp.on("change", () => {
    const d = pswp.currSlide?.data;
    console.log(
      `[Lightbox] slide change: itemIndex=${d?.itemIndex}, alt=${d?.alt}, selectedIndices=${JSON.stringify(props.selectedIndices)}`
    );
    updateSelectButtonUI();
  });

  pswp.on("close", () => {
    pswp = null;
    emit("close");
  });

  pswp.init();
};

watch(
  () => props.visible,
  (v) => {
    if (v) {
      createPhotoSwipe();
    } else {
      if (pswp) pswp.close();
    }
  },
  { immediate: true }
);

onUnmounted(() => {
  if (pswp) pswp.close();
});
</script>

<style>
/* Ensure PhotoSwipe renders above all other UI */
.pswp {
  --pswp-bg: #000;
  z-index: 20000 !important;
}

/* Material Icons inside PhotoSwipe custom buttons */
.pswp__button .pswp-custom-icon {
  display: block;
  font-size: 22px;
  line-height: 50px;
  text-align: center;
}

.pswp__button .pswp-custom-icon--danger {
  color: #ff5252;
}

/* Select button active state */
.pswp__button--select-button .material-icons {
  transition: color 0.15s;
}
</style>
