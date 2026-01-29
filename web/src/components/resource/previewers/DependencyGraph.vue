<template>
  <div class="dependency-graph-container" ref="container">
    <VNetworkGraph
      v-if="Object.keys(nodes).length > 0"
      ref="graph"
      :nodes="nodes"
      :edges="edges"
      :configs="configs"
    />
    <el-empty v-else description="暂无依赖数据" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, nextTick } from 'vue'
import { VNetworkGraph, defineConfigs } from "v-network-graph"
import type { ResourceDependency } from '../../../../core/types/resource'

const props = defineProps<{
  dependencies: ResourceDependency[]
  rootName: string
}>()

interface Node {
  name: string
  label: string
  color: string
  isRoot: boolean
  size: number
}

interface Edge {
  source: string
  target: string
}

const nodes = ref<Record<string, Node>>({})
const edges = ref<Record<string, Edge>>({})
const graph = ref<any>(null)

// 深度定制可视化配置
const configs = defineConfigs({
  view: {
    autoPanAndZoomOnLoad: "fit-content",
    // 使用内置布局处理器，确保节点不会重叠
    layoutHandler: undefined, 
  },
  node: {
    normal: {
      type: "circle",
      radius: (node: any) => node.size,
      color: (node: any) => node.color,
      strokeWidth: 2,
      strokeColor: "#1d1e1f",
    },
    hover: {
      radius: (node: any) => node.size + 2,
      color: "#409EFF",
    },
    label: {
      visible: true,
      fontFamily: "Inter, sans-serif",
      fontSize: 12,
      color: "#E5EAF3", // 亮白色，适配暗色背景
      margin: 8,
      direction: "south",
    },
    focusring: {
      color: "#409EFF",
    },
  },
  edge: {
    normal: {
      width: 2,
      color: "#4C4D4F",
      dasharray: "0",
      linecap: "round",
    },
    hover: {
      width: 3,
      color: "#409EFF",
    },
    marker: {
      target: {
        type: "arrow",
        width: 4,
        height: 4,
      },
    },
  },
})

const buildGraph = () => {
  const newNodes: Record<string, Node> = {}
  const newEdges: Record<string, Edge> = {}
  let edgeCounter = 0

  // 1. 注册主节点
  newNodes['root'] = {
    name: props.rootName,
    label: `${props.rootName}\n(当前资源)`,
    color: "#409EFF", // 科技蓝
    isRoot: true,
    size: 24
  }

  // 2. 递归构建依赖
  const traverse = (deps: ResourceDependency[], parentId: string, depth: number) => {
    deps.forEach(dep => {
      // 这里的 ID 必须唯一，否则连线会错乱
      const nodeId = dep.id || dep.resource_id || `dep-${edgeCounter++}`
      
      if (!newNodes[nodeId]) {
        newNodes[nodeId] = {
          name: dep.resource_name,
          label: `${dep.resource_name}\n${dep.semver || 'latest'}`,
          color: "#67C23A", // 成功绿
          isRoot: false,
          size: 18
        }
      }

      const edgeId = `edge-${edgeCounter++}`
      newEdges[edgeId] = { source: parentId, target: nodeId }

      if (dep.dependencies && dep.dependencies.length > 0) {
        traverse(dep.dependencies, nodeId, depth + 1)
      }
    })
  }

  if (props.dependencies && props.dependencies.length > 0) {
    traverse(props.dependencies, 'root', 1)
  }

  nodes.value = newNodes
  edges.value = newEdges

  // 自适应视野
  nextTick(() => {
    setTimeout(() => {
      if (graph.value) graph.value.fitToContents()
    }, 150)
  })
}

const container = ref<HTMLElement | null>(null)

onMounted(() => {
  buildGraph()
})

watch(() => props.dependencies, buildGraph, { deep: true })
watch(() => props.rootName, buildGraph)
</script>

<style scoped>
.dependency-graph-container {
  width: 100%;
  height: 420px;
  background-color: #1a1a1a;
  border-radius: 12px;
  border: 1px solid #333;
  overflow: hidden;
  position: relative;
}

/* 简单的进场动画 */
.dependency-graph-container :deep(svg) {
  animation: fadeIn 0.6s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: scale(0.98); }
  to { opacity: 1; transform: scale(1); }
}
</style>
