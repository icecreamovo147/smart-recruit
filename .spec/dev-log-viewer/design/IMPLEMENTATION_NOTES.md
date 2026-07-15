# Dev Log Viewer Design Reference

This directory contains the frozen design references for the local development
log viewer.

## Source priority

When references disagree, use this order:

1. `screens/` is the visual source of truth for layout and state appearance.
2. `DESIGN.md` defines shared colors, typography, spacing, and shape tokens.
3. `html/` is a structural reference for dimensions and Tailwind class intent.
4. Repository runtime facts such as service names and ports override sample data
   embedded in any design export.

The exported HTML is static prototype code. Do not copy it directly into
production, load Tailwind from a CDN, or retain remote Google-hosted placeholder
images. Implement the UI as React + TypeScript + Vite + Tailwind CSS components.

## Screen mapping

| Screenshot | HTML reference | State |
| --- | --- | --- |
| `Live Logs - Canonical 1440p.png` | `canonical-desktop.html` | Default live stream |
| `Live Logs - Compact 1366p.png` | `compact-laptop.html` | Compact laptop layout |
| `Live Logs - Selected Entry View.png` | `selected-entry.html` | Selected log and detail panel |
| `Live Logs - SSE Reconnecting Banner.png` | `sse-reconnecting.html` | SSE reconnecting |
| `Live Logs - Paused View.png` | `paused.html` | Auto-scroll paused |
| `Live Logs - High Volume & Paused.png` | `high-volume-paused.html` | Buffer pressure and dropped lines |
| `Live Logs - No Results.png` | `no-results.html` | Filters return no logs |
| `Live Logs - No Services Running (Shell).png` | `no-services.html` | No local services running |

## Runtime facts

The implementation must obtain service state from the viewer backend rather
than hard-coding prototype data. The canonical local catalog is:

| Service | Port |
| --- | ---: |
| `identity-service` | 50061 |
| `recruitment-service` | 50062 |
| `interview-service` | 50063 |
| `offer-service` | 50064 |
| `notification-service` | 50065 |
| `ai-agent-service` | 50066 |
| `analytics-service` | 50067 |
| `worker-service` | 50068 |
| `smart-recruit-gateway` | 8080 |
| `hr-frontend` | 5173 |
| `user-frontend` | 5174 |
| `interviewer-frontend` | 5175 |

The viewer is local-only, reads allowlisted files under `.dev/logs`, and should
bind to `127.0.0.1`.

## Asset guidance

- Use local or packaged fonts where required by the implementation policy;
  the target typefaces are Inter and JetBrains Mono.
- Replace Material Symbols and remote placeholder assets with the selected
  React icon library or local SVG assets.
- Log text must be HTML-escaped. ANSI sequences should be parsed or stripped,
  never injected as raw HTML.
