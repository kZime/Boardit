# Blogedit Frontend

## Development

```bash
npm install
npm run dev # use real api
npm run dev:mock # use mock api
```

## Every time you change the API contract, run

```bash
npm run orval
```

generated `src/api/gen/client.ts` from `backend/docs/api/api-contract-v1.yaml`

check `src/mocks/browser.ts` to use fake api handlers.

## styles design

use `@tailwindcss/vite` to generate css classes.

use plugin `@tailwindcss/typography` to generate typography styles.

use `@mdxeditor/editor` to generate markdown editor.

## Debug

remember to clear localstorage before switching between mock and real mode.

```js
localStorage.removeItem('accessToken');
localStorage.removeItem('refreshToken');
location.reload();
```
