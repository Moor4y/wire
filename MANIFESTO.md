# The Headless Orchestration Manifesto: Building for Machines, Not Humans

The existing automation stack is broken. It was architected for a world that no longer exists—a world where humans click drag-and-drop boxes to patch static APIs. 

Tools like Zapier, n8n, and Clay are spectacular for non-technical managers wiring marketing tools together. But when forced into the modern AI agent architecture, they fail catastrophically. They are slow, restrictively licensed, heavily bloated, and structurally optimized for human eyes rather than programmatic machine intelligence.

We are building a new primitive. This engine is a headless, open-source, deterministic state-machine built from the ground up to be consumed, modified, and executed exclusively by AI agents.

---

## 1. What is Wrong with the Status Quo?

### The Tax on Execution (The Zapier Problem)
Traditional automation platforms charge you a micro-tariff every single time a node triggers. In an autonomous loop, an AI agent might run 50 iterative tool calls in 30 seconds to self-correct, research a data schema, or negotiate an output. Under a legacy "pay-per-click" model, running true autonomous loops will bankrupt your business. **Computation overhead must scale with hardware capacity, not vendor pricing models.**

### The Visual Illusion (The n8n Problem)
n8n popularized the visual node-graph canvas. This canvas is an abstraction designed to help human brains visualize state logic. But AI agents do not have eyes, and they do not use a computer mouse. Forcing an AI agent to build workflows by programmatically calculating arbitrary $X/Y$ coordinates on a visual UI grid is an engineering farce. **Agents need to read and write code and declarative, structured text (YAML)—not manipulate geometric boxes.**

### The Commercial Lockdown
The moment you want to embed an automation engine inside a micro-SaaS application, white-label it for clients, or distribute it cleanly, you run headfirst into licensing paywalls. "Source-available" or "Fair-code" models lock you down when you scale. **The core plumbing of the internet's agentic data routing layer must be completely unrestrictive (MIT/Apache 2.0).**

---

## 2. Core Principles of Machine-Native Orchestration

Our engine is built on four non-negotiable architectural dogmas:

### I. Separating Intelligence from Execution
Large Language Models are excellent at reasoning, planning, and tool discovery. They are notoriously unreliable at running predictable, long-running sequential logic over hours or days without experiencing context degradation or hallucinations. 
* **The Paradigm:** An AI agent should use its intelligence *exactly once* to reason through a problem, generate a structured, declarative execution blueprint (a YAML file), and hand it off to a dumb, hyper-fast, deterministic engine to run safely.

### II. Dual-Mode Model Context Protocol (MCP) as a First-Class Citizen
We reject the burden of maintaining 500+ hardcoded, shifting REST API wrappers. We treat Anthropic's **Model Context Protocol (MCP)** as our native network interface. 
* **Engine as an MCP Server:** The orchestrator exposes its entire directory of automated workflows as standard executable tools that an LLM can dynamically discover and trigger.
* **Engine as an MCP Client:** Every workflow step simply acts as an execution bridge to local or remote MCP servers. 

### III. Headless Self-Modification
Because workflows are defined as raw, text-based YAML, an agent can dynamically rewrite its own instruction manual mid-execution. If a step returns an unexpected error schema, the agent can intercept the failure log, edit the YAML workflow file to handle the mutation, and instruct the orchestrator to safely resume execution from the exact state of failure.

### IV. Compilation into a Single System Binary
We reject Node.js bloat, massive server clusters, heavy message brokers, and front-end UI dependencies. This engine is written in Go. It compiles down to a single, statically-linked 20MB binary file. It boots up in under 5 milliseconds, uses less than 15MB of idle RAM, and stores its state securely inside a local SQLite database configured with Write-Ahead Logging (WAL). It can be deployed identically on an edge router, an air-gapped enterprise local network, or a massive cloud node.

---

## 3. Who is This For?

* **AI Infrastructure Architects:** Who need an open-source, bulletproof execution harness to offload token-heavy loops from their agent frameworks (LangChain, CrewAI, Autogen).
* **Indie Hackers & Founders:** Who want a permissive, legally transparent automation layer they can modify, fork, rename, and seamlessly embed directly into their proprietary commercial software products.
* **SecOps & Enterprise Security Teams:** Who are completely locked out of cloud-hosted AI tools because their data governance policies strictly forbid sending internal infrastructure logs to public clouds. They need an air-gapped binary that runs locally.

---

## 4. Join the Shift

We are not building a tool for humans to drag lines between colored boxes. We are building the invisible, industrial-grade copper plumbing that automated software factories will use to run the world.

If you believe that software architecture should be optimized for programmatic machine execution, low-latency determinism, and zero licensing friction—you belong here.

* **Read the Specs:** Check out `/docs/SPEC.md`
* **Run Local:** `go run cmd/orchestrator/main.go --workflow ./examples/demo.yaml`

Let's build the headless future. Permissively.