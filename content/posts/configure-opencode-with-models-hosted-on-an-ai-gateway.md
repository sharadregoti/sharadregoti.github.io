---
title: 'How to Configure OpenCode with Models Hosted on an AI Gateway'
date: 2026-05-15T11:45:00+05:30
draft: false
---

If you already use OpenCode but your models are exposed through an AI Gateway instead of the model provider's public endpoint, the setup is straightforward once you know where OpenCode expects the provider URL to stop.

This post walks through configuring OpenCode to use Anthropic-hosted models through an AI Gateway, the mistakes that commonly cause `Not Found` errors, and the small details that make the setup reliable.

## The goal

We want OpenCode to:

- use a model served through an AI Gateway
- keep credentials out of the checked-in config
- work for both the primary model and the small model OpenCode uses internally

In my case, the gateway exposed Anthropic models. The same pattern applies to any similar setup where OpenCode talks to a provider-compatible API through a proxy or gateway.

## What OpenCode needs

There are three parts to get right:

1. Credentials
2. The model names OpenCode should use
3. The provider `baseURL`

OpenCode stores provider credentials separately from `opencode.json` or `opencode.jsonc`, so you do not need to hardcode API tokens into your config file.

## Step 1: Export your gateway credentials

If your gateway expects Anthropic-compatible auth, export the token in your shell:

```bash
export ANTHROPIC_AUTH_TOKEN="your-token"
export ANTHROPIC_BASE_URL="https://gateway.example.com/anthropic"
```

You can also authenticate via OpenCode's provider flow instead of relying only on environment variables.

## Step 2: Update your OpenCode config

OpenCode's global config lives at:

```text
~/.config/opencode/opencode.jsonc
```

Here is a working example:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "anthropic/claude-sonnet-4-6",
  "small_model": "anthropic/claude-sonnet-4-6",
  "provider": {
    "anthropic": {
      "options": {
        "baseURL": "https://gateway.example.com/anthropic/v1"
      }
    }
  }
}
```

There are two details worth calling out here.

### 1. Set both `model` and `small_model`

OpenCode uses a smaller model internally for tasks like title generation and other background work. If your gateway exposes only one model, point both fields to the same model.

If you do not do this, OpenCode may fail even when your main model is valid, because the internal small model is still unavailable.

### 2. Use the provider endpoint that already includes `/v1`

This is the detail that caused the real failure for me.

OpenCode's Anthropic provider appends the final API path itself. In practice, that means your configured `baseURL` should end at the provider root, usually including `/v1`.

Correct:

```text
https://gateway.example.com/anthropic/v1
```

Wrong:

```text
https://gateway.example.com/anthropic
```

Why this matters:

- OpenCode appends `/messages`
- if your base URL stops too early, the final request path becomes wrong
- the result is usually a `Not Found` error

## Step 3: Validate the resolved config

Before trying a real session, verify what OpenCode is actually loading:

```bash
opencode debug config
```

This helps confirm:

- your config file is valid
- the selected `model` and `small_model` are what you expect
- the provider `baseURL` override is being picked up

## Step 4: Run a quick smoke test

Use a minimal prompt first:

```bash
opencode run "Reply with OK and nothing else." --model anthropic/claude-sonnet-4-6
```

If everything is configured correctly, you should get:

```text
OK
```

## If you get `Not Found`

This usually means one of these:

- your gateway URL is missing `/v1`
- the model name is not available on the gateway
- `small_model` points to a model your gateway does not expose

The fastest way to debug this is:

```bash
opencode run "Reply with OK and nothing else." --model anthropic/claude-sonnet-4-6 --print-logs --log-level DEBUG
```

That will show the actual request URL OpenCode is trying to hit.

If you see it calling something like:

```text
https://gateway.example.com/anthropic/messages
```

instead of:

```text
https://gateway.example.com/anthropic/v1/messages
```

then the `baseURL` is the problem.

## A safer way to handle secrets

Keep tokens out of version-controlled config.

Good options:

- environment variables in your shell profile
- OpenCode's stored provider credentials
- a local-only config file that is not committed

Avoid pasting real gateway tokens directly into a checked-in `opencode.jsonc`.

## Final checklist

When configuring OpenCode with models behind an AI Gateway, make sure you have all of these in place:

- valid provider credentials
- the correct provider-prefixed model name, such as `anthropic/claude-sonnet-4-6`
- `small_model` set to a model that actually exists on the gateway
- a `baseURL` that points to the provider root and includes `/v1`
- a quick `opencode debug config` and `opencode run` smoke test

## Closing thoughts

The setup itself is small. The tricky part is that the error message is often just `Not Found`, which does not immediately tell you whether the problem is the model name, the base URL, or the internal small model.

Once I treated it as a request-shape problem instead of an authentication problem, the fix was simple: point OpenCode at the correct provider root, make sure `/v1` is included, and keep `small_model` aligned with what the gateway actually serves.

That is it for this post. If you are routing models through a gateway for governance, observability, or centralized access, OpenCode can work with that setup cleanly.
