# FormForge 1.6 Local Agent Runtime

FormForge Agent runs inside the FormForge server and does not open or redirect to ChatGPT. It keeps its own task queue, tool history, user-controlled long-term memories, approved sources, research drafts, and audit records. It can call an OpenAI-compatible model endpoint—normally Ollama or llama.cpp on the same Raspberry Pi—and may use a separately configured self-hosted search endpoint such as SearXNG.

## Knowledge system

The Forge Knowledge Vault contains 60 locally bundled, original-language reference modules covering hypertrophy, strength, programming, technique, nutrition, supplements, recovery, pain and injury, cardio, body composition, special populations, athletic performance, mobility, calisthenics, powerlifting, bodybuilding, behavior, evidence literacy, and safety. For each request, FormForge retrieves relevant modules and combines them with the user’s profile, workout and nutrition history, recovery and health data, pain flags, Coaching Team preferences, approved creator sources, and user-controlled memories. Optional web research can add current public sources.

This design makes the agent broad and highly personalized, but no model is literally omniscient. Medical, diagnostic, eating-disorder, and performance-enhancing-drug safety boundaries remain active.

## Sources, links, and quotations

When the user requests evidence, sources, or links, the agent returns the actual approved or researched URLs it used. Public research is tagged separately from local general knowledge. Exact quotations are allowed only when the quotation was administrator-verified or is visibly present in fetched source text; otherwise the agent paraphrases and says that it does not have a verified quote. It must never invent an author, paper, URL, quotation, creator endorsement, or licensing status.

## Autonomous tools and safeguards

The agent can create and process local research tasks, query the configured search service, fetch bounded public pages, synthesize findings, and store auditable steps. Public-page fetching rejects localhost, loopback, link-local, and private-network result URLs to reduce SSRF risk. Creator research remains a draft until an administrator reviews and publishes it. Security changes, purchases, external messages, destructive operations, and Official creator status remain approval-gated.

## Raspberry Pi recommendation

For the first users, use a Raspberry Pi 5 with 8 GB or more RAM, active cooling, and SSD storage. Small quantized text models are appropriate; large vision/video models should run on stronger hardware or a separate GPU server. The Pi package includes a Linux ARM64 server binary, start script, and sample systemd unit.
