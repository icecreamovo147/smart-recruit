---
name: Smart Recruit Homepage
description: A deployable hiring system that unfolds talent operations, governed AI, and open architecture as one connected structure.
colors:
  paper: "#f5f4ef"
  paper-raised: "#fbfaf6"
  paper-fold: "#ecece7"
  ink: "#17231d"
  ink-soft: "#526059"
  ink-faint: "#78847e"
  institutional-green: "#175b42"
  institutional-green-strong: "#0f4934"
  institutional-green-soft: "#dfe9e2"
  deployment-gold: "#c8a24d"
  deployment-gold-strong: "#aa812d"
  deployment-gold-soft: "#eee2c4"
  crease-blue: "#bdd0d7"
  rule: "#cfd2cc"
  code-surface: "#111713"
  code-text: "#e7ece8"
typography:
  display:
    fontFamily: "Chakra Petch, sans-serif"
    fontSize: "clamp(70px, 7vw, 108px)"
    fontWeight: 400
    lineHeight: 0.82
    letterSpacing: "-0.025em"
  headline:
    fontFamily: "Chakra Petch, sans-serif"
    fontSize: "clamp(48px, 5vw, 76px)"
    fontWeight: 400
    lineHeight: 0.96
    letterSpacing: "-0.025em"
  title:
    fontFamily: "Helvetica Neue, Helvetica, Arial, PingFang SC, sans-serif"
    fontSize: "23px"
    fontWeight: 600
    lineHeight: 1.2
  body:
    fontFamily: "Helvetica Neue, Helvetica, Arial, PingFang SC, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.78
  label:
    fontFamily: "JetBrains Mono, monospace"
    fontSize: "9px"
    fontWeight: 400
    lineHeight: 1.4
    letterSpacing: "0.08em"
spacing:
  control-gap: "8px"
  compact: "12px"
  component: "24px"
  container-gutter: "28px"
  section: "120px"
components:
  button-primary:
    backgroundColor: "{colors.deployment-gold}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    padding: "0 24px"
    height: "50px"
  button-secondary:
    backgroundColor: "transparent"
    textColor: "{colors.institutional-green}"
    typography: "{typography.body}"
    padding: "0 24px"
    height: "50px"
  navigation:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink-soft}"
    height: "72px"
  code-panel:
    backgroundColor: "{colors.code-surface}"
    textColor: "{colors.code-text}"
    typography: "{typography.label}"
    padding: "26px 28px 32px"
---

# Design System: Smart Recruit Homepage

## Overview

**Creative North Star: "Deployable Hiring System"**

Smart Recruit feels like a precisely engineered system unfolding from a compact deployment packet. Miura-fold geometry makes recruiting workflow, governed AI, and open architecture read as connected layers rather than isolated feature cards. Matte paper keeps the world calm and credible; institutional green carries trust, crease blue explains construction, and gold marks the moment of deployment.

The system is professional, factual, and materially restrained. It serves talent teams and developers at the same altitude: the first see operational clarity and governance, while the second see modularity, typed structure, and a credible path into the repository.

**Key Characteristics:**

- Engineered fold geometry instead of rounded SaaS containers.
- One dominant physical system object, supported by semantic diagrams and real copy.
- Quiet matte surfaces with rare, tactile gold actions.
- Angular display type paired with neutral bilingual body text and precise mono labels.
- Light and dark themes retain the same material hierarchy.

## Colors

The palette behaves like technical paper engineering: warm matte neutrals, deep institutional green, pale construction blue, and scarce deployment gold.

### Primary

- **Institutional Green:** The trust and system color for headings, outlines, active structure, and governed states.
- **Deployment Gold:** Reserved for primary actions, sequence markers, and small moments that initiate or confirm deployment.

### Secondary

- **Crease Blue:** Fine construction lines and background folds; it explains geometry without becoming a general accent.

### Neutral

- **Matte Paper:** The default light-theme field.
- **Raised Paper:** Local contrast for fold panels and navigation surfaces.
- **Carbon Paper:** The dark-theme field and code-adjacent surface.
- **Botanical Ink:** Primary text; softened and faint variants establish supporting hierarchy.
- **Drafting Rule:** Borders, dividers, and assembly seams.

**The Scarce Foil Rule.** Gold is an action material, not a page wash; keep it rare enough that a gold control always signals consequence.

**The Construction-Line Rule.** Crease blue may guide structure but must never compete with content or behave like a second brand color.

## Typography

**Display Font:** Chakra Petch (with sans-serif fallback)

**Body Font:** Helvetica Neue / Helvetica / Arial with Chinese system sans fallbacks
**Label/Mono Font:** JetBrains Mono

**Character:** Chakra Petch supplies angular, engineered silhouettes for the deployable-system metaphor. Neutral system sans keeps bilingual paragraphs readable; JetBrains Mono turns labels, indices, and code into measured technical evidence.

### Hierarchy

- **Display** (400, fluid 70–108px, 0.82): Product name and singular large statements; mobile scales to 62–82px.
- **Headline** (400, fluid 48–76px, 0.96): Section-level framing with compact line breaks.
- **Title** (600, 23px): Capability and role names.
- **Body** (400, 15px, 1.78): Explanations constrained to roughly 50–58 characters per line.
- **Label** (400, 9px, 0.08em tracking): Uppercase system labels, role tags, fold identifiers, and sequence indices.

**The Three-Voice Rule.** Chakra Petch names the system, the neutral sans explains it, and JetBrains Mono annotates it; do not swap their roles casually.

## Layout

The desktop shell uses a centered 1240px container with a 28px minimum gutter and a sticky 72px header. The hero is deliberately asymmetric: a compact copy block on the left and a wider Miura panorama on the right. Subsequent sections use unequal two-column grids so editorial explanation and system proof remain distinct.

Sections generally use 120px vertical rhythm. At 860px, navigation and primary grids collapse; at 560px, gutters tighten to 14px, actions become full-width, role rows simplify, and fold assemblies stack vertically. Mobile keeps the full system story: the hero image moves below the copy and retains a compact three-layer legend.

**The Connected-Preview Rule.** A major surface should visibly lead into the next system layer; avoid terminating a viewport with an isolated decorative band.

## Elevation & Depth

Depth is structural and sparse. Paper sheets use broad, soft ambient shadows; gold actions use a smaller warm shadow. Most hierarchy comes from tonal layers, overlap, fold direction, and thin rules rather than card elevation.

### Shadow Vocabulary

- **Paper Lift:** A broad low-contrast shadow for the Miura panorama and code panel.
- **Foil Lift:** A restrained warm shadow for primary and closing actions.

**The Physical-Cause Rule.** A shadow must correspond to a sheet, packet, or interactive surface that could physically lift; flat content rows stay flat.

## Shapes

The system avoids generic rounded rectangles. Buttons, code panels, fold panels, and architecture cells use clipped polygons with diagonal cuts. Hairline rules behave like assembly seams. The recurring silhouette is a sheet under tension: edges taper, panels overlap, and one direction of travel remains obvious.

**The Fold-Not-Decoration Rule.** A diagonal edge must explain direction, layering, or deployment; never scatter unrelated polygons as ornament.

## Components

### Buttons

- **Shape:** A rectangular tab with an 18px directional cut on the trailing edge.
- **Primary:** Gold foil raster over the deployment-gold base, dark text, 50px minimum height, and 24px horizontal padding.
- **Secondary:** Transparent with an institutional-green rule; hover adds the soft green surface.
- **Hover / Focus:** A 2px lift for primary actions, restrained color shift, and a visible 2px gold focus outline with 4px offset.

### Cards / Containers

- **Corner Style:** No radii; clipped fold polygons define the perimeter.
- **Background:** Raised paper for passive sheets, strong green for a selected or central fold.
- **Shadow Strategy:** Flat by default; use Paper Lift only for real overlapping assemblies.
- **Border:** One-pixel drafting rules on structural edges.
- **Internal Padding:** Typically 24–30px.

### Navigation

The sticky desktop header uses a translucent paper surface, quiet 13px links, compact square utilities, and a fold-cut GitHub action. Mobile preserves the brand and complete destination inventory in a full-width ruled list. Hover changes color; focus remains visibly gold.

### Fold Assemblies

Miura panoramas, evidence folds, and architecture stacks are the signature components. They combine overlap, alternating surface color, explicit labels, and directional clip paths. Only these structural assemblies receive fold-based reveal motion.

### Code Panel

The quick-start panel is a clipped carbon surface with mono text, a ruled toolbar, and an explicit copy action. It is evidence of deployability, not a decorative terminal motif.

## Do's and Don'ts

### Do:

- **Do** show talent workflow, governed AI, and open architecture as one connected system.
- **Do** use gold foil for high-intent deployment actions and visible focus treatment.
- **Do** keep bilingual body copy in the neutral sans and technical labels in JetBrains Mono.
- **Do** preserve sharp, legible fold imagery and all three system labels on mobile.
- **Do** honor reduced motion by removing deployment and clip transitions.

### Don't:

- **Don't** replace fold assemblies with generic rounded cards, pills, or dashboard chrome.
- **Don't** invent metrics, customers, UI screenshots, or AI claims that the product cannot substantiate.
- **Don't** apply repeated fade-and-rise reveals across ordinary sections.
- **Don't** use gold, crease blue, or polygon cuts as free-floating decoration.
- **Don't** hide developer evidence or talent-team workflows to simplify the page for one audience.
