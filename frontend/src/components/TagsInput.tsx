import { useState } from "react"
import { $api } from "../api/client"

interface TagsInputProps {
  value: string[]
  onChange: (tags: string[]) => void
  placeholder?: string
}

/** Controlled tag input that renders selected tags as chips. */
export default function TagsInput({
  value,
  onChange,
  placeholder = "Add tags…",
}: TagsInputProps) {
  const [input, setInput] = useState("")

  const prefix = input.trim()
  const { data: suggestionsData } = $api.useQuery(
    "get",
    "/tags",
    { params: { query: { prefix } } },
    { enabled: prefix.length >= 2 }
  )

  function addTag(raw: string) {
    const tag = raw.trim()
    if (!tag || value.includes(tag)) return
    onChange([...value, tag])
    setInput("")
  }

  function removeTag(tag: string) {
    onChange(value.filter((t) => t !== tag))
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault()
      addTag(input)
    } else if (e.key === "Backspace" && !input && value.length > 0) {
      onChange(value.slice(0, -1))
    }
  }

  return (
    <div className="flex items-center gap-1 overflow-x-auto rounded border border-gray-200 px-2 py-1 focus-within:ring-1 focus-within:ring-gray-300">
      {value.map((tag) => (
        <span
          key={tag}
          className="flex items-center gap-1 rounded-full border border-gray-200 bg-gray-100 px-2 py-px text-xs text-gray-700"
        >
          {tag}
          <button
            type="button"
            onClick={() => removeTag(tag)}
            className="cursor-pointer leading-none text-gray-400 hover:text-gray-600"
          >
            &times;
          </button>
        </span>
      ))}
      <datalist id="tags-input-suggestions">
        {(suggestionsData?.tags ?? []).map((t) => (
          <option key={t} value={t} />
        ))}
      </datalist>
      <input
        type="text"
        list="tags-input-suggestions"
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={value.length === 0 ? placeholder : ""}
        className="min-w-16 flex-1 text-xs text-gray-700 placeholder-gray-400 outline-none"
      />
    </div>
  )
}
