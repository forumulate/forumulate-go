# Forumulate Go SDK

The official Go client for the [Forumulate](https://forumulate.com) API: docs, forum, blog and every other space type, plus the organization around them. Every operation of the API (v1.1.0) is a typed method on a service, grouped the way the product is: `client.Spaces`, `client.Docs.Pages`, `client.Forum.Topics`, `client.Blog.Articles`, …

- Go 1.21+, standard library only.
- Typed request bodies, query parameters and responses; `context.Context` on every call.
- Retries rate limits and server errors with backoff, honouring `Retry-After`.
- Auto-pagination (`…AutoPaging` returns an `Iter[T]`), server-sent events (`…Stream`), a typed `*forumulate.Error`.
- Pinned to API version `2026-11-01`: every request sends `Forumulate-Version: 2026-11-01`, so the response types are the shapes you receive.

## Install

```sh
go get github.com/forumulate/forumulate-go
```

## Authenticate

Create a token under **Account → Tokens** and put it in the environment:

```sh
export FORUMULATE_TOKEN=ak_...
```

```go
import "github.com/forumulate/forumulate-go"

client := forumulate.NewClient() // reads FORUMULATE_TOKEN / FORUMULATE_URL / FORUMULATE_API_URL
// or explicitly:
client = forumulate.NewClient(
	forumulate.WithToken("ak_..."),
	forumulate.WithBaseURL("https://forumulate.com/api"),
)
```

| Variable | Meaning |
| --- | --- |
| `FORUMULATE_TOKEN` | Your API token (Account → Tokens: `ak_…` or `cft_…`), sent as `Authorization: Bearer`. |
| `FORUMULATE_URL` | Your site, default `https://forumulate.com`. The API is its `/api` (a value already ending in `/api` is used as is). |
| `FORUMULATE_API_URL` | The exact API base, e.g. `http://localhost:4000`. Wins over `FORUMULATE_URL`. |

These are the names `@forumulate/sdk`, the CLI and `@forumulate/mcp` read, so one `.env` drives all of them.

Organizations and spaces take an id or a slug wherever the API asks for one. Optional fields are pointers; `forumulate.String`, `forumulate.Int`, `forumulate.Bool` and `forumulate.Float` make them.

## Quickstart: the five most common tasks

### 1. List your organizations and their spaces

```go
ctx := context.Background()
orgs := client.Orgs.ListAutoPaging(ctx, nil)
for orgs.Next() {
	org := orgs.Current()
	spaces := client.Spaces.ListAutoPaging(ctx, org.ID, nil)
	for spaces.Next() {
		s := spaces.Current()
		fmt.Println(org.Slug, s.Type, s.Slug, s.Name)
	}
	if err := spaces.Err(); err != nil {
		log.Fatal(err)
	}
}
if err := orgs.Err(); err != nil {
	log.Fatal(err)
}
```

### 2. Write a docs page and publish it

```go
created, err := client.Docs.Pages.Create(ctx, "acme", "handbook", "guides", forumulate.DocsPagesCreateParams{
	Title: "Getting started",
	Slug:  "getting-started",
	Body:  forumulate.String("# Getting started\n\nInstall the CLI, then sign in."),
})
if err != nil {
	log.Fatal(err)
}
_, err = client.Docs.Pages.Publish(ctx, "acme", "handbook", created.Page.ID, forumulate.DocsPagesPublishParams{})
```

### 3. Read and start forum topics

```go
topics := client.Forum.Topics.ListAutoPaging(ctx, "acme", "community", &forumulate.ForumTopicsListQuery{
	Sort:  forumulate.String("latest"),
	Limit: forumulate.Int(50),
})
for topics.Next() {
	fmt.Println(topics.Current().Title)
}

_, err = client.Forum.Topics.Create(ctx, "acme", "community", forumulate.ForumTopicsCreateParams{
	Title:    "How do I rotate a token?",
	Markdown: forumulate.String("I want to rotate the token my CI uses without downtime."),
	Tags:     []string{"tokens"},
})
```

### 4. Search everything in an organization

```go
results, err := client.Search.Query(ctx, "acme", &forumulate.SearchQueryQuery{
	Q:     forumulate.String("rotate token"),
	Limit: forumulate.Int(10),
})
```

### 5. Verify a webhook delivery

```go
body, _ := io.ReadAll(r.Body) // the RAW body, before json decoding
ok := forumulate.VerifyWebhookSignature(forumulate.WebhookInput{
	Secret:    os.Getenv("FORUMULATE_WEBHOOK_SECRET"),
	Body:      body,
	Signature: r.Header.Get("X-Forumulate-Signature"),
	Timestamp: r.Header.Get("X-Forumulate-Timestamp"),
})
if !ok {
	http.Error(w, "bad signature", http.StatusUnauthorized)
	return
}
// X-Forumulate-Delivery is the key to dedupe on: retries reuse it.
```

## Errors

Any answer of 400 or above is a `*forumulate.Error` with `StatusCode`, `Code` (the API's machine-readable code), `Message` and `RequestID`:

```go
_, err := client.Spaces.Get(ctx, "acme", "nope")
switch {
case forumulate.IsNotFound(err):
	// …
case forumulate.IsUnauthorized(err):
	var e *forumulate.Error
	errors.As(err, &e)
	fmt.Println(e.Code, e.Message) // auth.unauthenticated Authentication required.
}
```

`IsUnauthorized` (401), `IsForbidden` (403), `IsNotFound` (404) and `IsRateLimited` (429, after retries) cover the common cases.

## More

- **Per-request options.** Every method takes `...RequestOption`, e.g. `forumulate.WithRequestHeader("Idempotency-Key", key)`.
- **Client options.** `WithHTTPClient`, `WithMaxRetries`, `WithUserAgent`, `WithDefaultHeader`.
- **Reference.** Every method's doc comment names its route; the API reference is at <https://forumulate.com/reference>.

This package is generated from Forumulate's OpenAPI document by Forumulate's own SDK generator — the same one that builds SDKs for the APIs our customers document with us. Do not edit it by hand: a fix belongs in the generator, and lands here on the next release. The source of truth is the private monorepo; this repository is a mirror (`forumulate-go`).

## License

MIT
