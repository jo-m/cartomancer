// @vitest-environment jsdom
import { describe, it, expect } from "vitest"
import { act, renderHook } from "@testing-library/react"
import { MemoryRouter, useLocation } from "react-router-dom"
import type { ReactNode } from "react"
import {
  boolParam,
  enumParam,
  numArrayParam,
  numberParam,
  rangeParam,
  strArrayParam,
  stringParam,
  useUrlState,
  type ParamDef,
} from "./useUrlState"

describe("stringParam", () => {
  it("uses '' as the default by default", () => {
    const p = stringParam()
    expect(p.defaultValue).toBe("")
    expect(p.parse("hello")).toBe("hello")
    expect(p.serialize("hello")).toBe("hello")
  })

  it("accepts a custom default", () => {
    expect(stringParam("x").defaultValue).toBe("x")
  })
})

describe("numberParam", () => {
  it("parses finite numbers", () => {
    const p = numberParam(0)
    expect(p.parse("42")).toBe(42)
    expect(p.parse("-1.5")).toBe(-1.5)
  })

  it("falls back to the default for non-finite input", () => {
    const p = numberParam(7)
    expect(p.parse("not-a-number")).toBe(7)
    expect(p.parse("Infinity")).toBe(7)
    expect(p.parse("NaN")).toBe(7)
  })

  it("serializes via String", () => {
    expect(numberParam(0).serialize(3.14)).toBe("3.14")
  })
})

describe("boolParam", () => {
  it("parses only '1' as true", () => {
    const p = boolParam()
    expect(p.parse("1")).toBe(true)
    expect(p.parse("0")).toBe(false)
    expect(p.parse("true")).toBe(false)
    expect(p.parse("")).toBe(false)
  })

  it("serializes true as '1' and false as '' (omitted)", () => {
    const p = boolParam()
    expect(p.serialize(true)).toBe("1")
    expect(p.serialize(false)).toBe("")
  })
})

describe("numArrayParam", () => {
  it("parses comma-separated numbers and skips non-finite entries", () => {
    const p = numArrayParam()
    expect(p.parse("1,2,3")).toEqual([1, 2, 3])
    expect(p.parse("1,bad,3")).toEqual([1, 3])
  })

  it("serializes by joining with commas", () => {
    expect(numArrayParam().serialize([1, 2, 3])).toBe("1,2,3")
    expect(numArrayParam().serialize([])).toBe("")
  })

  it("equals does element-wise comparison", () => {
    const eq = numArrayParam().equals!
    expect(eq([1, 2], [1, 2])).toBe(true)
    expect(eq([1, 2], [2, 1])).toBe(false)
    expect(eq([1], [1, 2])).toBe(false)
  })
})

describe("strArrayParam", () => {
  it("parses comma-separated strings and drops empties", () => {
    const p = strArrayParam()
    expect(p.parse("a,b,c")).toEqual(["a", "b", "c"])
    expect(p.parse("a,,b")).toEqual(["a", "b"])
    expect(p.parse("")).toEqual([])
  })

  it("equals does element-wise comparison", () => {
    const eq = strArrayParam().equals!
    expect(eq(["a", "b"], ["a", "b"])).toBe(true)
    expect(eq(["a"], ["b"])).toBe(false)
  })
})

describe("rangeParam", () => {
  it("parses valid 2-element tuples", () => {
    expect(rangeParam().parse("1,5")).toEqual([1, 5])
    expect(rangeParam().parse("-2.5,3.5")).toEqual([-2.5, 3.5])
  })

  it("returns null for malformed input", () => {
    const p = rangeParam()
    expect(p.parse("1")).toBeNull()
    expect(p.parse("1,2,3")).toBeNull()
    expect(p.parse("a,b")).toBeNull()
    expect(p.parse("")).toBeNull()
  })

  it("serializes null as '' (omitted) and tuples as 'min,max'", () => {
    const p = rangeParam()
    expect(p.serialize(null)).toBe("")
    expect(p.serialize([1, 5])).toBe("1,5")
  })

  it("equals handles nulls and tuples", () => {
    const eq = rangeParam().equals!
    expect(eq(null, null)).toBe(true)
    expect(eq([1, 2], [1, 2])).toBe(true)
    expect(eq(null, [1, 2])).toBe(false)
    expect(eq([1, 2], null)).toBe(false)
    expect(eq([1, 2], [1, 3])).toBe(false)
  })
})

describe("enumParam", () => {
  type Sort = "asc" | "desc"
  const allowed = ["asc", "desc"] as const

  it("parses values within the allowed set", () => {
    const p = enumParam<Sort>("asc", allowed)
    expect(p.parse("asc")).toBe("asc")
    expect(p.parse("desc")).toBe("desc")
  })

  it("falls back to the default for out-of-set input", () => {
    const p = enumParam<Sort>("asc", allowed)
    expect(p.parse("sideways")).toBe("asc")
    expect(p.parse("")).toBe("asc")
  })
})

interface Probe<S extends Record<string, ParamDef<unknown>>> {
  state: ReturnType<typeof useUrlState<S>>[0]
  setState: ReturnType<typeof useUrlState<S>>[1]
  search: string
}

function renderUrlState<S extends Record<string, ParamDef<unknown>>>(
  schema: S,
  initialUrl = "/"
) {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <MemoryRouter initialEntries={[initialUrl]}>{children}</MemoryRouter>
  )
  return renderHook(
    (): Probe<S> => {
      const [state, setState] = useUrlState(schema)
      const { search } = useLocation()
      return { state, setState, search }
    },
    { wrapper }
  )
}

describe("useUrlState", () => {
  const schema = {
    q: stringParam(""),
    page: numberParam(1),
    star: boolParam(false),
    tags: strArrayParam([]),
    range: rangeParam(),
    sort: enumParam<"asc" | "desc">("asc", ["asc", "desc"] as const),
  }

  it("returns defaults when the URL has no params", () => {
    const { result } = renderUrlState(schema, "/")
    expect(result.current.state).toEqual({
      q: "",
      page: 1,
      star: false,
      tags: [],
      range: null,
      sort: "asc",
    })
  })

  it("parses each param type from the URL", () => {
    const { result } = renderUrlState(
      schema,
      "/?q=hello&page=3&star=1&tags=a,b&range=10,20&sort=desc"
    )
    expect(result.current.state).toEqual({
      q: "hello",
      page: 3,
      star: true,
      tags: ["a", "b"],
      range: [10, 20],
      sort: "desc",
    })
  })

  it("falls back to defaults for invalid values", () => {
    const { result } = renderUrlState(
      schema,
      "/?page=bad&range=1,2,3&sort=sideways"
    )
    expect(result.current.state.page).toBe(1)
    expect(result.current.state.range).toBeNull()
    expect(result.current.state.sort).toBe("asc")
  })

  it("setState writes values into the URL", () => {
    const { result } = renderUrlState(schema, "/")
    act(() => result.current.setState({ q: "hi", page: 2 }))
    const params = new URLSearchParams(result.current.search)
    expect(params.get("q")).toBe("hi")
    expect(params.get("page")).toBe("2")
    expect(result.current.state.q).toBe("hi")
    expect(result.current.state.page).toBe(2)
  })

  it("omits params when their value equals the default", () => {
    const { result } = renderUrlState(schema, "/?q=hi&page=5")
    act(() => result.current.setState({ q: "", page: 1 }))
    const params = new URLSearchParams(result.current.search)
    expect(params.has("q")).toBe(false)
    expect(params.has("page")).toBe(false)
  })

  it("omits params when serialize returns ''", () => {
    const { result } = renderUrlState(schema, "/?star=1")
    act(() => result.current.setState({ star: false }))
    expect(new URLSearchParams(result.current.search).has("star")).toBe(false)
  })

  it("merges partial updates without disturbing other params", () => {
    const { result } = renderUrlState(schema, "/?q=keep&page=2")
    act(() => result.current.setState({ page: 7 }))
    const params = new URLSearchParams(result.current.search)
    expect(params.get("q")).toBe("keep")
    expect(params.get("page")).toBe("7")
  })

  it("ignores keys not in the schema", () => {
    const { result } = renderUrlState(schema, "/?q=keep")
    act(() =>
      (result.current.setState as (u: Record<string, unknown>) => void)({
        unknown: "x",
      })
    )
    const params = new URLSearchParams(result.current.search)
    expect(params.has("unknown")).toBe(false)
    expect(params.get("q")).toBe("keep")
  })

  it("uses the custom equals to decide default-omission for arrays", () => {
    const { result } = renderUrlState(schema, "/?tags=a,b")
    act(() => result.current.setState({ tags: [] }))
    expect(new URLSearchParams(result.current.search).has("tags")).toBe(false)
  })
})
