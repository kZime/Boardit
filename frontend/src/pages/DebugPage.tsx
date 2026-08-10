import { useListNotes, useUpdateNote } from '../api/gen/client'

export default function DebugNotes() {
  const { data, isLoading } = useListNotes({ limit: 10, offset: 0 })
  const update = useUpdateNote()
  const firstNote = data?.data.items[0]

  if (isLoading) return <div>Loading…</div>
  return (
    <div>
      <pre>{JSON.stringify(data, null, 2)}</pre>
      <button
        onClick={() =>
          firstNote && update.mutate({
            id: firstNote.id,
            data: { content_md: '# hi', version: firstNote.version },
          })
        }
        disabled={!firstNote}
      >
        Patch one note
      </button>
    </div>
  )
}
