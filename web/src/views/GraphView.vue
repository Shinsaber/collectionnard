<template>
  <div>
    <div class="flex items-center gap-3 mb-4">
      <Button icon="pi pi-arrow-left" text @click="router.back()" />
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        Graphe de relations
      </h1>
      <div class="flex-1"></div>
      <div class="flex items-center gap-2">
        <label class="text-sm text-gray-600 dark:text-gray-400"
          >Profondeur :</label
        >
        <InputNumber
          v-model="maxDepth"
          :min="1"
          :max="10"
          show-buttons
          :step="1"
          class="w-24"
          @update:model-value="loadGraph"
        />
      </div>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <ProgressSpinner />
    </div>

    <div
      v-else
      class="bg-white dark:bg-gray-800 rounded-lg shadow"
      style="height: 600px"
    >
      <div ref="cyContainer" class="w-full h-full"></div>
    </div>

    <!-- Legend -->
    <div
      v-if="!loading && Object.keys(typeColors).length"
      class="mt-2 flex flex-wrap gap-3 px-2"
    >
      <span
        v-for="(color, typeName) in typeColors"
        :key="typeName"
        class="flex items-center gap-1.5 text-xs text-gray-600 dark:text-gray-400"
      >
        <span
          class="inline-block w-3 h-3 rounded-sm"
          :style="{ backgroundColor: color }"
        ></span>
        {{ typeName }}
      </span>
    </div>

    <!-- Node detail modal -->
    <Dialog
      v-model:visible="nodeDialogVisible"
      :header="selectedNodeTitle"
      modal
      :style="{ width: '550px' }"
    >
      <div v-if="selectedNode" class="flex flex-col gap-4">
        <!-- Header: type + title -->
        <div class="flex items-center gap-2">
          <Tag :value="selectedNode.type" severity="info" />
          <span
            v-if="selectedNode.depth === 0"
            class="text-xs text-gray-400 dark:text-gray-500"
            >(racine)</span
          >
        </div>

        <!-- Poster image -->
        <div v-if="selectedNodeImage" class="flex justify-center">
          <img
            :src="selectedNodeImage"
            :alt="selectedNodeTitle"
            class="max-h-48 rounded-lg shadow object-contain"
          />
        </div>

        <!-- Attributes -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-2 text-sm">
          <div
            v-for="(value, key) in filteredNodeAttributes"
            :key="key"
            class="flex flex-col"
          >
            <span class="text-gray-400 dark:text-gray-500 text-xs">{{
              key
            }}</span>
            <span class="dark:text-gray-200">{{ formatAttrValue(value) }}</span>
          </div>
        </div>
      </div>

      <template #footer>
        <Button label="Fermer" text @click="nodeDialogVisible = false" />
        <Button
          label="Voir la fiche"
          icon="pi pi-external-link"
          @click="
            router.push(`/items/${selectedNode!.id}`);
            nodeDialogVisible = false;
          "
        />
      </template>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from "vue";
import { useRouter } from "vue-router";
import { useTheme } from "@/composables/useTheme";
import Button from "primevue/button";
import InputNumber from "primevue/inputnumber";
import Tag from "primevue/tag";
import Dialog from "primevue/dialog";
import ProgressSpinner from "primevue/progressspinner";
import cytoscape from "cytoscape";
import type { Core } from "cytoscape";
import { itemApi } from "@/api/client";
import type { GraphNode, GraphEdge, GraphResponse } from "@/api/types";

const props = defineProps<{ id: string }>();
const router = useRouter();
const { theme } = useTheme();

const cyContainer = ref<HTMLElement | null>(null);
const maxDepth = ref(3);
const loading = ref(true);

// Node detail modal
const selectedNode = ref<GraphNode | null>(null);
const nodeDialogVisible = ref(false);

let cy: Core | null = null;

// ── Color palette ──
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

// ── Localized label extraction ──
function extractTitle(attrs: Record<string, unknown>): string {
  for (const key of ["titre", "nom", "name", "title", "label"]) {
    const val = attrs[key];
    if (val == null) continue;
    if (typeof val === "object" && !Array.isArray(val)) {
      const obj = val as Record<string, string>;
      if (typeof obj["fr"] === "string") return obj["fr"];
      if (typeof obj["en"] === "string") return obj["en"];
    }
    if (typeof val === "string") return val;
  }
  return "";
}

function nodeTitle(n: GraphNode): string {
  return extractTitle(n.attributes) || n.id.slice(0, 8);
}

// ── SVG badge for type label inside node ──
function makeTypeSvg(type: string, color: string, isRoot: boolean): string {
  const charWidth = 7.5;
  const padding = 24;
  const w = Math.max(type.length * charWidth + padding, 50);
  const h = 30;
  const rx = 8;
  const borderWidth = isRoot ? 3 : 0;
  const borderColor = "#1e1b4b";

  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}">
    <rect rx="${rx}" ry="${rx}" width="${w}" height="${h}" fill="${color}"
      ${borderWidth ? `stroke="${borderColor}" stroke-width="${borderWidth}"` : ""} />
    <text x="${w / 2}" y="${h / 2}" text-anchor="middle" dominant-baseline="central"
      fill="white" font-family="system-ui, sans-serif" font-size="12"
      font-weight="bold">${escapeXml(type)}</text>
  </svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

function escapeXml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

// ── Edge label formatting ──
function edgeLabel(e: GraphEdge): string {
  const attrs = e.attributes ?? {};
  const tags: string[] = [];
  if (attrs.order != null) tags.push(`#${attrs.order}`);
  if (attrs.language) tags.push(String(attrs.language));
  if (attrs.role) tags.push(String(attrs.role));
  if (tags.length) return `${e.relation_type}\n${tags.join(" · ")}`;
  return e.relation_type;
}

// ── Node detail modal helpers ──
const selectedNodeTitle = computed(() =>
  selectedNode.value ? nodeTitle(selectedNode.value) : "",
);

const selectedNodeImage = computed(() => {
  if (!selectedNode.value) return "";
  const attrs = selectedNode.value.attributes;
  for (const key of Object.keys(attrs)) {
    if (!/image/i.test(key)) continue;
    const val = attrs[key];
    if (typeof val === "object" && val && !Array.isArray(val)) {
      const obj = val as Record<string, unknown>;
      if (typeof obj.default_image === "string" && obj.default_image)
        return obj.default_image;
      for (const v of Object.values(obj)) {
        if (Array.isArray(v) && typeof v[0] === "string") return v[0];
      }
    }
  }
  return "";
});

const filteredNodeAttributes = computed(() => {
  if (!selectedNode.value) return {};
  const skip = new Set(["images", "Images"]);
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(selectedNode.value.attributes)) {
    if (skip.has(k)) continue;
    out[k] = v;
  }
  return out;
});

function formatAttrValue(value: unknown): string {
  if (value == null) return "—";
  if (typeof value === "object" && !Array.isArray(value)) {
    const obj = value as Record<string, unknown>;
    if (typeof obj["fr"] === "string") return obj["fr"];
    if (typeof obj["en"] === "string") return obj["en"];
    return JSON.stringify(value);
  }
  if (Array.isArray(value)) return value.join(", ");
  return String(value);
}

// ── Graph loading ──
async function loadGraph() {
  loading.value = true;
  selectedNode.value = null;

  try {
    const graph = await itemApi.graph(props.id, { max_depth: maxDepth.value });

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

  const isDark = theme.value === "dark";
  const textColor = isDark ? "#d1d5db" : "#374151";
  const edgeLineColor = isDark ? "#4b5563" : "#d1d5db";
  const edgeArrowColor = isDark ? "#9ca3af" : "#6b7280";
  const edgeLabelColor = isDark ? "#9ca3af" : "#6b7280";

  const allNodes = [graph.root, ...(graph.nodes ?? [])];
  const elements: cytoscape.ElementDefinition[] = [
    // Nodes
    ...allNodes.map((n) => {
      const color = getColor(n.type);
      const isRoot = n.id === graph.root.id;
      const title = nodeTitle(n);
      const w = Math.max(n.type.length * 7.5 + 24, 50);
      return {
        data: {
          id: n.id,
          title,
          type: n.type,
          color,
          isRoot,
          nodeData: n,
          typeSvg: makeTypeSvg(n.type, color, isRoot),
          nodeWidth: w,
        },
      };
    }),
    // Edges
    ...(graph.edges ?? []).map((e, i) => ({
      data: {
        id: `e-${i}`,
        source: e.source_id,
        target: e.target_id,
        label: edgeLabel(e),
        edgeData: e,
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
          shape: "roundrectangle",
          "background-image": "data(typeSvg)",
          "background-fit": "cover",
          "background-opacity": 0,
          "border-width": 0,
          width: "data(nodeWidth)",
          height: 30,
          // Title below the type badge
          label: "data(title)",
          "text-valign": "bottom",
          "text-halign": "center",
          "text-margin-y": 6,
          "font-size": "10px",
          color: textColor,
          "text-wrap": "wrap",
          "text-max-width": "140px",
        },
      },
      {
        selector: "node[?isRoot]",
        style: {
          "font-weight": "bold",
          "font-size": "11px",
        },
      },
      {
        selector: "edge",
        style: {
          width: 2,
          "line-color": edgeLineColor,
          "target-arrow-color": edgeArrowColor,
          "target-arrow-shape": "triangle",
          "arrow-scale": 1.2,
          "curve-style": "bezier",
          label: "data(label)",
          "font-size": "9px",
          color: edgeLabelColor,
          "text-wrap": "wrap",
          "text-max-width": "120px",
          "text-rotation": "autorotate",
          "text-margin-y": -10,
          "text-background-color": isDark ? "#1f2937" : "#ffffff",
          "text-background-opacity": 0.85,
          "text-background-padding": "3px",
          "text-background-shape": "roundrectangle",
        },
      },
    ],
    layout: {
      name: "cose",
      animate: true,
      animationDuration: 500,
      idealEdgeLength: () => 180,
      nodeRepulsion: () => 8000,
      edgeElasticity: () => 100,
      gravity: 0.25,
      padding: 40,
      nodeDimensionsIncludeLabels: true,
    } as cytoscape.CoseLayoutOptions,
  });

  // Node click → detail modal
  cy.on("tap", "node", (evt) => {
    selectedNode.value = evt.target.data("nodeData") as GraphNode;
    nodeDialogVisible.value = true;
  });

  // Click on empty space → close modal
  cy.on("tap", (evt) => {
    if (evt.target === cy) {
      nodeDialogVisible.value = false;
      selectedNode.value = null;
    }
  });
}

onMounted(() => loadGraph());
onUnmounted(() => cy?.destroy());
</script>
