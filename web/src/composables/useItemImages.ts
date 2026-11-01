// Images d'une fiche (views/ItemDetailView.vue) : couverture par défaut et son ratio, toutes les
// images pour la visionneuse, compteur de l'onglet « Images ». Tout se déduit du schéma du type
// (`x-widget: image-gallery`, `x-image-categories`), rien de métier codé en dur.
import { computed, ref, type Ref } from "vue";
import {
  defaultImage,
  galleryUrls,
  imageCategories,
  isImageGallery,
  type Schema,
} from "@/lib/schema";
import type { Item } from "@/api/types";

export function useItemImages(
  item: Ref<Item | null>,
  schema: Ref<Schema | undefined>,
) {
  /** Images de la galerie (toutes catégories) : l'onglet n'apparaît que s'il y en a. */
  const imageCount = computed(() => {
    const it = item.value;
    if (!it) return 0;
    let n = 0;
    for (const key of Object.keys(it.attributes)) {
      if (!isImageGallery(schema.value, key)) continue;
      for (const cat of imageCategories(schema.value, key))
        n +=
          (
            (it.attributes[key] as Record<string, unknown>)?.[cat.name] as
              unknown[] | undefined
          )?.length ?? 0;
    }
    return n;
  });

  const defaultImageUrl = computed(() =>
    item.value ? defaultImage(schema.value, item.value.attributes) : "",
  );

  /** Toutes les images de la fiche, pour la visionneuse ouverte depuis la couverture. */
  const allImageUrls = computed(() =>
    item.value ? galleryUrls(schema.value, item.value.attributes) : [],
  );

  const coverViewerOpen = ref(false);
  function openCover() {
    coverViewerOpen.value = true;
  }

  /** Ratio de la catégorie qui contient l'image par défaut (sinon portrait 2/3). */
  const defaultImageRatio = computed(() => {
    const url = defaultImageUrl.value;
    if (!item.value || !url) return "2 / 3";
    for (const key of Object.keys(item.value.attributes)) {
      if (!isImageGallery(schema.value, key)) continue;
      const data = item.value.attributes[key] as Record<string, unknown>;
      for (const cat of imageCategories(schema.value, key)) {
        if (((data[cat.name] as string[]) ?? []).includes(url))
          return cat.ratio;
      }
    }
    return "2 / 3";
  });

  return {
    imageCount,
    defaultImageUrl,
    allImageUrls,
    defaultImageRatio,
    coverViewerOpen,
    openCover,
  };
}
