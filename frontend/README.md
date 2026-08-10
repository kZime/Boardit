# Boardit frontend

React route composition, public publishing pages, authentication UI, and the lazy-loaded Markdown editor.

## Run

```bash
npm ci
npm run dev       # real API through the Vite proxy
npm run dev:mock  # deterministic MSW API, no backend required
```

When switching between real and mock authentication, clear the local session if needed:

```js
localStorage.removeItem('accessToken')
localStorage.removeItem('refreshToken')
location.reload()
```

## Test and build

```bash
npm run lint
npm test
npm run build
npm run test:e2e
```

## API generation

`backend/docs/api/api-contract-v1.yaml` is the source of truth.

```bash
npm run orval
git diff --exit-code -- src/api/gen
```

Never edit `src/api/gen/**` manually. Update the OpenAPI contract, regenerate, and keep MSW handlers aligned with required generated fields.

## Boundaries

- React Query owns server cache.
- `src/auth/tokenStorage.ts` owns token persistence; Axios does not import React context.
- `src/features/editor` owns pagination, tree UI state, metadata, and save coordination.
- `src/pages` composes route-level UI.
- The Editor route stays lazy-loaded so MDXEditor does not enter the public initial bundle.

See the root [architecture](../docs/architecture.md), [testing strategy](../docs/testing-strategy.md), and [known debt](../docs/known-debt.md) for details.
