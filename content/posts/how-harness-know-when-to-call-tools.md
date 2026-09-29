---
title: "How Does an AI Harness Know When to Call a Tool?"
date: 2026-09-29T21:50:00+05:30
draft: false
description: "LLMs are stateless and can't reach the outside world. Here's how the harness spots a tool call in the model's output, runs it, and feeds the result back: the agent loop, explained from first principles."
cover:
  image: "/images/how-harness-know-when-to-call-tools/cover.png"
  alt: "How an AI harness knows when to call a tool"
  relative: false
---

I have been reading a lot about LLMs, MCP, Harnesses, etc., all the new AI stuff. 

I learn from first principles and try to nail down the basics. So far, one thing I understand about LLMs: they are stateless. You provide input tokens (your message), and they provide output tokens; it is like a black box. It can't talk to the outside world on its own; that capability is added in a harness. If an LLM is the brain, the harness is like the body.

What I failed to understand: how does this piece of software (the harness) know when to call a tool, e.g., web_fetch, to fetch an external URL?

It turns out it is just a protocol that both the harness and the LLM already agreed on. The communication happens between them in plain text. The harness monitors a special set of characters ("reserved tokens") and, when it sees that a model wants to perform a tool call, it performs the call and appends the result back as another input message.

Let's understand using the `web_fetch` example.

![The tool call loop between the harness and the LLM](/images/how-harness-know-when-to-call-tools/tool_call_harness_loop.png)

1. When you send a query to the LLM, "Install stuff from this GitHub repo," along with this input message, the harness adds the system prompt that includes information about the available tools.

```
Tools available:
  web_fetch: "Fetch the contents of a web page"
  params: { "url": string (required) }

User: What's on the front page of example.com?
---
more tools...
```
2. When the LLM processes your input message, it knows that it has access to tools because Harness made them available to it in the initial message. Now, the LLM is pretrained on this, so if it decides it needs external information from the URL to answer the query, it outputs a set of tokens in a pre-agreed format. The LLM outputs those special tokens in a specific format; for simplicity, this is an XML representation.

```
Assistant: Let me check.
<tool_call>
  {"name": "web_fetch", "arguments": {"url": "https://example.com"}}
</tool_call>
```

3. The harness keeps reading token streams, checks for this specific syntax, and when it finds it, stops the stream, executes the tool call.

4. The result is pasted back, and the model is called again. The harness appends the fetched page as a new message and sends the entire conversation back:

```
...previous stuff...
<tool_result name="web_fetch">
  Example Domain. This domain is for use in illustrative examples...
</tool_result>
```

The model now "sees" the page as if someone typed it in, and continues writing. It might call another tool or give the final answer. That repetition is the agent loop.
