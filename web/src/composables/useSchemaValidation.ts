// Validation des champs d'un formulaire généré (DynamicForm) contre leur JSON Schema :
// à la sortie d'un champ (blur), puis à chaque saisie tant qu'il est en erreur, et pour tous
// les champs à l'envoi (`validate`). Les erreurs renvoyées par l'API passent par `setErrors`.
import { reactive, type ComputedRef } from "vue";
import { useI18n } from "vue-i18n";
import { checkValue, type Issue, type Schema } from "@/lib/validateValue";

export function useSchemaValidation(
  schema: ComputedRef<Schema>,
  model: ComputedRef<Record<string, unknown>>,
) {
  const { t } = useI18n();
  const errors = reactive<Record<string, string>>({});
  const touched = reactive<Record<string, boolean>>({});
  /** Saisie JSON en cours (champs objet / liste sans widget) : gardée telle quelle si invalide. */
  const drafts = reactive<Record<string, string>>({});

  function propOf(name: string): Schema | undefined {
    return (schema.value.properties as Record<string, Schema> | undefined)?.[
      name
    ];
  }

  function isRequired(name: string): boolean {
    return ((schema.value.required as string[] | undefined) ?? []).includes(
      name,
    );
  }

  function message(issue: Issue): string {
    const params = { ...issue.params };
    if (issue.key === "type")
      params.type = t(`errors.check.types.${params.type}`);
    const text = t(`errors.check.${issue.key}`, params);
    return issue.path ? `${issue.path} : ${text}` : text;
  }

  /** Erreur d'un champ pour une valeur, ou "" s'il est valide. */
  function fieldError(name: string, value: unknown): string {
    if (name in drafts) {
      try {
        JSON.parse(drafts[name]!);
      } catch {
        return t("errors.check.json");
      }
    }
    const empty =
      value === undefined ||
      value === null ||
      value === "" ||
      (Array.isArray(value) && value.length === 0);
    if (empty) return isRequired(name) ? t("errors.required") : "";
    const issue = checkValue(propOf(name), value);
    return issue ? message(issue) : "";
  }

  function check(name: string, value: unknown) {
    const e = fieldError(name, value);
    if (e) errors[name] = e;
    else delete errors[name];
  }

  /** Sortie d'un champ : vérifié, puis revérifié à chaque saisie. */
  function touch(name: string) {
    touched[name] = true;
    check(name, model.value[name]);
  }

  /** Après une saisie : revérifie un champ déjà visité ou en erreur. */
  function changed(name: string, value: unknown) {
    if (touched[name] || errors[name]) check(name, value);
  }

  function clearErrors() {
    for (const key of Object.keys(errors)) delete errors[key];
  }

  /** Tous les champs, avant l'envoi : false si l'un est en erreur. */
  function validate(names: string[]): boolean {
    clearErrors();
    for (const name of names) {
      touched[name] = true;
      check(name, model.value[name]);
    }
    return Object.keys(errors).length === 0;
  }

  function setErrors(fieldErrors: Record<string, string>) {
    clearErrors();
    Object.assign(errors, fieldErrors);
  }

  return {
    errors,
    drafts,
    touch,
    changed,
    check,
    validate,
    setErrors,
    clearErrors,
  };
}
