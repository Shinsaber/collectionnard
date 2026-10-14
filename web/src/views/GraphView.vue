<template>
  <div>
    <div class="flex items-center gap-3 mb-4">
      <ElButton text :aria-label="t('common.back')" @click="router.back()">
        <i class="pi pi-arrow-left" aria-hidden="true"></i>
      </ElButton>
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        {{ t("graph.view.title") }}
      </h1>
      <div class="flex-1"></div>
      <div class="flex items-center gap-2">
        <label
          for="graph-depth"
          class="text-sm text-gray-600 dark:text-gray-400"
          >{{ t("graph.view.depth") }}</label
        >
        <ElInputNumber
          id="graph-depth"
          v-model="maxDepth"
          :min="1"
          :max="auth.isAdmin ? 10 : 5"
          :step="1"
          size="small"
          class="w-28!"
          @change="loadGraph"
        />
      </div>
    </div>

    <ElAlert
      v-if="truncated && !loading"
      type="warning"
      :closable="false"
      show-icon
      class="mb-3"
      :title="t('graph.view.truncated')"
    />

    <AppSpinner v-if="loading" />

    <div
      v-else
      class="bg-white dark:bg-gray-800 rounded-lg shadow-sm"
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
          class="inline-block w-3 h-3 rounded-xs"
          :style="{ backgroundColor: color }"
        ></span>
        {{ typeDisplay(String(typeName)) }}
      </span>
    </div>

    <GraphNodeDialog
      v-model:visible="nodeDialogVisible"
      :node="selectedNode"
      :schema="selectedNode ? itemTypeSchemas[selectedNode.type] : undefined"
      :title="selectedNodeTitle"
      :type-label="selectedNode ? typeDisplay(selectedNode.type) : ''"
      @open="openItem"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useTheme } from "@/composables/useTheme";
import AppSpinner from "@/components/common/AppSpinner.vue";
import cytoscape from "cytoscape";
import type { Core } from "cytoscape";
import { itemApi } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useMetaTypes } from "@/stores/metaTypes";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import {
  displayValue,
  fieldLabel,
  fieldValue,
  orderedKeys,
  renderLabel,
  typeTitle,
} from "@/lib/schema";
import GraphNodeDialog from "@/components/graph/GraphNodeDialog.vue";
import type { GraphNode, GraphEdge, GraphResponse } from "@/api/types";

const props = defineProps<{ id: string }>();
const router = useRouter();
const { theme } = useTheme();

const cyContainer = ref<HTMLElement | null>(null);
const maxDepth = ref(3);
const truncated = ref(false);
const auth = useAuthStore();
const { lang } = useLang();
const { t } = useI18n();
const { notifyError } = useNotify();
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

// ── Schémas des méta-types (par nom), depuis le cache partagé ──
const metaTypes = useMetaTypes();
const itemTypeSchemas = computed(() => metaTypes.itemTypeSchemasByName);
const relationTypeSchemas = computed(() => metaTypes.relationTypeSchemasByName);

// Méta-types par nom (nœuds et arêtes ne portent que le nom) : nom affiché via x-title.
const itemTypesByName = computed(
  () => new Map(metaTypes.itemTypes.map((it) => [it.name, it])),
);
const relationTypesByName = computed(
  () => new Map(metaTypes.relationTypes.map((rt) => [rt.name, rt])),
);
function typeDisplay(name: string): string {
  return typeTitle(itemTypesByName.value.get(name), lang.value) || name;
}
function relationDisplay(name: string): string {
  return typeTitle(relationTypesByName.value.get(name), lang.value) || name;
}

async function loadSchemas() {
  await Promise.all([metaTypes.loadItemTypes(), metaTypes.loadRelationTypes()]);
}

// Le serveur calcule `label` en français : on le recalcule dans la langue courante
// avec le x-label de l'ItemType, repli sur celui du serveur.
function nodeTitle(n: GraphNode): string {
  return (
    renderLabel(itemTypeSchemas.value[n.type], n.attributes, lang.value) ||
    n.label ||
    n.id.slice(0, 8)
  );
}

// Données d'affichage d'un nœud dépendant de la langue (titre, badge du type).
function nodeVisual(n: GraphNode, isRoot: boolean) {
  const color = getColor(n.type);
  const typeLabel = typeDisplay(n.type);
  return {
    title: nodeTitle(n),
    typeSvg: makeTypeSvg(typeLabel, color, isRoot),
    nodeWidth: Math.max(typeLabel.length * 7.5 + 24, 50),
  };
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
// Nom du type + attributs libellés via les x-labels du RelationType ; `order` en badge #n.
function edgeLabel(e: GraphEdge): string {
  const schema = relationTypeSchemas.value[e.relation_type];
  const attrs = e.attributes ?? {};
  const parts = orderedKeys(schema, attrs)
    .filter((k) => attrs[k] != null && attrs[k] !== "")
    .map((k) =>
      k === "order"
        ? `#${displayValue(attrs[k], lang.value)}`
        : `${fieldLabel(schema, k, lang.value)} ${fieldValue(schema, k, attrs[k], lang.value)}`,
    );
  const name = relationDisplay(e.relation_type);
  if (parts.length) return `${name}\n${parts.join(" · ")}`;
  return name;
}

// ── Node detail modal helpers ──
const selectedNodeTitle = computed(() =>
  selectedNode.value ? nodeTitle(selectedNode.value) : "",
);

function openItem(id: string) {
  nodeDialogVisible.value = false;
  router.push(`/items/${id}`);
}

// ── Graph loading ──
async function loadGraph() {
  loading.value = true;
  selectedNode.value = null;

  try {
    const graph = await itemApi.graph(props.id, { max_depth: maxDepth.value });
    truncated.value = graph.truncated;

    loading.value = false;
    await nextTick();
    if (!cyContainer.value) return;

    renderGraph(graph);
  } catch (e) {
    loading.value = false;
    notifyError(e, t("graph.view.loadFailed"));
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
      const isRoot = n.id === graph.root.id;
      return {
        data: {
          id: n.id,
          type: n.type,
          color: getColor(n.type),
          isRoot,
          nodeData: n,
          ...nodeVisual(n, isRoot),
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

// Changement de langue : libellés des nœuds et arêtes recalculés sans recharger le graphe.
watch(lang, () => {
  if (!cy) return;
  cy.batch(() => {
    cy!.nodes().forEach((node) => {
      node.data(
        nodeVisual(node.data("nodeData") as GraphNode, !!node.data("isRoot")),
      );
    });
    cy!.edges().forEach((edge) => {
      edge.data("label", edgeLabel(edge.data("edgeData") as GraphEdge));
    });
  });
});

onMounted(async () => {
  await loadSchemas().catch((e) =>
    notifyError(e, t("graph.view.schemasFailed")),
  );
  await loadGraph();
});
onUnmounted(() => cy?.destroy());
</script>
