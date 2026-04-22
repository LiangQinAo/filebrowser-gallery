<template>
  <Teleport to="body">
    <Transition name="sheet-slide">
      <div v-if="visible" id="filter-sheet-overlay" @click.self="emit('close')">
        <div class="filter-sheet" role="dialog" aria-labelledby="filter-sheet-title">
          <div class="sheet-handle" />
          <h3 id="filter-sheet-title">{{ t('buttons.filter') }}</h3>

          <ul class="filter-options">
            <li
              v-for="option in filterOptions"
              :key="option.key"
              :class="{ active: fileStore.typeFilter === option.key }"
              @click="selectFilter(option.key)"
            >
              <i class="material-icons filter-icon">{{ option.icon }}</i>
              <span>{{ option.label }}</span>
              <i
                v-if="fileStore.typeFilter === option.key"
                class="material-icons filter-check-icon"
              >check</i>
            </li>
          </ul>

          <button class="sheet-close-btn" @click="emit('close')">
            {{ t("buttons.close") }}
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useFileStore } from "@/stores/file";

const props = defineProps<{
  visible: boolean;
}>();

const emit = defineEmits<{
  (e: "close"): void;
}>();

const { t } = useI18n();
const fileStore = useFileStore();

const filterOptions = computed(() => [
  { key: "all", label: "全部 (All)", icon: "all_inclusive" },
  { key: "image", label: "图片 (Images)", icon: "image" },
  { key: "video", label: "视频 (Videos)", icon: "movie" },
  { key: "audio", label: "音频 (Audio)", icon: "audiotrack" },
  { key: "text", label: "文档 (Documents)", icon: "description" },
]);

const selectFilter = (key: string) => {
  fileStore.typeFilter = key;
  emit("close");
};
</script>

<style scoped>
#filter-sheet-overlay {
  position: fixed;
  inset: 0;
  z-index: 8888;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: flex-end;
  justify-content: stretch;
}

.filter-sheet {
  width: 100%;
  background: var(--surfacePrimary, #fff);
  border-radius: 1.2em 1.2em 0 0;
  padding: 0.5em 0 calc(env(safe-area-inset-bottom, 0px) + 1em);
  box-shadow: 0 -4px 24px rgba(0, 0, 0, 0.18);
  max-height: 70vh;
  overflow-y: auto;
}

.sheet-handle {
  width: 2.5em;
  height: 0.28em;
  margin: 0.4em auto 0.8em;
  border-radius: 2em;
  background: var(--divider, rgba(0, 0, 0, 0.15));
}

h3 {
  margin: 0 1.2em 0.6em;
  font-size: 1em;
  font-weight: 600;
  color: var(--textSecondary, #666);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.filter-options {
  list-style: none;
  margin: 0;
  padding: 0;
}

.filter-options li {
  display: flex;
  align-items: center;
  gap: 0.8em;
  padding: 0.9em 1.2em;
  cursor: pointer;
  color: var(--textPrimary, #222);
  transition: background 0.12s ease;
  user-select: none;
}

.filter-options li:hover {
  background: var(--hover, rgba(0, 0, 0, 0.05));
}

.filter-options li.active {
  color: var(--blue, #5c7cfa);
  font-weight: 600;
}

.filter-options li .filter-icon {
  font-size: 1.3em;
  opacity: 0.75;
}

.filter-options li span {
  flex: 1;
  font-size: 1em;
}

.filter-options li .filter-check-icon {
  font-size: 1.1em;
  color: var(--blue, #5c7cfa);
}

.sheet-close-btn {
  display: block;
  margin: 0.8em 1.2em 0;
  width: calc(100% - 2.4em);
  padding: 0.75em;
  border: none;
  border-radius: 0.6em;
  background: var(--hover, rgba(0, 0, 0, 0.06));
  color: var(--textPrimary, #222);
  font-size: 1em;
  cursor: pointer;
  transition: background 0.12s ease;
}

.sheet-close-btn:hover {
  background: var(--hover, rgba(0, 0, 0, 0.12));
}

.sheet-slide-enter-active,
.sheet-slide-leave-active {
  transition: opacity 0.2s ease;
}
.sheet-slide-enter-active .filter-sheet,
.sheet-slide-leave-active .filter-sheet {
  transition: transform 0.25s cubic-bezier(0.32, 0.72, 0, 1);
}
.sheet-slide-enter-from,
.sheet-slide-leave-to {
  opacity: 0;
}
.sheet-slide-enter-from .filter-sheet,
.sheet-slide-leave-to .filter-sheet {
  transform: translateY(100%);
}
</style>
