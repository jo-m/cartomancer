import { useId, useState } from "react"
import { $api } from "../api/client"
import Badge from "./ui/Badge"

interface TagsInputProps {
  value: string[]
  onChange: (tags: string[]) => void
  placeholder?: string
}

/** Controlled tag input that renders selected tags as chips. */
export default function TagsInput({
  value,
  onChange,
  placeholder = "Add tags...",
}: TagsInputProps) {
  const [input, setInput] = useState("")
  const listId = useId()

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
    if (!/^[\p{L}\p{N}]{2,32}$/u.test(tag)) return
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
    <div className="flex items-center gap-1 overflow-x-auto rounded border border-border bg-panel px-2 py-1 focus-within:border-primary transition-colors">
      {value.map((tag) => (
        <Badge key={tag} onRemove={() => removeTag(tag)}>
          {tag}
        </Badge>
      ))}
      <datalist id={listId}>
        {(suggestionsData?.tags ?? []).map((t) => (
          <option key={t} value={t} />
        ))}
      </datalist>
      <input
        type="text"
        list={listId}
        value={input}
        onChange={(e) => {
          const v = e.target.value
          if (v && !/^[\p{L}\p{N}]*$/u.test(v)) return
          setInput(v)
        }}
        onKeyDown={handleKeyDown}
        placeholder={value.length === 0 ? placeholder : ""}
        className="min-w-16 flex-1 bg-transparent text-xs text-text placeholder-text-muted outline-none"
        aria-label="Add tag"
      />
    </div>
  )
}
