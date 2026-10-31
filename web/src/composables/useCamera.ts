import { ref, type Ref } from "vue";

/** Pourquoi la caméra n'a pas démarré : aucune ou inaccessible, accès refusé, lecture impossible. */
export type CameraFailure = "none" | "denied" | "failed";

export type CameraState = "idle" | "starting" | "ready" | "unavailable";

/**
 * Caméra arrière affichée dans un `<video>` (scanner de codes-barres, prise de vue d'une
 * couverture) : démarrage par `getUserMedia`, lampe torche quand la piste l'offre, arrêt des pistes.
 * Les textes affichés restent au composant appelant (le repli proposé n'est pas le même partout).
 */
export function useCamera(video: Ref<HTMLVideoElement | null>) {
  const state = ref<CameraState>("idle");
  const torchAvailable = ref(false);
  const torchOn = ref(false);
  let stream: MediaStream | null = null;
  let stopped = true;

  /** Démarre l'aperçu à la résolution souhaitée ; null si tout va bien, sinon la raison de l'échec. */
  async function start(
    width: number,
    height: number,
  ): Promise<{ reason: CameraFailure; error?: unknown } | null> {
    stopped = false;
    state.value = "starting";
    const media = navigator.mediaDevices;
    if (typeof media?.getUserMedia !== "function") return unavailable("none");
    try {
      stream = await media.getUserMedia({
        audio: false,
        video: {
          facingMode: { ideal: "environment" },
          width: { ideal: width },
          height: { ideal: height },
        },
      });
    } catch (e) {
      const denied = e instanceof DOMException && e.name === "NotAllowedError";
      return unavailable(denied ? "denied" : "none");
    }
    if (stopped || !video.value) {
      release();
      return unavailable("none");
    }
    video.value.srcObject = stream;
    try {
      await video.value.play();
    } catch (e) {
      return unavailable("failed", e);
    }
    setupTorch(stream);
    state.value = "ready";
    return null;
  }

  function unavailable(reason: CameraFailure, error?: unknown) {
    release();
    state.value = "unavailable";
    return { reason, error };
  }

  /** Piste vidéo en cours (prise de vue par `ImageCapture`). */
  function track(): MediaStreamTrack | null {
    return stream?.getVideoTracks()[0] ?? null;
  }

  // ── Lampe torche ─────────────────────────────────────────────────────
  // `torch` n'est pas dans les types du DOM (capacité propre aux caméras de téléphone).
  function setupTorch(s: MediaStream) {
    const caps = s.getVideoTracks()[0]?.getCapabilities?.() as
      { torch?: boolean } | undefined;
    torchAvailable.value = !!caps?.torch;
    torchOn.value = false;
  }

  /** Allume ou éteint la lampe ; rejette si la piste refuse (à notifier par l'appelant). */
  async function toggleTorch() {
    const t = track();
    if (!t) return;
    await t.applyConstraints({
      advanced: [{ torch: !torchOn.value }],
    } as unknown as Parameters<MediaStreamTrack["applyConstraints"]>[0]);
    torchOn.value = !torchOn.value;
  }

  // ── Arrêt ────────────────────────────────────────────────────────────
  function release() {
    stream?.getTracks().forEach((t) => t.stop());
    stream = null;
  }

  function stop() {
    stopped = true;
    release();
    if (video.value) video.value.srcObject = null;
    torchAvailable.value = false;
    torchOn.value = false;
    state.value = "idle";
  }

  return { state, torchAvailable, torchOn, start, stop, track, toggleTorch };
}
