<template>
  <div>
    <div class="flex items-center gap-3 mb-4">
      <Button icon="pi pi-arrow-left" text @click="router.back()" />
      <h1 class="text-2xl font-bold text-gray-800">Graphe de relations</h1>
      <div class="flex-1"></div>
      <div class="flex items-center gap-2">
        <label class="text-sm text-gray-600">Profondeur :</label>
        <InputNumber
          v-model="maxDepth"
          :min="1"
          :max="10"
          showButtons
          :step="1"
          class="w-24"
          @update:modelValue="loadGraph"
        />
      </div>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <ProgressSpinner />
    </div>

    <div v-else class="bg-white rounded-lg shadow" style="height: 600px">
      <div ref="cyContainer" class="w-full h-full"></div>
    </div>

    <!-- Node info panel -->
    <div v-if="selectedNode" class="mt-4 bg-white rounded-lg shadow p-4">
      <div class="flex items-center gap-2 mb-2">
        <Tag :value="selectedNode.type" severity="info" />
        <span class="font-semibold text-gray-800">{{
          selectedNode.label
        }}</span>
        <Button
          icon="pi pi-external-link"
          text
          size="small"
          @click="router.push(`/items/${selectedNode.id}`)"
          v-tooltip="'Voir la fiche'"
        />
      </div>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-2 text-sm">
        <div v-for="(value, key) in selectedNode.attributes" :key="key">
          <span class="text-gray-400">{{ key }}:</span> {{ value }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from "vue";
import { useRouter } from "vue-router";
import Button from "primevue/button";
import InputNumber from "primevue/inputnumber";
import Tag from "primevue/tag";
import ProgressSpinner from "primevue/progressspinner";
import cytoscape from "cytoscape";
import type { Core } from "cytoscape";
import { itemApi } from "@/api/client";
import type { GraphNode, GraphResponse } from "@/api/types";

const props = defineProps<{ id: string }>();
const router = useRouter();

const cyContainer = ref<HTMLElement | null>(null);
const maxDepth = ref(3);
const loading = ref(true);
const selectedNode = ref<GraphNode | null>(null);

let cy: Core | null = null;

// Color palette for node types
const typeColors: Record<string, string> = {};
const palette = [
  "#4f46e5",
  "#059669",
  "#d97706",
  "#dc2626",
  "#7c3aed",
  "#0891b2",
  "#be185d",
  "#65a30d",
];
let colorIdx = 0;

function getColor(type: string): string {
  if (!typeColors[type]) {
    typeColors[type] = palette[colorIdx % palette.length]!;
    colorIdx++;
  }
  return typeColors[type]!;
}

async function loadGraph() {
  loading.value = true;
  selectedNode.value = null;

  try {
    const graph = await itemApi.graph(props.id, { max_depth: maxDepth.value });

    // Stop loading so the container enters the DOM
    loading.value = false;
    await nextTick();
    if (!cyContainer.value) return;

    renderGraph(graph);
  } catch {
    loading.value = false;
  }
}

function renderGraph(graph: GraphResponse) {
  if (!cyContainer.value) return;

  const allNodes = [graph.root, ...(graph.nodes ?? [])];
  const elements: cytoscape.ElementDefinition[] = [
    // Nodes
    ...allNodes.map((n) => ({
      data: {
        id: n.id,
        label: n.label || n.id.slice(0, 8),
        type: n.type,
        color: getColor(n.type),
        isRoot: n.id === graph.root.id,
        nodeData: n,
      },
    })),
    // Edges
    ...(graph.edges ?? []).map((e, i) => ({
      data: {
        id: `e-${i}`,
        source: e.source_id,
        target: e.target_id,
        label: e.relation_type,
      },
    })),
  ];

  if (cy) cy.destroy();

  cy = cytoscape({
    container: cyContainer.value,
    elements,
    style: [
      {
        selector: "node",
        style: {
          "background-color": "data(color)",
          label: "data(label)",
          "text-valign": "bottom",
          "text-halign": "center",
          "font-size": "11px",
          color: "#374151",
          width: 30,
          height: 30,
        },
      },
      {
        selector: "node[?isRoot]",
        style: {
          width: 45,
          height: 45,
          "border-width": 3,
          "border-color": "#1e1b4b",
          "font-weight": "bold",
        },
      },
      {
        selector: "edge",
        style: {
          width: 2,
          "line-color": "#d1d5db",
          "target-arrow-color": "#9ca3af",
          "target-arrow-shape": "triangle",
          "curve-style": "bezier",
          label: "data(label)",
          "font-size": "9px",
          color: "#9ca3af",
          "text-rotation": "autorotate",
          "text-margin-y": -8,
        },
      },
    ],
    layout: { name: "cose", animate: true, animationDuration: 500 },
  });

  cy.on("tap", "node", (evt) => {
    selectedNode.value = evt.target.data("nodeData") as GraphNode;
  });

  cy.on("tap", (evt) => {
    if (evt.target === cy) selectedNode.value = null;
  });
}

onMounted(() => loadGraph());
onUnmounted(() => cy?.destroy());
</script>
