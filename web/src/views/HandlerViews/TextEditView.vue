<template>
  <div ref="el" class="text-edit-view" @keydown="onKeyDown">
    <handler-title-bar :title="filename" @close="emit('close')">
      <template #actions>
        <simple-button v-if="!readonly" :loading="saving" @click="saveFile">
          {{ $t('hv.text_edit.save') }}
        </simple-button>
      </template>
    </handler-title-bar>
    <text-editor
      v-if="!error"
      v-model="content"
      :filename="filename"
      line-numbers
      :disabled="readonly"
    />
    <error-view v-else :status="error.status" :message="error.message" />
    <div v-if="!inited" class="loading-tips">Loading...</div>
  </div>
</template>
<script setup>
import { filename as filenameFn } from '@/utils'
import { getContent } from '@/api'
import TextEditor from '@/components/TextEditor/index.vue'
import HandlerTitleBar from '@/components/HandlerTitleBar.vue'
import uploadManager from '@/api/upload-manager'
import { alert } from '@/utils/ui-utils'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({
  entry: {
    type: Object,
    required: true,
  },
  entries: { type: Array },
})

const emit = defineEmits(['close', 'save-state'])

const error = ref(null)
const inited = ref(false)

const content = ref('')

const saving = ref(false)

const path = computed(() => props.entry.path)

const filename = computed(() => filenameFn(path.value))

const readonly = computed(() => !props.entry.meta.writable)

const el = ref(null)

const loadFile = async () => {
  inited.value = false
  try {
    return await loadFileContent()
  } catch (e) {
    error.value = e
  } finally {
    inited.value = true
  }
}

const loadFileContent = async () => {
  content.value = await getContent(path.value, props.entry.meta, {
    noCache: true,
  })
  nextTick(() => {
    changeSaveState(true)
  })
  return content.value
}

const saveFile = async () => {
  if (saving.value) {
    return
  }
  saving.value = true
  try {
    await uploadManager.upload(
      {
        path: path.value,
        file: content.value,
        override: true,
      },
      true
    )
    changeSaveState(true)
  } catch (e) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

const changeSaveState = (saved) => {
  emit('save-state', saved)
}

const onKeyDown = (e) => {
  if (e.key === 's' && e.ctrlKey && !readonly.value) {
    e.preventDefault()
    saveFile()
  }
}

const onWindowResize = () => {
  if (window.innerWidth <= 800) {
    el.value.style.height = `${window.innerHeight}px`
  }
}

watch(
  () => content.value,
  () => {
    changeSaveState(false)
  }
)

onMounted(() => {
  window.addEventListener('resize', onWindowResize)
  onWindowResize()
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', onWindowResize)
})

loadFile()
</script>
<style lang="scss">
.text-edit-view {
  position: relative;
  width: 800px;
  height: calc(100vh - 64px);
  padding-top: 48px;
  background-color: var(--secondary-bg-color);
  overflow: hidden;
  box-sizing: border-box;
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);

  .handler-title-bar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }

  .text-editor {
    height: 100%;

    .CodeMirror {
      height: 100%;
    }
  }

  .loading-tips {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    justify-content: center;
    align-items: center;
    width: 100%;
    height: 300px;
    font-weight: bold;
    font-size: 24px;
    text-transform: uppercase;
    user-select: none;
  }
}

@media screen and (max-width: 800px) {
  .text-edit-view {
    width: 100vw;
    height: 100vh;
    max-width: unset;
    margin: 0;
  }
}
</style>
