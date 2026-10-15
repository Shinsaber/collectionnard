<template>
  <div>
    <RouterLink
      :to="`/items/${id}`"
      class="mb-4 inline-flex items-center gap-1.5 text-sm font-semibold text-link"
    >
      <i class="pi pi-arrow-left text-xs" aria-hidden="true"></i>
      {{ t("common.back") }}
    </RouterLink>

    <PageHeader :title="t('graph.view.title')">
      <template #actions>
        <div class="flex items-center gap-2.5">
          <label for="graph-depth" class="text-sm text-muted">{{
            t("graph.view.depth")
          }}</label>
          <ElInputNumber
            id="graph-depth"
            v-model="maxDepth"
            :min="1"
            :max="auth.isAdmin ? 10 : 5"
            :step="1"
            class="w-32!"
            @change="loadGraph"
          />
        </div>
      </template>
    </PageHeader>

    <ElAlert
      v-if="truncated && !loading"
      type="warning"
      :closable="false"
      show-icon
      class="mb-3"
      :title="t('graph.view.truncated')"
    />

    <AppSpinner v-if="loading" />

    <!-- Hauteur : l'écran moins l'en-tête (et la barre d'onglets sous `lg`). -->
    <div
      v-else
      class="h-[calc(100dvh-18rem)] min-h-105 overflow-hidden rounded-2xl bg-surface shadow-card lg:h-[calc(100dvh-13rem)]"
    >
      <div ref="cyContainer" class="size-full"></div>
    </div>

    <!-- Légende des types -->
    <ul
      v-if="!loading && Object.keys(typeColors).length"
      class="mt-3 flex flex-wrap gap-x-4 gap-y-2 px-1"
    >
      <li
        v-for="(color, typeName) in typeColors"
        :key="typeName"
        class="flex items-center gap-1.5 text-[13px] text-muted"
      >
        <span
          class="inline-block size-3 rounded-[3px]"
          :style="{ backgroundColor: color }"
          aria-hidden="true"
        ></span>
        {{ typeDisplay(String(typeName)) }}
      </li>
    </ul>

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
import PageHeader from "@/components/common/PageHeader.vue";
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

// ── Couleurs des types (identité « Étiquette ») ──
// Tons sourds et distincts, tous lisibles sous un texte blanc gras (contraste ≥ 4,5:1),
// sur fond clair comme sombre ; le pétrole de l'identité en premier.
const typeColors: Record<string, string> = {};
const palette = [
  "#1d5c63", // pétrole
  "#a2342b", // brique
  "#2b5aa8", // bleu
  "#8a5300", // ambre brûlé
  "#6b3f7a", // prune
  "#4f6420", // olive
  "#8a3b5c", // framboise
  "#3b4552", // graphite
];
/** Contour du nœud racine : l'ambre de l'identité. */
const ROOT_BORDER = "#e09a2c";
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
  const borderColor = ROOT_BORDER;

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

/** Styles Cytoscape aux couleurs du thème courant (réappliqués au changement de thème). */
function graphStyle(): cytoscape.StylesheetJson {
  const isDark = theme.value === "dark";
  const font = '"Schibsted Grotesk Variable", system-ui, sans-serif';
  const ink = isDark ? "#e8ecea" : "#17202b";
  const muted = isDark ? "#9aa7b0" : "#586270";
  const line = isDark ? "#3a4d58" : "#c9cec6";
  const surface = isDark ? "#17232c" : "#ffffff";
  return [
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
        // Titre sous le badge du type
        label: "data(title)",
        "text-valign": "bottom",
        "text-halign": "center",
        "text-margin-y": 6,
        "font-family": font,
        "font-size": "11px",
        "font-weight": 600,
        color: ink,
        "text-wrap": "wrap",
        "text-max-width": "140px",
      },
    },
    {
      selector: "node[?isRoot]",
      style: { "font-weight": 800, "font-size": "12px" },
    },
    {
      selector: "edge",
      style: {
        width: 1.5,
        "line-color": line,
        "target-arrow-color": line,
        "target-arrow-shape": "triangle",
        "arrow-scale": 1.1,
        "curve-style": "bezier",
        label: "data(label)",
        "font-family": font,
        "font-size": "9px",
        color: muted,
        "text-wrap": "wrap",
        "text-max-width": "120px",
        "text-rotation": "autorotate",
        "text-margin-y": -10,
        "text-background-color": surface,
        "text-background-opacity": 0.9,
        "text-background-padding": "3px",
        "text-background-shape": "roundrectangle",
      },
    },
  ];
}

function renderGraph(graph: GraphResponse) {
  if (!cyContainer.value) return;

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
    style: graphStyle(),
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

// Changement de thème : styles réappliqués sans recharger le graphe.
watch(theme, () => cy?.style(graphStyle()));

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
