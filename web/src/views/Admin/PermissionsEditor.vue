<template>
  <div class="permissions">
    <table class="simple-table">
      <thead>
        <tr>
          <th>{{ $t('p.admin.p_edit.subject') }}</th>
          <th>{{ $t('p.admin.p_edit.rw') }}</th>
          <th>{{ $t('p.admin.p_edit.policy') }}</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(p, i) in permissions" :key="p.subject">
          <td class="center">
            <select v-model="p.subject">
              <option
                v-for="s in subjects"
                :key="s.subject"
                :value="s.subject"
                :disabled="selectedSubjects[s.subject]"
              >
                {{
                  s.type === 'any'
                    ? $t('p.admin.p_edit.any')
                    : `${s.type}: ${s.name}`
                }}
              </option>
            </select>
          </td>
          <td class="center">
            <input v-model="p.permission.read" type="checkbox" />
            <input v-model="p.permission.write" type="checkbox" />
          </td>
          <td class="center">
            <simple-button
              :title="$t('p.admin.p_edit.reject')"
              icon="#icon-reject"
              small
              :type="p.policy === 0 ? 'danger' : 'info'"
              @click="p.policy = 0"
            />
            <simple-button
              :title="$t('p.admin.p_edit.accept')"
              icon="#icon-accept"
              small
              :type="p.policy === 1 ? '' : 'info'"
              @click="p.policy = 1"
            />
          </td>
          <td>
            <simple-button
              type="danger"
              icon="#icon-delete"
              small
              @click="removePermission(i)"
            />
          </td>
        </tr>
        <tr>
          <td class="center" colspan="4">
            <simple-button icon="#icon-add" small @click="addPermission" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
<script setup>
import {
  getGroups,
  getPermissions,
  getUsers,
  savePermissions,
} from '@/api/admin'
import { mapOf } from '@/utils'
import { alert } from '@/utils/ui-utils'
import { computed, nextTick, ref, watch, watchEffect } from 'vue'

const PERMISSION_EMPTY = 0
const PERMISSION_READ = 1 << 0
const PERMISSION_WRITE = 1 << 1

const props = defineProps({
  path: {
    type: String,
    required: true,
  },
  modelValue: {
    type: Array,
  },
})

const emit = defineEmits(['update:modelValue', 'save-state'])

const permissions = ref([])
const subjects = ref([])

const selectedSubjects = computed(() =>
  mapOf(
    permissions.value,
    (p) => p.subject,
    () => true
  )
)

const validate = () => {
  for (const p of permissions.value) {
    if (p.subject === null) {
      return false
    }
  }
  return true
}

const addPermission = () => {
  permissions.value.push({
    subject: null,
    permission: { read: true, write: false },
    policy: 0,
  })
}
const removePermission = (i) => {
  permissions.value.splice(i, 1)
}

const loadPermissions = async () => {
  try {
    const data = await getPermissions(props.path)
    permissions.value = data.map((p) => ({
      subject: p.subject,
      permission: {
        read: (p.permission & PERMISSION_READ) === PERMISSION_READ,
        write: (p.permission & PERMISSION_WRITE) === PERMISSION_WRITE,
      },
      policy: p.policy,
    }))
    nextTick(() => {
      setSaveState(true)
    })
  } catch (e) {
    alert(e.message)
  }
}

const save = async () => {
  await savePermissions(
    props.path,
    permissions.value.map((p) => ({
      subject: p.subject,
      permission:
        (p.permission.read ? PERMISSION_READ : PERMISSION_EMPTY) |
        (p.permission.write ? PERMISSION_WRITE : PERMISSION_EMPTY),
      policy: p.policy,
    }))
  )
  setSaveState(true)
}

const loadSubjects = async () => {
  try {
    const res = await Promise.all([getUsers(), getGroups()])
    subjects.value = [
      { type: 'any', name: '*', subject: 'ANY' },
      ...res[0].map((u) => ({
        type: 'user',
        name: u.username,
        subject: `u:${u.username}`,
      })),
      ...res[1].map((g) => ({
        type: 'group',
        name: g.name,
        subject: `g:${g.name}`,
      })),
    ]
  } catch (e) {
    alert(e.message)
  }
}
const setSaveState = (saved) => {
  emit('save-state', saved)
}

watch(
  () => props.modelValue,
  (val) => {
    if (val === permissions.value) return
    permissions.value = [...(val || [])]
  },
  { immediate: true }
)

watch(
  () => permissions.value,
  () => {
    setSaveState(false)
    emit('update:modelValue', permissions.value)
  }
)

watchEffect(() => {
  loadPermissions()
})

loadSubjects()

defineExpose({ validate, save })
</script>
