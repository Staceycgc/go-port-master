<template>
  <el-dialog
    :model-value="modelValue"
    @update:model-value="$emit('update:modelValue', $event)"
    :title="t('monitor.title')"
    class="pm-dialog pm-dialog-w600"
    width="94vw"
    @open="loadConfig"
  >
    <el-alert
      :title="t('monitor.hint')"
      type="info"
      show-icon
      :closable="false"
      style="margin-bottom: 16px"
    />

    <div class="monitor-status">
      <el-tag :type="wsStatus === 'open' ? 'success' : 'info'" size="small">
        {{ wsStatusLabel }}
      </el-tag>
    </div>

    <div class="monitor-form">
      <el-input v-model="newPort" :placeholder="t('monitor.portPlaceholder')" style="width: 120px" @keyup.enter="addPort" />
      <el-input v-model="newRemark" :placeholder="t('monitor.remarkPlaceholder')" style="width: 200px" />
      <el-button type="primary" @click="addPort">{{ t('monitor.add') }}</el-button>
      <el-switch v-model="monitorEnabled" :active-text="t('monitor.enable')" @change="toggleMonitor" />
    </div>

    <el-table :data="monitorList" size="small" border style="margin-top: 12px">
      <el-table-column prop="port" :label="t('monitor.port')" width="80" />
      <el-table-column prop="remark" :label="t('monitor.remark')" />
      <el-table-column prop="expectedState" :label="t('monitor.expectedState')" width="100">
        <template #default="{ row }">
          <el-select v-model="row.expectedState" size="small" @change="saveConfig">
            <el-option :label="t('monitor.any')" value="any" />
            <el-option :label="t('monitor.occupied')" value="occupied" />
            <el-option :label="t('monitor.free')" value="free" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column :label="t('monitor.currentState')" width="100">
        <template #default="{ row }">
          <el-tag :type="occupancyTagType(row.lastOccupied)" size="small">
            {{ formatOccupied(row.lastOccupied) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.action')" width="80">
        <template #default="{ $index }">
          <el-button link type="danger" size="small" @click="removePort($index)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-dialog>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import request from '@/api'
import { loadFromStorage, saveToStorage, STORAGE_KEYS } from '@/utils/storage'
import { syncMonitorConfig, connectMonitorWs, disconnectMonitorWs, getMonitorWsStatus } from '@/utils/monitorWs'

defineProps({ modelValue: Boolean })
const emit = defineEmits(['update:modelValue', 'alert', 'monitor-change'])

const { t } = useI18n()

const monitorList = ref([])
const monitorEnabled = ref(false)
const newPort = ref('')
const newRemark = ref('')
const wsStatus = ref('closed')

const wsStatusLabel = computed(() => {
  if (wsStatus.value === 'open') return t('monitor.wsConnected')
  if (wsStatus.value === 'connecting') return t('monitor.wsConnecting')
  return t('monitor.wsDisconnected')
})

function formatOccupied(value) {
  if (value === null || value === undefined) return t('monitor.unknown')
  return value ? t('monitor.occupied') : t('monitor.free')
}

function occupancyTagType(value) {
  if (value === null || value === undefined) return 'info'
  return value ? 'danger' : 'success'
}

function loadConfig() {
  const config = loadFromStorage(STORAGE_KEYS.MONITOR, { enabled: false, ports: [] })
  monitorList.value = config.ports || []
  monitorEnabled.value = config.enabled || false
  wsStatus.value = getMonitorWsStatus()
  if (monitorEnabled.value) {
    ensureMonitorWs()
    pushConfigToServer()
  }
  refreshMonitorStatus()
}

async function pushConfigToServer() {
  const config = { enabled: monitorEnabled.value, ports: monitorList.value }
  try {
    await syncMonitorConfig(request, config)
  } catch { /* ignore */ }
}

function saveConfig() {
  const config = { enabled: monitorEnabled.value, ports: monitorList.value }
  saveToStorage(STORAGE_KEYS.MONITOR, config)
  pushConfigToServer()
  emit('monitor-change', config)
}

function addPort() {
  const port = parseInt(newPort.value)
  if (!port || port < 1 || port > 65535) {
    ElMessage.warning(t('monitor.invalidPort'))
    return
  }
  if (monitorList.value.some(p => p.port === port)) {
    ElMessage.info(t('monitor.alreadyExists'))
    return
  }
  monitorList.value.push({
    port,
    protocol: 'TCP',
    remark: newRemark.value || t('monitor.defaultRemark', { port }),
    expectedState: 'any',
    lastOccupied: null
  })
  newPort.value = ''
  newRemark.value = ''
  saveConfig()
}

function removePort(index) {
  monitorList.value.splice(index, 1)
  saveConfig()
}

function toggleMonitor(enabled) {
  monitorEnabled.value = enabled
  saveConfig()
  if (enabled) {
    ensureMonitorWs()
  } else {
    disconnectMonitorWs()
    wsStatus.value = 'closed'
    emit('monitor-change', { enabled: false, ports: monitorList.value })
  }
}

function ensureMonitorWs() {
  connectMonitorWs((alerts) => {
    emit('alert', alerts)
  }, (status) => {
    wsStatus.value = status
  })
  wsStatus.value = getMonitorWsStatus()
}

async function refreshMonitorStatus() {
  if (monitorList.value.length === 0) return
  wsStatus.value = getMonitorWsStatus()
  try {
    const res = await request.post('/ports/monitor', {
      ports: monitorList.value.map(p => ({
        port: p.port,
        protocol: p.protocol,
        remark: p.remark,
        expectedState: p.expectedState
      }))
    })
    const results = res.data || []
    results.forEach(r => {
      const item = monitorList.value.find(p => p.port === r.port)
      if (item) item.lastOccupied = r.occupied
    })
    saveToStorage(STORAGE_KEYS.MONITOR, { enabled: monitorEnabled.value, ports: monitorList.value })
  } catch { /* ignore */ }
}
</script>

<style scoped>
.monitor-form {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.monitor-status {
  margin-bottom: 8px;
}
</style>
