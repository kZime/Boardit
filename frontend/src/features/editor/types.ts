export type Visibility = 'private' | 'public' | 'unlisted'

export interface PageDetails {
  title: string
  coverUrl: string
  description: string
  tags: string
  visibility: Visibility
}
