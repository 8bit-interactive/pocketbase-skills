# SPA Routing for PocketBase and Pockethost

Use this pattern when the frontend is a client-side SPA served from `pb_public`.

## Recommendation

Prefer an explicit SPA mount path instead of a generic catch-all fallback.

Default contract:

- `GET /` serves `index.html`
- `GET /page` serves `index.html`
- `GET /page/*` serves `index.html`
- `/api/*` remains dedicated to PocketBase APIs and custom backend routes
- static assets are served from real public paths such as `/assets/*` or `/dist/*`
- `/page` is a routing namespace, not a real asset directory under `pb_public`

This is simpler and safer than trying to detect whether an arbitrary request should fall back to `index.html`.

## Frontend Implication

The frontend router should use `/page` as its navigation base, but the asset base should stay on a real static path.

Examples:

- React Router: `basename="/page"`
- Vue Router: history base `/page`
- Vite: keep `base: "/"` so compiled assets stay under `/assets/*`
- other bundlers: emit assets under `/assets/*` or `/dist/*`, not under `/page/*`

If the frontend assumes it lives at `/`, but PocketBase serves SPA navigation under `/page`, deep links and client-side navigation will be inconsistent.

If the bundler emits files under `/page/*`, you lose the routing contract because `/page` stops being a pure SPA fallback namespace.

## PocketBase Hook Example

This example keeps the SPA fallback explicit:

- `/` returns the SPA entrypoint
- `/page` returns the SPA entrypoint
- `/page/*` serves `index.html` through PocketBase's static fallback behavior
- `/api/*` is untouched because the SPA route is scoped to `/page`

```javascript
/// <reference path="../pb_data/types.d.ts" />

routerAdd('GET', '/', function (e) {
  return serveSpaIndex(e)
})

routerAdd('GET', '/page', function (e) {
  return serveSpaIndex(e)
})

routerAdd(
  'GET',
  '/page/{path...}',
  $apis.static($os.dirFS($app.rootCmd.rootDir + '/pb_public'), true)
)

function serveSpaIndex(e) {
  return e.fileFS($os.dirFS($app.rootCmd.rootDir + '/pb_public'), 'index.html')
}
```

## Suggested Layout

Typical frontend output:

```text
pb_public/
├── index.html
├── assets/
│   ├── app.js
│   └── app.css
├── dist/
│   └── optional-extra-bundles.js
└── favicon.ico
```

Do not treat `/page` as a real folder contract.

The copyable `Makefile` template also fails fast if `pb_public/page/` exists physically in the project.

Avoid this layout:

```text
pb_public/
└── page/
    ├── app.js
    └── app.css
```

Direct asset requests should use their real public paths, for example:

- `/assets/app.js`
- `/assets/app.css`
- `/dist/optional-extra-bundles.js`
- `/favicon.ico`

The SPA navigation URLs should live under `/page`, for example:

- `/page`
- `/page/login`
- `/page/settings/profile`

## Why This Pattern

This avoids several common problems:

- no filesystem existence check on every unknown request
- no accidental rewrite of API traffic to `index.html`
- no accidental coupling between SPA routes and asset directories
- a clearer separation between backend routes, navigation URLs, and static bundles

Using `$apis.static(..., true)` under `/page/{path...}` delegates the fallback behavior to PocketBase's built-in static handler instead of reimplementing it manually.

That fallback only stays clean if `/page` remains a phantom route namespace and not a place where real built files are published.

## Testing Guidance

Validate the routing behavior with Playwright.

At minimum, cover:

1. `GET /` loads the SPA shell
2. `GET /page` loads the SPA shell
3. `GET /page/some/deep/link` loads the SPA shell
4. `/api/health` or another API route is not rewritten to `index.html`
5. `/assets/app.js` or another direct asset URL still loads successfully
6. no build output is emitted under `pb_public/page/`

When the agent drives these tests, prefer `$playwright-cli`.
