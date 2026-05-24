# AI Module

Interact with large language models and AI services. Provides interfaces for text completion, chat, embeddings, classification, and content generation.

**Use case:** Use for natural language processing, chatbots, content analysis, and semantic search.

---

## Import

```lunex
val ai = @import("std.ai")
```

---

## Available Functions

### `complete(prompt, options)`

Executes the `complete` operation with the given parameters (prompt, options).

**Signature:**
```lunex
fn complete(prompt, options)
```

### `chat(messages, options)`

Executes the `chat` operation with the given parameters (messages, options).

**Signature:**
```lunex
fn chat(messages, options)
```

### `embed(text)`

Executes the `embed` operation with the given parameter (text).

**Signature:**
```lunex
fn embed(text)
```

### `classify(text, labels)`

Executes the `classify` operation with the given parameters (text, labels).

**Signature:**
```lunex
fn classify(text, labels)
```

### `moderate(text)`

Executes the `moderate` operation with the given parameter (text).

**Signature:**
```lunex
fn moderate(text)
```

### `similarity(a, b)`

Executes the `similarity` operation with the given parameters (a, b).

**Signature:**
```lunex
fn similarity(a, b)
```

### `create(options)`

Executes the `create` operation with the given parameter (options).

**Signature:**
```lunex
fn create(options)
```

### `summarize(text, options)`

Executes the `summarize` operation with the given parameters (text, options).

**Signature:**
```lunex
fn summarize(text, options)
```

### `translate(text, targetLang, options)`

Executes the `translate` operation with the given parameters (text, targetLang, options).

**Signature:**
```lunex
fn translate(text, targetLang, options)
```

### `sentiment(text)`

Executes the `sentiment` operation with the given parameter (text).

**Signature:**
```lunex
fn sentiment(text)
```

### `extract(text, schema, options)`

Executes the `extract` operation with the given parameters (text, schema, options).

**Signature:**
```lunex
fn extract(text, schema, options)
```

### `createAssistant(systemPrompt, options)`

Executes the `createAssistant` operation with the given parameters (systemPrompt, options).

**Signature:**
```lunex
fn createAssistant(systemPrompt, options)
```

### `ask(userMessage)`

Executes the `ask` operation with the given parameter (userMessage).

**Signature:**
```lunex
fn ask(userMessage)
```

### `reset()`

Executes the `reset` operation with the given no arguments.

**Signature:**
```lunex
fn reset()
```

### `getHistory()`

Executes the `getHistory` operation with the given no arguments.

**Signature:**
```lunex
fn getHistory()
```

### `findMostSimilar(query, documents)`

Executes the `findMostSimilar` operation with the given parameters (query, documents).

**Signature:**
```lunex
fn findMostSimilar(query, documents)
```

