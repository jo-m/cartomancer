import { useCallback, useMemo } from "react"
import { useSearchParams } from "react-router-dom"

/**
 * Defines how a single URL search parameter is parsed and serialized.
 * T is the TypeScript type of the value.
 */
export interface ParamDef<T> {
  /** The default value, used when the parameter is absent from the URL. */
  defaultValue: T
  /** Parses a URL string into the typed value. */
  parse: (raw: string) => T
  /** Serializes a typed value into a URL string. Returns "" to omit the parameter. */
  serialize: (value: T) => string
  /** Compares two values for equality. Defaults to ===. */
  equals?: (a: T, b: T) => boolean
}

/** Extracts the value type from a ParamDef. */
type ValueOf<D> = D extends ParamDef<infer T> ? T : never

/** Maps a schema of ParamDefs to a plain object of their value types. */
type StateFrom<S extends Record<string, ParamDef<unknown>>> = {
  [K in keyof S]: ValueOf<S[K]>
}

/** Creates a string parameter definition. */
export function stringParam(defaultValue = ""): ParamDef<string> {
  return {
    defaultValue,
    parse: (raw) => raw,
    serialize: (value) => value,
  }
}

/** Creates a number parameter definition. */
export function numberParam(defaultValue: number): ParamDef<number> {
  return {
    defaultValue,
    parse: (raw) => {
      const n = Number(raw)
      return Number.isFinite(n) ? n : defaultValue
    },
    serialize: (value) => String(value),
  }
}

/** Creates a boolean parameter definition, serialized as "1"/"0". */
export function boolParam(defaultValue = false): ParamDef<boolean> {
  return {
    defaultValue,
    parse: (raw) => raw === "1",
    serialize: (value) => (value ? "1" : ""),
  }
}

/** Creates a number array parameter definition, serialized as comma-separated values. */
export function numArrayParam(defaultValue: number[] = []): ParamDef<number[]> {
  return {
    defaultValue,
    parse: (raw) =>
      raw
        .split(",")
        .map(Number)
        .filter((n) => Number.isFinite(n)),
    serialize: (value) => value.join(","),
    equals: (a, b) => a.length === b.length && a.every((v, i) => v === b[i]),
  }
}

/** Creates a string array parameter definition, serialized as comma-separated values. */
export function strArrayParam(defaultValue: string[] = []): ParamDef<string[]> {
  return {
    defaultValue,
    parse: (raw) => raw.split(",").filter(Boolean),
    serialize: (value) => value.join(","),
    equals: (a, b) => a.length === b.length && a.every((v, i) => v === b[i]),
  }
}

/**
 * Creates a nullable numeric range parameter definition.
 * Serialized as "min,max"; null when absent.
 */
export function rangeParam(): ParamDef<[number, number] | null> {
  return {
    defaultValue: null,
    parse: (raw) => {
      const parts = raw.split(",").map(Number)
      if (parts.length === 2 && parts.every(Number.isFinite))
        return parts as [number, number]
      return null
    },
    serialize: (value) => (value ? `${value[0]},${value[1]}` : ""),
    equals: (a, b) => {
      if (a === null && b === null) return true
      if (a === null || b === null) return false
      return a[0] === b[0] && a[1] === b[1]
    },
  }
}

/**
 * Creates a string literal union parameter definition.
 * Falls back to defaultValue if the URL value is not in the allowed set.
 */
export function enumParam<T extends string>(
  defaultValue: T,
  allowed: readonly T[]
): ParamDef<T> {
  return {
    defaultValue,
    parse: (raw) =>
      (allowed as readonly string[]).includes(raw) ? (raw as T) : defaultValue,
    serialize: (value) => value,
  }
}

function isDefault<T>(def: ParamDef<T>, value: T): boolean {
  const eq = def.equals ?? ((a: T, b: T) => a === b)
  return eq(value, def.defaultValue)
}

/**
 * Syncs a typed state object to URL search parameters.
 *
 * Parameters at their default value are omitted from the URL.
 * Returns a [state, setState] tuple where setState accepts a partial update.
 *
 * @param schema - a record mapping parameter names to ParamDef definitions.
 */
export function useUrlState<S extends Record<string, ParamDef<unknown>>>(
  schema: S
): [StateFrom<S>, (update: Partial<StateFrom<S>>) => void] {
  const [searchParams, setSearchParams] = useSearchParams()

  const state = useMemo(() => {
    const result: Record<string, unknown> = {}
    for (const [key, def] of Object.entries(schema)) {
      const raw = searchParams.get(key)
      result[key] = raw !== null ? def.parse(raw) : def.defaultValue
    }
    return result as StateFrom<S>
  }, [schema, searchParams])

  const setState = useCallback(
    (update: Partial<StateFrom<S>>) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          for (const [key, value] of Object.entries(update)) {
            const def = schema[key]
            if (!def) continue
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            if (isDefault(def, value as any)) {
              next.delete(key)
            } else {
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              const serialized = def.serialize(value as any)
              if (serialized === "") {
                next.delete(key)
              } else {
                next.set(key, serialized)
              }
            }
          }
          return next
        },
        { replace: true }
      )
    },
    [schema, setSearchParams]
  )

  return [state, setState]
}
