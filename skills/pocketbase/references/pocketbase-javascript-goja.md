# PocketBase JavaScript Goja Reference

Source:
`https://gist.githubusercontent.com/evaisse/fa95b7c063567b56c73f200d8a073ec3/raw/2aa1c3aad02f59a9b3f02eed6f215cedc646b447/POCKETBASE_JAVASCRIPT_GOJA.md`

This reference summarizes the important runtime constraints and patterns for PocketBase JavaScript running on the embedded Goja engine.

## Runtime Model

PocketBase server-side JavaScript is executed by Goja.

- It is not Node.js.
- It is not a browser runtime.
- It should be treated as ES5-oriented JavaScript with a limited API surface.

Practical consequences:

- No Node.js APIs such as `fs`, `buffer`, or `process`.
- No DOM APIs such as `window`, `document`, or `fetch`.
- No modern JavaScript assumptions unless verified against PocketBase's runtime behavior.

## Module Loading

Do not assume Node-style module behavior everywhere.

Preferred pattern:

```javascript
routerAdd('POST', '/api/example', function (e) {
  var utils = require(`${__hooks}/utils.js`)
  return utils.handleRequest(e)
})
```

Avoid loading local helper modules at top level for code that runs inside route registration or handler flows.

Path guidance:

- Use `${__hooks}` for files under PocketBase hooks.
- Do not rely on `__dirname`.
- Do not rely on `process.cwd()`.
- Do not rely on fragile relative paths when `${__hooks}` can be used.

Export guidance:

```javascript
function handleRequest(e) {
  return e.json(200, { ok: true })
}

exports.handleRequest = handleRequest
```

Prefer:

- `exports.name = value`

Over:

- `module.exports = { ... }`

## Authentication Patterns

Use PocketBase event auth APIs directly.

Preferred pattern:

```javascript
var authRecord = e.auth
var isGuest = !e.auth
var isSuperuser = e.hasSuperuserAuth()
```

Avoid legacy or undocumented access patterns such as:

```javascript
var authRecord = e.get('authRecord')
```

## ES5 Compatibility

Default to ES5-compatible syntax for hooks, migrations, and custom route modules.

Prefer:

```javascript
var selected = null

for (var i = 0; i < items.length; i++) {
  if (items[i].id === targetId) {
    selected = items[i]
    break
  }
}

var callback = function (item) {
  return item.active
}

var message = 'Hello ' + name + ', count=' + count
```

Avoid assuming support for:

- Arrow functions
- `let` and `const`
- Destructuring
- Template literals
- `async` and `await`
- Modern array helpers unless verified in the actual runtime

When in doubt, use:

- `var`
- `function`
- `for` loops
- explicit conditionals

## HTTP Response Handling

PocketBase helper objects may not behave like browser `fetch()` responses.

Preferred pattern:

```javascript
var response = $http.send({
  url: 'https://api.example.com/data',
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(payload)
})

if (response.statusCode === 200) {
  var data = response.json
}
```

Avoid assuming that `response.json()` exists.

## Collections and Migrations

Be explicit and conservative when defining schema and rules.

### Boolean Fields

Boolean fields often behave better when optional and explicitly coerced:

```javascript
new BoolField({
  name: 'success',
  required: false
})

record.set('success', Boolean(success))
```

### Relation Fields

Avoid complex relations until they are clearly needed.

- Be careful with aggressive cascading behavior.
- Resolve related collection IDs dynamically if migrations may run before all collections exist.

Example:

```javascript
var users = app.findCollectionByNameOrId('users')

new RelationField({
  name: 'user',
  collectionId: users ? users.id : '',
  required: false
})
```

### Collection Rules

Prefer simple rules.

- Use simple access rules when possible.
- Move complex validation or branching logic into hooks instead of overloading collection rules.

### Timestamp Fields

Do not assume `created` and `updated` fields are added automatically in every schema design flow.

Add them explicitly when needed:

```javascript
collection.fields.add(
  new AutodateField({
    name: 'created',
    onCreate: true,
    onUpdate: false
  })
)

collection.fields.add(
  new AutodateField({
    name: 'updated',
    onCreate: true,
    onUpdate: true
  })
)
```

## Routes and Proxying

Custom PocketBase routes should remain explicit.

Example:

```javascript
routerAdd('POST', '/api/generate-story', function (e) {
  return require(`${__hooks}/api.js`).generateStory(e)
})
```

Frontend proxying often needs two distinct behaviors:

- custom `/api/...` routes forwarded as-is
- PocketBase collection endpoints rewritten as needed by the frontend proxy

When using the PocketBase JS client from a frontend app, verify the chosen base path carefully so custom routes and collection endpoints both resolve correctly.

## Validation and Type Handling

Validate inputs early and convert values explicitly before persisting them.

Preferred pattern:

```javascript
function trackUsage(userId, requestType, success) {
  if (!userId || typeof userId !== 'string') {
    throw new Error('userId is required and must be a non-empty string')
  }

  if (!requestType) {
    throw new Error('requestType is required')
  }

  if (typeof success !== 'boolean') {
    throw new Error('success is required and must be a boolean')
  }
}
```

Preferred writes:

```javascript
record.set('count', Number(count))
record.set('active', Boolean(active))
record.set('text', String(text))
record.set('data', JSON.stringify(data))
```

For number fields, guard against `null`, `undefined`, and `NaN`.

## Error Handling

Error output in PocketBase JavaScript should capture both the error object and a usable stack.

Preferred pattern:

```javascript
try {
  runRiskyOperation()
} catch (error) {
  console.error('Operation failed:', error)
  console.error('Error stack:', error.stack || new Error().stack)
  throw error
}
```

If an external API call fails, log enough raw response context to make debugging possible.

## Debuggable Route Pattern

Keep route registration thin and move real work to named functions exported from helper modules.

Preferred pattern:

```javascript
routerAdd(
  'POST',
  '/api/generate-story',
  function (e) {
    return require(`${__hooks}/api.js`).generateStory(e)
  },
  $apis.requireAuth()
)
```

```javascript
function generateStory(e) {
  try {
    return e.json(200, { ok: true })
  } catch (error) {
    console.error('Story generation error:', error)
    console.error(
      'Story generation error stack:',
      error.stack || new Error().stack
    )
    return e.json(500, { error: 'Failed to generate story' })
  }
}

exports.generateStory = generateStory
```

This improves:

- stack trace clarity
- module organization
- testability
- local reasoning about route behavior

## Development Tooling

Helpful supporting practices:

- use JSDoc where it improves validation and maintainability
- lint PocketBase JavaScript with rules that do not force modern syntax
- keep shared utilities under `pb_hooks/utils/`
- separate route wiring from business logic

Example structure:

```text
pb_hooks/
├── api.pb.js
├── auth.pb.js
├── lint.pb.js
└── utils/
    ├── api_usage.js
    └── helpers.js
```

## Common Pitfalls

Watch for these recurring failures:

1. Top-level `require()` usage in route-oriented files.
2. ES6+ syntax that parses or behaves differently in Goja.
3. Using `e.get('authRecord')` instead of `e.auth`.
4. Treating boolean fields like always-safe required values.
5. Overcomplicated collection rules replacing simple hooks.
6. Assuming number fields already contain valid numeric values.
7. Assuming browser-style HTTP helpers such as `response.json()`.

## Working Rules

Before finishing a PocketBase JavaScript change, verify that the code:

- uses ES5-compatible syntax
- avoids Node/browser-only APIs
- loads modules in a Goja-safe way
- validates inputs explicitly
- writes normalized field values
- keeps route registration simple and debuggable

If a task touches PocketBase hooks or migrations and any of these assumptions are unclear, reopen this reference before editing.
