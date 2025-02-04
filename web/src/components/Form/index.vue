<template>
  <form
    class="simple-form"
    :autocomplete="noAutoComplete ? 'off' : 'on'"
    @submit="onSubmit"
  >
    <form-item
      v-for="item in form"
      :key="item.field"
      :ref="addFieldsRef"
      v-model="data[item.field]"
      :item="item"
      :class="item.class"
      :style="{
        width: typeof item.width === 'string' ? item.width : item.width,
      }"
      @update:model-value="emitInput"
    >
      <template v-if="item.slot" #value>
        <slot :name="item.slot" />
      </template>
    </form-item>
  </form>
</template>
<script>
export default { name: 'FormView' }
</script>
<script setup>
import { onBeforeUpdate, ref, watch } from 'vue'
import FormItem from './FormItem.vue'

const props = defineProps({
  form: {
    type: Array,
    required: true,
  },
  modelValue: {
    type: Object,
  },
  noAutoComplete: {
    type: Boolean,
  },
})

const data = ref({})
let fields = []

const emit = defineEmits(['update:modelValue'])

const addFieldsRef = (el) => {
  if (el) fields.push(el)
}
onBeforeUpdate(() => {
  fields = []
})

watch(
  () => props.modelValue,
  (val) => {
    if (val === data.value) return
    data.value = val || {}
  },
  { immediate: true }
)

const validate = async () => {
  await Promise.all(fields.map((f) => f.validate()))
}

const clearError = () => {
  fields.forEach((f) => {
    f.clearError()
  })
}

defineExpose({ validate, clearError })

const onSubmit = (e) => e.preventDefault()

const emitInput = () => emit('update:modelValue', data.value)

const fillDefaultValue = () => {
  if (props.modelValue) return
  const dat = {}
  for (const f of props.form) {
    dat[f.field] = f.defaultValue || null
  }
  data.value = dat
  emitInput()
}

fillDefaultValue()
</script>
<style lang="scss">
.simple-form {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;

  .form-item {
    width: 232px;
  }
}
</style>
