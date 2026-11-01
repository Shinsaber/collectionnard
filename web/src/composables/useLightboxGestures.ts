// Gestes de la visionneuse (components/common/ImageLightbox.vue) : un doigt déplace l'image
// agrandie ou balaie vers l'image voisine, deux doigts pincent autour du point visé, un
// double-tap alterne ajusté / agrandi. L'état (zoom, décalage) reste au composant ; ce module
// ne fait que le piloter (audit Q-008 : composant ramené sous les 400 lignes).
import type { Ref } from "vue";
import {
  clampZoom,
  distance,
  midpoint,
  swipeDirection,
  zoomAround,
  type Offset,
} from "@/lib/lightbox";

/** Deux appuis en moins de 300 ms et à moins de 30 px : double-tap. */
const DOUBLE_TAP_MS = 300;
const DOUBLE_TAP_PX = 30;

export interface LightboxGestureHost {
  /** Scène où les pointeurs sont capturés ; les coordonnées sont relatives à son centre. */
  stage: Ref<HTMLElement | null>;
  zoom: Ref<number>;
  offset: Ref<Offset>;
  /** Décalage horizontal d'un balayage en cours (zoom 1). */
  swipeX: Ref<number>;
  /** Vrai pendant un geste : les transitions sont coupées. */
  gesturing: Ref<boolean>;
  /** Nombre d'images (le balayage n'a de sens qu'à partir de deux). */
  count: () => number;
  /** Remet le décalage dans les bornes. */
  settle: () => void;
  /** Image voisine (delta ±1). */
  step: (delta: number) => void;
  /** Double-tap : agrandi autour du point, ou retour à l'ajustement. */
  toggleZoom: (point: Offset) => void;
  close: () => void;
}

interface Gesture {
  /** Décalage et zoom au début du geste. */
  offset: Offset;
  zoom: number;
  /** Un doigt : point de départ ; deux doigts : écart et centre de départ. */
  start: Offset;
  dist: number;
  moved: boolean;
  pinched: boolean;
}

export function useLightboxGestures(host: LightboxGestureHost) {
  const pointers = new Map<number, Offset>();
  let gesture: Gesture | null = null;
  let lastTap: { at: number; point: Offset } | null = null;
  /** Un glissement vient de finir : le clic qui suit ne ferme pas. */
  let justDragged = false;

  /** Coordonnées d'un événement relatives au centre de la scène. */
  function toStage(e: { clientX: number; clientY: number }): Offset {
    const r = host.stage.value?.getBoundingClientRect();
    if (!r) return { x: 0, y: 0 };
    return {
      x: e.clientX - (r.left + r.width / 2),
      y: e.clientY - (r.top + r.height / 2),
    };
  }

  function beginGesture() {
    const pts = [...pointers.values()];
    if (!pts.length) return;
    const two = pts.length >= 2;
    gesture = {
      offset: host.offset.value,
      zoom: host.zoom.value,
      start: two ? midpoint(pts[0]!, pts[1]!) : pts[0]!,
      dist: two ? distance(pts[0]!, pts[1]!) : 0,
      moved: gesture?.moved ?? false,
      pinched: two || (gesture?.pinched ?? false),
    };
    host.gesturing.value = true;
  }

  function onDown(e: PointerEvent) {
    if (e.button !== 0 && e.pointerType === "mouse") return;
    host.stage.value?.setPointerCapture(e.pointerId);
    pointers.set(e.pointerId, toStage(e));
    host.swipeX.value = 0;
    beginGesture();
  }

  function onMove(e: PointerEvent) {
    if (!pointers.has(e.pointerId) || !gesture) return;
    const p = toStage(e);
    pointers.set(e.pointerId, p);
    const pts = [...pointers.values()];
    if (pts.length >= 2) {
      // Pincement : zoom autour du centre de départ, puis le centre suit les doigts.
      const d = distance(pts[0]!, pts[1]!);
      const c = midpoint(pts[0]!, pts[1]!);
      const r = zoomAround(
        gesture.offset,
        gesture.zoom,
        clampZoom((gesture.zoom * d) / Math.max(1, gesture.dist)),
        gesture.start,
      );
      host.zoom.value = r.zoom;
      host.offset.value = {
        x: r.offset.x + (c.x - gesture.start.x),
        y: r.offset.y + (c.y - gesture.start.y),
      };
      host.settle();
      gesture.moved = true;
      return;
    }
    const dx = p.x - gesture.start.x;
    const dy = p.y - gesture.start.y;
    if (Math.hypot(dx, dy) > 6) gesture.moved = true;
    if (host.zoom.value > 1) {
      host.offset.value = {
        x: gesture.offset.x + dx,
        y: gesture.offset.y + dy,
      };
      host.settle();
    } else if (host.count() > 1 && Math.abs(dx) > Math.abs(dy)) {
      host.swipeX.value = dx;
    }
  }

  function onUp(e: PointerEvent) {
    if (!pointers.has(e.pointerId)) return;
    const p = toStage(e);
    pointers.delete(e.pointerId);
    const g = gesture;
    if (pointers.size > 0) {
      // Un doigt reste après un pincement : il reprend un déplacement sans à-coup.
      beginGesture();
      return;
    }
    gesture = null;
    host.gesturing.value = false;
    if (!g) return;
    justDragged = g.moved;
    if (!g.pinched && host.zoom.value <= 1 && host.swipeX.value) {
      const dir = swipeDirection(host.swipeX.value, p.y - g.start.y);
      host.swipeX.value = 0;
      if (dir) {
        host.step(-dir);
        return;
      }
    }
    if (!g.moved && !g.pinched) {
      const now = performance.now();
      if (
        lastTap &&
        now - lastTap.at < DOUBLE_TAP_MS &&
        distance(lastTap.point, p) < DOUBLE_TAP_PX
      ) {
        lastTap = null;
        host.toggleZoom(p);
      } else {
        lastTap = { at: now, point: p };
      }
    }
  }

  /** Clic sur le fond : ferme, sauf au relâchement d'un glissement. */
  function onBackdropClick() {
    if (justDragged) {
      justDragged = false;
      return;
    }
    host.close();
  }

  return { toStage, onDown, onMove, onUp, onBackdropClick };
}
