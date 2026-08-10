import { useQuery } from '@tanstack/react-query'

import { listNotes } from '../../api/gen/client'
import type { Note } from '../../api/gen/models/note'
import type { NotesPage } from '../../api/gen/models/notesPage'

const PAGE_SIZE = 200

type PageLoader = (limit: number, offset: number) => Promise<NotesPage>

export async function loadAllNotes(loadPage: PageLoader): Promise<NotesPage> {
  const items: Note[] = []
  let offset = 0
  let total = 0

  do {
    const page = await loadPage(PAGE_SIZE, offset)
    items.push(...page.items)
    total = page.total
    offset += page.items.length
    if (page.items.length === 0) break
  } while (offset < total)

  return { items, total, limit: PAGE_SIZE, offset: 0 }
}

export function useAllNotes() {
  return useQuery({
    queryKey: ['/api/v1/notes', 'all'],
    queryFn: () =>
      loadAllNotes(async (limit, offset) => {
        const response = await listNotes({ limit, offset })
        return response.data
      }),
  })
}
