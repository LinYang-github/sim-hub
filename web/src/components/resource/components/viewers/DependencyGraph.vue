<template>
  <div class="dependency-graph-container" ref="container">
    <v-network-graph
      v-if="Object.keys(nodes).length > 0"
      ref="graph"
      :nodes="nodes"
      :edges="edges"
      :configs="configs"
    >
      <!-- Custom node to show icons or better labels -->
      <template #override-node="{ nodeId, scale, config, outputs, x, y }">
        <circle
          :r="config.radius * scale"
          :fill="nodes[nodeId].isRoot ? '#409EFF' : '#67C23A'"
          :stroke="config.strokeColor"
          :stroke-width="config.strokeWidth * scale"
          @mouseenter="outputs.onMouseenter"
          @mouseleave="outputs.onMouseout"
        />
        <text
          class="node-label"
          :x="x"
          :y="y + config.radius * scale + 15"
          text-anchor="middle"
          :font-size="12 * scale"
          fill="var(--el-text-color-primary)"
        >
          {{ nodes[nodeId].name }}
        </text>
        <text
          class="node-version"
          :x="x"
          :y="y + config.radius * scale + 30"
          text-anchor="middle"
          :font-size="10 * scale"
          fill="var(--el-text-color-secondary)"
        >
          {{ nodes[nodeId].version }}
        </text>
      </template>
    </v-network-graph>
    <el-empty v-else description="暂无依赖数据" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { defineConfigs } from "v-network-graph"
import "v-network-graph/lib/style.css"
import type { ResourceDependency } from '../../../../core/types/resource'

const props = defineProps<{
  dependencies: ResourceDependency[]
  rootName: string
}>()

interface Node {
  name: string
  version: string
  isRoot?: boolean
}

interface Edge {
  source: string
  target: string
}

const nodes = ref<Record<string, Node>>({})
const edges = ref<Record<string, Edge>>({})
const graph = ref<any>(null)

const configs = defineConfigs({
  node: {
    radius: 18,
    strokeWidth: 2,
    strokeColor: "#ffffff",
    hover: {
      radius: 20,
      color: "#409EFF",
    },
    label: {
      visible: false,
    },
    focusring: {
      color: "#409EFF",
    },
  },
  edge: {
    width: 2,
    color: "#DCDFE6",
    margin: 4,
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
  view: {
    autoPanAndZoomOnLoad: "fit-content",
  },
})

const buildGraph = () => {
  const newNodes: Record<string, Node> = {}
  const newEdges: Record<string, Edge> = {}
  let edgeCounter = 0

  newNodes['root'] = {
    name: props.rootName,
    version: '当前资源',
    isRoot: true
  }

  const traverse = (deps: ResourceDependency[], parentId: string) => {
    deps.forEach(dep => {
      const nodeId = dep.resource_id
      if (!newNodes[nodeId]) {
        newNodes[nodeId] = {
          name: dep.resource_name,
          version: dep.semver || 'latest'
        }
      }

      const edgeId = `edge-${edgeCounter++}`
      newEdges[edgeId] = {
        source: parentId,
        target: nodeId
      }

      if (dep.dependencies && dep.dependencies.length > 0) {
        traverse(dep.dependencies, nodeId)
      }
    })
  }

  traverse(props.dependencies, 'root')

  nodes.value = newNodes
  edges.value = newEdges
}

const container = ref<HTMLElement | null>(null)
let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  buildGraph()
  if (container.value) {
    resizeObserver = new ResizeObserver(() => {
      graph.value?.fitToContents()
    })
    resizeObserver.observe(container.value)
  }
})

watch(() => props.dependencies, buildGraph)
watch(() => props.rootName, buildGraph)
</script>

<style scoped>
.dependency-graph-container {
  width: 100%;
  height: 400px;
  background-color: var(--el-fill-color-extra-light);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}

.node-label {
  font-weight: 600;
  pointer-events: none;
}

.node-version {
  font-family: monospace;
  pointer-events: none;
}
</style>
