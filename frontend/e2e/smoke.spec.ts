import { expect, test } from '@playwright/test'

test('public post list loads from the mock API', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByRole('heading', { name: 'Public posts' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Public Page Example' })).toBeVisible()
})

test('mock login opens the editor', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('button', { name: 'DEV: Skip Login (Mock Mode)' }).click()

  await expect(page).toHaveURL(/\/editor$/)
  await expect(page.getByText('Welcome to Boardit', { exact: true }).first()).toBeVisible()
})

test('an author can edit and publish a note', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('button', { name: 'DEV: Skip Login (Mock Mode)' }).click()
  await page.getByRole('button', { name: 'Welcome to Boardit', exact: true }).click()

  const title = page.getByPlaceholder('Enter page title...')
  await expect(title).toHaveValue('Welcome to Boardit')
  await title.fill('Published by smoke test')
  await page.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByText('Saved successfully!')).toBeVisible()
  await expect(page.getByText('Saved successfully!')).toBeHidden({ timeout: 5_000 })
  await page.getByRole('button', { name: 'Edit Details' }).click()
  await page.getByLabel('Visibility').selectOption('public')
  await page.getByRole('button', { name: 'Save Changes' }).click()
  await expect(page.getByText('Saved successfully!')).toBeVisible()

  await page.getByRole('link', { name: 'Boardit' }).click()
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByRole('heading', { name: 'Published by smoke test' })).toBeVisible()
})

test('unlisted notes are published for direct-link access', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('button', { name: 'DEV: Skip Login (Mock Mode)' }).click()
  await page.getByRole('button', { name: 'Welcome to Boardit', exact: true }).click()
  await page.getByRole('button', { name: 'Edit Details' }).click()
  await page.getByLabel('Visibility').selectOption('unlisted')

  const updateRequest = page.waitForRequest(
    (request) => request.method() === 'PATCH' && /\/api\/v1\/notes\/\d+$/.test(request.url()),
  )
  await page.getByRole('button', { name: 'Save Changes' }).click()
  expect((await updateRequest).postDataJSON()).toMatchObject({
    is_published: true,
    visibility: 'unlisted',
  })
})
