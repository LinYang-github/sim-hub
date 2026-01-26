<template>
  <div class="glb-preview-container" ref="container">
    <div v-if="loading" class="preview-loading">
      <el-icon class="is-loading"><Loading /></el-icon>
    </div>
    <div v-if="error" class="preview-error">
      <el-icon><Warning /></el-icon>
      <span>加载失败</span>
    </div>
    <canvas ref="canvas" v-show="!loading && !error" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import * as THREE from 'three'
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import { Loading, Warning } from '@element-plus/icons-vue'

const props = defineProps<{
  url: string,
  force?: boolean
}>()

const container = ref<HTMLElement | null>(null)
const canvas = ref<HTMLCanvasElement | null>(null)
const loading = ref(true)
const error = ref(false)

let scene: THREE.Scene
let camera: THREE.PerspectiveCamera
let renderer: THREE.WebGLRenderer
let controls: OrbitControls
let animationId: number
let model: THREE.Group | null = null
let observer: IntersectionObserver | null = null
let isVisible = false

const initScene = () => {
  if (!canvas.value || !container.value) return

  // 即使初始尺寸为 0 也继续初始化，ResizeObserver 会处理后续尺寸更新
  const width = container.value.clientWidth || 300
  const height = container.value.clientHeight || 200

  scene = new THREE.Scene()
  scene.background = null

  camera = new THREE.PerspectiveCamera(45, width / height, 0.1, 1000)
  camera.position.set(2, 2, 2)

  renderer = new THREE.WebGLRenderer({
    canvas: canvas.value,
    antialias: true,
    alpha: true,
    powerPreference: 'low-power'
  })
  renderer.setSize(width, height)
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))

  // Lights
  const ambientLight = new THREE.AmbientLight(0xffffff, 0.8)
  scene.add(ambientLight)

  const directionalLight = new THREE.DirectionalLight(0xffffff, 1)
  directionalLight.position.set(5, 5, 5)
  scene.add(directionalLight)

  // Controls
  controls = new OrbitControls(camera, canvas.value)
  controls.enableDamping = true
  controls.dampingFactor = 0.05
  controls.enablePan = false
  controls.enableZoom = false
  controls.autoRotate = true
  controls.autoRotateSpeed = 2

  animate()
  loadModel()
  setupObservers()
}

const setupObservers = () => {
  if (!container.value) return
  
  // 1. 可见性监听
  observer = new IntersectionObserver((entries) => {
    isVisible = entries[0].isIntersecting
  }, { threshold: 0.1 })
  observer.observe(container.value)

  // 2. 尺寸监听 (关键：处理侧滑抽屉展开过程中的尺寸变化)
  const resizeObserver = new ResizeObserver(() => {
    handleResize()
  })
  resizeObserver.observe(container.value)
}

const animate = () => {
  animationId = requestAnimationFrame(animate)
  
  // 如果开启了 force，则跳过可见性检查
  if (!isVisible && !props.force) return

  let needsRender = false
  
  if (controls) {
    controls.update()
    needsRender = true
  }

  if (needsRender && renderer && scene && camera) {
    renderer.render(scene, camera)
  }
}

const loadModel = () => {
  if (!props.url) return

  loading.value = true
  error.value = false

  if (model) {
    scene.remove(model)
    // 基础材质清理
    model.traverse((child: any) => {
      if (child.isMesh) {
        child.geometry.dispose()
        if (child.material.isMaterial) {
          cleanMaterial(child.material)
        }
      }
    })
    model = null
  }

  const loader = new GLTFLoader()
  loader.load(
    props.url,
    (gltf) => {
      model = gltf.scene
      
      const box = new THREE.Box3().setFromObject(model)
      const size = box.getSize(new THREE.Vector3())
      const center = box.getCenter(new THREE.Vector3())
      
      const maxDim = Math.max(size.x, size.y, size.z)
      const scale = 1.5 / maxDim
      model.scale.set(scale, scale, scale)
      model.position.sub(center.multiplyScalar(scale))
      
      scene.add(model)
      loading.value = false
    },
    undefined,
    (err) => {
      console.error('Error loading GLB:', err)
      loading.value = false
      error.value = true
    }
  )
}

const cleanMaterial = (material: any) => {
  material.dispose()
  for (const key of Object.keys(material)) {
    const value = material[key]
    if (value && typeof value.dispose === 'function') {
      value.dispose()
    }
  }
}

const handleResize = () => {
  if (!container.value || !renderer || !camera) return
  const width = container.value.clientWidth
  const height = container.value.clientHeight
  camera.aspect = width / height
  camera.updateProjectionMatrix()
  renderer.setSize(width, height)
}

watch(() => props.url, () => {
  loadModel()
})

onMounted(() => {
  initScene()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  cancelAnimationFrame(animationId)
  window.removeEventListener('resize', handleResize)
  if (observer) observer.disconnect()
  if (renderer) {
    renderer.dispose()
  }
  if (scene) {
    scene.clear()
  }
})
</script>

<style scoped>
.glb-preview-container {
  width: 100%;
  height: 100%;
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.preview-loading, .preview-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-placeholder);
}

.preview-loading .el-icon {
  font-size: 24px;
}

.preview-error {
  color: var(--el-color-danger);
}

canvas {
  width: 100% !important;
  height: 100% !important;
  display: block;
}
</style>
