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
import {
  createTypeColors,
  graphStyle,
  makeTypeSvg,
  nodeWidth,
} from "@/lib/graphStyle";
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

// ── Couleurs des types (identité « Étiquette », lib/graphStyle) ──
const types = createTypeColors();
const typeColors = types.colors;

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
  const typeLabel = typeDisplay(n.type);
  return {
    title: nodeTitle(n),
    typeSvg: makeTypeSvg(typeLabel, types.colorOf(n.type), isRoot),
    nodeWidth: nodeWidth(typeLabel),
  };
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

/** Styles Cytoscape du thème courant (lib/graphStyle), réappliqués au changement de thème. */
const currentStyle = () => graphStyle(theme.value === "dark");

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
          color: types.colorOf(n.type),
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
    style: currentStyle(),
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
watch(theme, () => cy?.style(currentStyle()));

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
