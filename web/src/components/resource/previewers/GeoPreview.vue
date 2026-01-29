<template>
  <div class="geo-preview-container">
    <div class="leaflet-map" ref="mapElement"></div>
    <div class="geo-info-overlay" v-if="hasGeoInfo">
      <div class="info-item">
        <span class="label">坐标系:</span>
        <span class="value">{{ metaData?.crs || 'EPSG:4326' }}</span>
      </div>
    </div>
    <div v-if="!hasGeoInfo" class="geo-no-data">
      <el-icon :size="48"><MapLocation /></el-icon>
      <span>未检出地理空间元数据</span>
      <p class="hint">资源可能仅包含原始数据，建议下载后使用专业 GIS 软件查看</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { MapLocation } from '@element-plus/icons-vue'

const props = defineProps<{
  url: string
  metaData?: Record<string, any>
  interactive?: boolean // Added interactive prop
}>()

const mapElement = ref<HTMLElement | null>(null)
let map: L.Map | null = null

const hasGeoInfo = computed(() => {
  // Either HAS specific geometry info (center/bounds) OR is a map service link
  const hasSpecificInfo = props.metaData && (props.metaData.center || props.metaData.bounds || props.metaData.geometry)
  const isMapService = props.metaData && props.metaData.url && props.metaData.service_type
  return hasSpecificInfo || isMapService
})

const initMap = () => {
  if (!mapElement.value || !hasGeoInfo.value) return

  // 1. 初始化地图
  map = L.map(mapElement.value, {
    zoomControl: props.interactive !== false,
    dragging: props.interactive !== false,
    scrollWheelZoom: props.interactive !== false,
    doubleClickZoom: props.interactive !== false,
    boxZoom: props.interactive !== false,
    keyboard: props.interactive !== false,
    touchZoom: props.interactive !== false,
    attributionControl: false
  }).setView([0, 0], 2)

  // 2. 添加底图 (使用无需 Key 的灰度风格地图)
  L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
    maxZoom: 19,
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>'
  }).addTo(map)

  // 3. 处理地图服务 (WMS/XYZ/etc.)
  if (props.metaData && props.metaData.url) {
    const { url, service_type, layers, format, proxy_enabled } = props.metaData
    
    if (service_type === 'WMS') {
      L.tileLayer.wms(url, {
        layers: layers || '',
        format: format || 'image/png',
        transparent: true,
        version: '1.1.1'
      }).addTo(map)
    } else if (service_type === 'XYZ' || service_type === 'TMS') {
      // TMS usually needs tms: true option
      L.tileLayer(url, {
        tms: service_type === 'TMS'
      }).addTo(map)
    }
  }

  // 4. 处理元数据中的地理信息 (标注、边界等)
  if (props.metaData) {
    const { center, bounds, geometry } = props.metaData

    if (bounds) {
      // 预计格式: [[lat, lng], [lat, lng]]
      try {
        const b = L.latLngBounds(bounds)
        map.fitBounds(b)
        L.rectangle(b, { color: "#409EFF", weight: 1, fillOpacity: 0.1 }).addTo(map)
      } catch (e) { console.error("解析边界失败", e) }
    } else if (center) {
      // 预计格式: [lat, lng]
      const latlng = L.latLng(center[0], center[1])
      map.setView(latlng, 10)
      L.marker(latlng).addTo(map)
    }

    if (geometry) {
      // 这里的 geometry 支持 GeoJSON 格式解析
      try {
        L.geoJSON(geometry, {
          style: { color: "#409EFF", weight: 2 }
        }).addTo(map)
      } catch (e) { console.error("解析几何信息失败", e) }
    }
  }

  // 延迟一秒再次调整尺寸，确保在抽屉动画结束后地图布局正确
  setTimeout(() => {
    map?.invalidateSize()
  }, 400)
}

onMounted(() => {
  if (hasGeoInfo.value) {
    initMap()
  }
})

onUnmounted(() => {
  if (map) {
    map.remove()
    map = null
  }
})

// 监听元数据并行重绘
watch(() => props.metaData, () => {
  if (map) {
    map.remove()
    map = null
  }
  if (hasGeoInfo.value) {
    initMap()
  }
}, { deep: true })
</script>

<style scoped>
.geo-preview-container {
  width: 100%;
  height: 100%;
  background: #121212;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.leaflet-map {
  width: 100%;
  height: 100%;
  z-index: 1;
}

.geo-info-overlay {
  position: absolute;
  top: 12px;
  left: 12px;
  z-index: 100;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  padding: 6px 12px;
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.info-item {
  font-size: 11px;
  color: #ddd;
  display: flex;
  gap: 8px;
}

.info-item .label { color: #888; }

.geo-no-data {
  position: absolute;
  color: #555;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  text-align: center;
  padding: 0 40px;
}

.geo-no-data span {
  font-size: 14px;
  font-weight: 500;
}

.geo-no-data .hint {
  font-size: 12px;
  color: #444;
  margin: 0;
}

/* 修正 Leaflet 默认标记库图标在 Vite 下解析路径的问题 */
:deep(.leaflet-default-icon-path) {
  display: none;
}
</style>
