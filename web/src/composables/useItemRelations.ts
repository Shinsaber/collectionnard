// Relations d'un item pour sa fiche (views/ItemDetailView.vue) : une page à la fois (l'API les
// pagine, audit Q-001), complétées de leurs méta-types depuis le cache (lib/relations.ts).
import { ref } from "vue";
import { itemApi } from "@/api/client";
import { useMetaTypes } from "@/stores/metaTypes";
import { byId, withTypes } from "@/lib/relations";
import type { ItemRelation } from "@/api/types";

/** Taille de page : toutes les relations d'un item ordinaire d'un coup, un nœud très lié est paginé. */
export const RELATIONS_PER_PAGE = 100;

export function useItemRelations() {
  const metaTypes = useMetaTypes();
  const relations = ref<ItemRelation[]>([]);
  const total = ref(0);
  const page = ref(1);

  /** Charge une page (la courante par défaut) ; les méta-types sont chargés une fois. */
  async function load(itemId: string, p = page.value) {
    await Promise.all([
      metaTypes.loadItemTypes(),
      metaTypes.loadRelationTypes(),
    ]);
    const res = await itemApi.relations(itemId, {
      page: p,
      per_page: RELATIONS_PER_PAGE,
    });
    const itemTypes = byId(metaTypes.itemTypes);
    const relationTypes = byId(metaTypes.relationTypes);
    relations.value = (res.data ?? []).map((r) =>
      withTypes(r, itemTypes, relationTypes),
    );
    total.value = res.total ?? relations.value.length;
    page.value = res.page ?? p;
  }

  /** Retire une relation supprimée sans recharger. */
  function remove(relationId: string) {
    relations.value = relations.value.filter((r) => r.id !== relationId);
    total.value = Math.max(0, total.value - 1);
  }

  return { relations, total, page, load, remove };
}
