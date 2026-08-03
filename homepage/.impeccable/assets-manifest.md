# Homepage asset manifest

## `src/assets/miura-deploy-hero.webp`

- **Role:** Build-critical hero illustration for the approved “Deployable Hiring System” / single-pull panorama direction.
- **Medium:** AI-generated editorial product still life (built-in ImageGen), then locally encoded as lossy WebP.
- **Dimensions:** 1800 × 1200 px (3:2 landscape), RGB, VP8 WebP, approximately 56 KB.
- **Composition:** One horizontal Miura-ori sheet opens from a small blank brushed-gold packet. Alternating matte cool-white and deep-green facets carry three connected, unlabeled proof zones: candidate journey, governance/evidence, and open architecture.
- **Crop notes:** Keep the complete packet and terminal right fold visible. Preserve the generous matte ground around the object so it can blend into the page; use `object-fit: contain`, not a cover crop. On narrow layouts, scale the whole asset below the copy rather than cropping any proof zone.
- **Theme notes:** The cool-white ground is authored for the light hero. Any dark-theme integration, contrast treatment, or alternate surface belongs in CSS/semantic layout; do not recolor this raster in a way that obscures the paper facets.
- **Prompt metadata:** The built-in generator returned no embedded prompt metadata. The generation spec was: a premium photorealistic paper-engineering still life, preserving comp A’s single horizontal Miura deployment from a restrained gold packet; matte cool-white ground; deep institutional green panels; precise abstract candidate-journey nodes, governance shield/evidence traces, and modular open-architecture cubes; soft upper-left daylight; no text, labels, numbers, logos, metrics, controls, people, hands, watermark, or full-page UI. A targeted edit removed the generator’s incidental packet mark while preserving the rest of the image.
- **Direction source:** the approved single-pull panorama recorded in `.impeccable/surface-brief.md`; the production asset above is the durable visual source of truth.

## What stays semantic code

- Product name, headline, body copy, navigation, locale/theme controls, links, and CTAs.
- All factual labels and explanations for hiring workflow, governed AI, and open architecture.
- Interactive states, focus treatment, responsive behavior, entry motion, and reduced-motion handling.
- Gold fold-tab CTA geometry, page-scale crease lines, section diagrams, and any theme-specific surface treatment.
- Accessibility text and any honest demonstration-data labels.

## `src/assets/gold-foil.webp`

- **Role:** Reusable restrained gold material for fold-tab CTAs and other small, explicitly approved foil accents.
- **Medium:** AI-generated macro material scan (built-in ImageGen), centrally cropped, gently desaturated, and locally encoded as lossy WebP.
- **Dimensions:** 768 × 256 px (3:1 landscape), RGB, VP8 WebP, approximately 52 KB.
- **Crop notes:** Designed for arbitrary horizontal cropping and CSS `clip-path` use. Keep `background-size: cover` at small component scale; there is no required focal point, border, shadow, or embedded shape. Avoid stretching it into large section backgrounds, where the fine grain would lose its intended material scale.
- **Prompt metadata:** The built-in generator returned no embedded prompt metadata. The generation prompt requested a flat, edge-to-edge antique-gold leaf / brushed-foil material scan with restrained fine grain, uniform density, diffuse lighting, and no focal point, hotspot, vignette, gradient, text, logo, symbol, seam, border, bevel, object, shadow, watermark, or glitter.
- **Semantic boundary:** Button labels, icons, focus/hover states, clipping geometry, contrast overlays, disabled treatment, and all interaction remain HTML/CSS. This raster supplies material grain only.
