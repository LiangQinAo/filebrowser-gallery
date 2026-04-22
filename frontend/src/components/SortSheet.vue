<template>
  <Teleport to="body">
    <Transition name="sheet-slide">
      <div v-if="visible" id="sort-sheet-overlay" @click.self="emit('close')">
        <div class="sort-sheet" role="dialog" aria-labelledby="sort-sheet-title">
          <div class="sheet-handle" />
          <h3 id="sort-sheet-title">{{ t("files.sortBy") }}</h3>

          <!-- Sort field options -->
          <ul class="sort-options">
            <li
              v-for="option in sortOptions"
              :key="option.key"
              :class="{ active: currentSort.by === option.key }"
              @click="selectSort(option.key)"
            >
              <i class="material-icons sort-icon">{{ option.icon }}</i>
              <span>{{ option.label }}</span>
              <i
                v-if="currentSort.by === option.key"
                class="material-icons sort-dir-icon"
              >
                {{ currentSort.asc ? "arrow_upward" : "arrow_downward" }}
              </i>
              <i v-else class="material-icons sort-dir-placeholder">unfold_more</i>
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
  (e: "sort", by: string, asc: boolean): void;
}>();

const { t } = useI18n();
const fileStore = useFileStore();

const currentSort = computed(() => ({
  by: fileStore.req?.sorting?.by ?? "name",
  asc: fileStore.req?.sorting?.asc ?? false,
}));

const sortOptions = computed(() => [
  { key: "name", label: t("files.name"), icon: "title" },
  { key: "size", label: t("files.size"), icon: "data_usage" },
  { key: "modified", label: t("files.lastModified"), icon: "schedule" },
]);

const selectSort = (by: string) => {
  let asc = false;
  if (currentSort.value.by === by) {
    // Toggle direction
    asc = !currentSort.value.asc;
  }
  emit("sort", by, asc);
  emit("close");
};
</script>

<style scoped>
#sort-sheet-overlay {
  position: fixed;
  inset: 0;
  z-index: 8888;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: flex-end;
  justify-content: stretch;
}

.sort-sheet {
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

.sort-options {
  list-style: none;
  margin: 0;
  padding: 0;
}

.sort-options li {
  display: flex;
  align-items: center;
  gap: 0.8em;
  padding: 0.9em 1.2em;
  cursor: pointer;
  color: var(--textPrimary, #222);
  transition: background 0.12s ease;
  user-select: none;
}

.sort-options li:hover {
  background: var(--hover, rgba(0, 0, 0, 0.05));
}

.sort-options li.active {
  color: var(--blue, #5c7cfa);
  font-weight: 600;
}

.sort-options li .sort-icon {
  font-size: 1.3em;
  opacity: 0.75;
}

.sort-options li span {
  flex: 1;
  font-size: 1em;
}

.sort-options li .sort-dir-icon {
  font-size: 1.1em;
}

.sort-options li .sort-dir-placeholder {
  font-size: 1.1em;
  opacity: 0.25;
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

/* Slide-up transition */
.sheet-slide-enter-active,
.sheet-slide-leave-active {
  transition: opacity 0.2s ease;
}
.sheet-slide-enter-active .sort-sheet,
.sheet-slide-leave-active .sort-sheet {
  transition: transform 0.25s cubic-bezier(0.32, 0.72, 0, 1);
}
.sheet-slide-enter-from,
.sheet-slide-leave-to {
  opacity: 0;
}
.sheet-slide-enter-from .sort-sheet,
.sheet-slide-leave-to .sort-sheet {
  transform: translateY(100%);
}
</style>
