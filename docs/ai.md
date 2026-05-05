# ai — AI / LLM Module

The `ai` module provides a unified interface for interacting with large language models (OpenAI, Anthropic, Gemini, Ollama, and more).

## Import

```ntl
use ai
```

---

## Configuration

### `ai.configure(options)`
Set the default provider and API key.

```ntl
ai.configure({
  provider: "openai",
  apiKey:   env.get("OPENAI_API_KEY"),
  model:    "gpt-4o",
})
```

#### Providers

| Provider | Models |
|---|---|
| `"openai"` | `gpt-4o`, `gpt-4-turbo`, `gpt-3.5-turbo` |
| `"anthropic"` | `claude-3-5-sonnet`, `claude-3-opus`, `claude-3-haiku` |
| `"gemini"` | `gemini-1.5-pro`, `gemini-1.5-flash` |
| `"ollama"` | `llama3`, `mistral`, `phi3`, `gemma2` (local) |
| `"openrouter"` | Any model via OpenRouter |

---

## Text Completion

### `ai.complete(prompt, [options])`
Generate a text completion. Returns the response string.

```ntl
val reply = ai.complete("Explain NTL scripting in one sentence.")
io.log(reply)
```

#### Options

| Option | Default | Description |
|---|---|---|
| `model` | from config | Model override |
| `temperature` | `0.7` | Randomness (0 = deterministic) |
| `maxTokens` | `1024` | Maximum response length |
| `system` | `null` | System prompt |
| `stop` | `null` | Stop sequences |

---

## Chat

### `ai.chat(messages, [options])`
Send a multi-turn conversation.

```ntl
val response = ai.chat([
  { role: "system",    content: "You are a helpful assistant." },
  { role: "user",      content: "What is 2 + 2?" },
  { role: "assistant", content: "4" },
  { role: "user",      content: "And 4 * 4?" },
])
io.log(response)    // "16"
```

---

## Embeddings

### `ai.embed(text)`
Generate a vector embedding for text. Returns an array of floats.

```ntl
val vec = ai.embed("The quick brown fox")
io.log(vec.length)    // 1536 (for OpenAI ada)
```

### `ai.similarity(vecA, vecB)`
Compute cosine similarity between two embeddings.

```ntl
val sim = ai.similarity(vec1, vec2)
io.log(sim)    // 0.0 to 1.0
```

---

## Streaming

### `ai.stream(prompt, handler, [options])`
Stream a response token by token.

```ntl
ai.stream("Write a haiku about rain.", fn(token, done) {
  io.print(token)
  if done {
    io.newline()
  }
})
```

---

## Structured Output

### `ai.json(prompt, schema, [options])`
Ask the model to produce JSON matching a schema.

```ntl
val data = ai.json(
  "Extract the name and age from: 'Alice is 30 years old'",
  { name: "string", age: "number" }
)
io.log(data.name)    // Alice
io.log(data.age)     // 30
```

---

## Image Analysis

### `ai.vision(prompt, imageUrl, [options])`
Analyze an image with a vision-capable model.

```ntl
val desc = ai.vision("What is in this image?", "https://example.com/photo.jpg")
io.log(desc)
```

---

## Example: CLI Assistant

```ntl
use ai
use io
use env

env.load()

ai.configure({
  provider: "openai",
  apiKey:   env.require("OPENAI_API_KEY"),
  model:    "gpt-4o",
})

val history = [
  { role: "system", content: "You are a helpful NTL programming assistant." },
]

fn ask(question) {
  history.push({ role: "user", content: question })
  val reply = ai.chat(history)
  history.push({ role: "assistant", content: reply })
  return reply
}

fn main() {
  io.banner("NTL AI Assistant")
  io.log(io.gray("Type 'quit' to exit"))
  io.newline()

  loop {
    val input = io.read(io.cyan("You: "))
    if input == "quit" { break }
    val reply = ask(input)
    io.log(io.green("AI:"), reply)
    io.newline()
  }
}
```
