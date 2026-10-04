import { useEffect, useRef, useState, type ReactNode } from "react"

export type FilterState = {
  language: string
  minRating: string
  minYear: string
  maxYear: string
}

const EMPTY_FILTERS: FilterState = { language: "", minRating: "", minYear: "", maxYear: "" }

const LANGUAGES: Array<{ value: string; label: string }> = [
  { value: "en", label: "English" },
  { value: "fr", label: "French" },
  { value: "de", label: "German" },
  { value: "es", label: "Spanish" },
  { value: "it", label: "Italian" },
  { value: "pt", label: "Portuguese" },
]

const RATINGS: Array<{ value: string; label: string }> = [
  { value: "3.5", label: "3.5 & up" },
  { value: "4", label: "4.0 & up" },
  { value: "4.5", label: "4.5 & up" },
]

const currentYear = new Date().getFullYear()

// Plain-language year ranges; the two bounds map straight onto the API's
// min_year/max_year parameters.
const YEAR_PRESETS: Array<{ label: string; min?: string; max?: string }> = [
  { label: "Past 10 years", min: String(currentYear - 10) },
  { label: "Past 50 years", min: String(currentYear - 50) },
  { label: "21st century", min: "2000" },
  { label: "20th century", min: "1900", max: "1999" },
  { label: "Before 1950", max: "1949" },
]

function activeYearPreset(filters: FilterState) {
  return YEAR_PRESETS.find(
    (preset) => (preset.min ?? "") === filters.minYear && (preset.max ?? "") === filters.maxYear,
  )
}

// Human label for the year filter button and chip: a preset name when the
// bounds match one, otherwise the custom range spelled out.
function yearFilterLabel(filters: FilterState): string {
  const preset = activeYearPreset(filters)
  if (preset) return preset.label
  const { minYear, maxYear } = filters
  if (minYear && maxYear) return `${minYear} to ${maxYear}`
  if (minYear) return `after ${minYear}`
  if (maxYear) return `before ${maxYear}`
  return ""
}

const btnBase =
  "inline-flex h-9 items-center gap-1.5 rounded-md border px-3 font-sans text-sm shadow-xs transition-colors"
const btnIdle = `${btnBase} border-input bg-white text-muted-foreground hover:border-primary/40 hover:text-foreground`
const btnActive = `${btnBase} border-primary bg-primary/10 font-medium text-primary hover:bg-primary/15`
const panelClass =
  "absolute left-0 top-10 z-20 min-w-60 rounded-[10px] border border-border bg-card p-1.5 shadow-[0_10px_28px_rgba(60,50,30,0.12)] max-sm:static max-sm:mt-2 max-sm:w-full max-sm:shadow-none"

function Dropdown({
  ariaLabel,
  label,
  active,
  onClear,
  children,
}: {
  ariaLabel: string
  label: string
  active: boolean
  onClear?: () => void
  children: (close: () => void) => ReactNode
}) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const close = () => setOpen(false)

  useEffect(() => {
    if (!open) return
    function onPointerDown(event: PointerEvent) {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false)
    }
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") setOpen(false)
    }
    document.addEventListener("pointerdown", onPointerDown)
    document.addEventListener("keydown", onKeyDown)
    return () => {
      document.removeEventListener("pointerdown", onPointerDown)
      document.removeEventListener("keydown", onKeyDown)
    }
  }, [open])

  return (
    <div className="relative" ref={rootRef}>
      <button
        aria-expanded={open}
        aria-haspopup="listbox"
        className={active ? btnActive : btnIdle}
        onClick={() => setOpen((value) => !value)}
        type="button"
      >
        <span className="sr-only">{ariaLabel}: </span>
        <span>{label}</span>
        <span aria-hidden="true" className="text-[10px] opacity-70">
          ▾
        </span>
        {active && onClear && (
          <span
            aria-label={`Clear ${ariaLabel}`}
            className="-mr-1 ml-0.5 inline-flex size-[18px] items-center justify-center rounded-full text-[13px] leading-none hover:bg-primary hover:text-primary-foreground"
            onClick={(event) => {
              event.stopPropagation()
              onClear()
            }}
            role="button"
          >
            ×
          </span>
        )}
      </button>
      {open && (
        <div aria-label={ariaLabel} className={panelClass} role="listbox">
          {children(close)}
        </div>
      )}
    </div>
  )
}

function Option({
  selected,
  onClick,
  children,
}: {
  selected: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      aria-selected={selected}
      className="group flex w-full items-center justify-between gap-3 rounded-md px-2.5 py-2 text-left font-sans text-sm hover:bg-secondary aria-selected:font-medium"
      onClick={onClick}
      role="option"
      type="button"
    >
      {children}
      <span aria-hidden="true" className="hidden font-bold text-primary group-aria-selected:block">
        ✓
      </span>
    </button>
  )
}

function ActiveChip({
  filter,
  value,
  onRemove,
}: {
  filter: string
  value: string
  onRemove: () => void
}) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-primary bg-primary/5 py-1 pl-3 pr-1.5 font-sans text-[13px] text-primary">
      <span>
        <span className="font-normal opacity-75">{filter}:</span> {value}
      </span>
      <button
        aria-label={`Remove ${filter} filter`}
        className="inline-flex size-[18px] items-center justify-center rounded-full text-[13px] leading-none hover:bg-primary hover:text-primary-foreground"
        onClick={onRemove}
        type="button"
      >
        ×
      </button>
    </span>
  )
}

export function FilterToolbar({
  filters,
  onChange,
  count,
}: {
  filters: FilterState
  onChange: (filters: FilterState) => void
  count?: ReactNode
}) {
  const languageLabel = LANGUAGES.find((item) => item.value === filters.language)?.label ?? ""
  const ratingLabel = RATINGS.find((item) => item.value === filters.minRating)?.label ?? ""
  const yearLabel = yearFilterLabel(filters)
  const preset = activeYearPreset(filters)

  // Custom-range inputs hold a draft while being typed; null means "not
  // editing", and the inputs then mirror the applied URL values.
  const [draft, setDraft] = useState<{ min: string; max: string } | null>(null)

  const chips: Array<{ key: string; filter: string; value: string; clear: () => void }> = []
  if (languageLabel) {
    chips.push({ key: "language", filter: "Language", value: languageLabel, clear: () => onChange({ ...filters, language: "" }) })
  }
  if (ratingLabel) {
    chips.push({ key: "minRating", filter: "Rating", value: ratingLabel, clear: () => onChange({ ...filters, minRating: "" }) })
  }
  if (yearLabel) {
    chips.push({
      key: "year",
      filter: "Published",
      value: yearLabel,
      clear: () => onChange({ ...filters, minYear: "", maxYear: "" }),
    })
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2.5">
        {count}
        <Dropdown
          active={Boolean(languageLabel)}
          ariaLabel="Language"
          label={languageLabel || "Any language"}
          onClear={() => onChange({ ...filters, language: "" })}
        >
          {(close) => (
            <>
              <Option
                selected={!filters.language}
                onClick={() => {
                  onChange({ ...filters, language: "" })
                  close()
                }}
              >
                Any language
              </Option>
              {LANGUAGES.map((item) => (
                <Option
                  key={item.value}
                  selected={filters.language === item.value}
                  onClick={() => {
                    onChange({ ...filters, language: item.value })
                    close()
                  }}
                >
                  {item.label}
                </Option>
              ))}
            </>
          )}
        </Dropdown>
        <Dropdown
          active={Boolean(ratingLabel)}
          ariaLabel="Minimum rating"
          label={ratingLabel || "Any rating"}
          onClear={() => onChange({ ...filters, minRating: "" })}
        >
          {(close) => (
            <>
              <Option
                selected={!filters.minRating}
                onClick={() => {
                  onChange({ ...filters, minRating: "" })
                  close()
                }}
              >
                Any rating
              </Option>
              {RATINGS.map((item) => (
                <Option
                  key={item.value}
                  selected={filters.minRating === item.value}
                  onClick={() => {
                    onChange({ ...filters, minRating: item.value })
                    close()
                  }}
                >
                  {item.label}
                </Option>
              ))}
            </>
          )}
        </Dropdown>
        <Dropdown
          active={Boolean(yearLabel)}
          ariaLabel="Published"
          label={yearLabel || "Any year"}
          onClear={() => onChange({ ...filters, minYear: "", maxYear: "" })}
        >
          {(close) => (
            <>
              <Option
                selected={!filters.minYear && !filters.maxYear}
                onClick={() => {
                  onChange({ ...filters, minYear: "", maxYear: "" })
                  setDraft(null)
                  close()
                }}
              >
                Any year
              </Option>
              {YEAR_PRESETS.map((item) => (
                <Option
                  key={item.label}
                  selected={preset?.label === item.label}
                  onClick={() => {
                    onChange({ ...filters, minYear: item.min ?? "", maxYear: item.max ?? "" })
                    setDraft(null)
                    close()
                  }}
                >
                  {item.label}
                </Option>
              ))}
              <div className="mx-2 my-1.5 h-px bg-border" />
              <Option
                selected={!preset && Boolean(yearLabel)}
                onClick={() => setDraft({ min: filters.minYear, max: filters.maxYear })}
              >
                Custom range…
              </Option>
              {draft && (
                <div className="px-2.5 pb-3 pt-2">
                  <label className="mb-1.5 block font-sans text-xs text-muted-foreground" htmlFor="year-from">
                    Published between
                  </label>
                  <div className="flex items-center gap-1.5">
                    <input
                      className="h-8 w-20 rounded-md border border-input bg-white px-2 font-sans text-sm"
                      id="year-from"
                      onChange={(event) => setDraft({ ...draft, min: event.target.value })}
                      placeholder="1990"
                      type="number"
                      value={draft.min}
                    />
                    <span aria-hidden="true" className="font-sans text-sm text-muted-foreground">
                      and
                    </span>
                    <input
                      className="h-8 w-20 rounded-md border border-input bg-white px-2 font-sans text-sm"
                      id="year-to"
                      onChange={(event) => setDraft({ ...draft, max: event.target.value })}
                      placeholder="2025"
                      type="number"
                      value={draft.max}
                    />
                    <button
                      className="ml-1 h-8 rounded-md bg-primary px-3 font-sans text-[13px] font-medium text-primary-foreground hover:bg-primary/90"
                      onClick={() => {
                        onChange({ ...filters, minYear: draft.min, maxYear: draft.max })
                        setDraft(null)
                        close()
                      }}
                      type="button"
                    >
                      Apply
                    </button>
                  </div>
                </div>
              )}
            </>
          )}
        </Dropdown>
        {chips.length > 0 && (
          <button
            className="inline-flex h-9 items-center font-sans text-[13px] text-muted-foreground underline underline-offset-[3px] hover:text-foreground"
            onClick={() => {
              onChange(EMPTY_FILTERS)
              setDraft(null)
            }}
            type="button"
          >
            Clear all
          </button>
        )}
      </div>

      {chips.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {chips.map((chip) => (
            <ActiveChip filter={chip.filter} key={chip.key} onRemove={chip.clear} value={chip.value} />
          ))}
        </div>
      )}
    </div>
  )
}
