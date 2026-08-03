# Design QA

- Source visual truth: `/private/var/folders/2w/66vv37qj5rn6t6xjc0nvj_sm0000gn/T/codex-clipboard-tlKsUc.png`
- Source dimensions: `1526 × 424` pixels
- Implementation target: `hr-frontend/src/components/chat/ChatComposer.vue`
- Pre-fix implementation screenshot: `/private/var/folders/2w/66vv37qj5rn6t6xjc0nvj_sm0000gn/T/codex-clipboard-rRM43s.png`
- Pre-fix implementation dimensions: `1780 × 580` pixels
- Side-by-side normalized comparison: `/private/tmp/skill-menu-comparison.png`
- Post-fix implementation screenshot: unavailable
- Intended state: desktop, light theme, `/` Skill menu open
- Intended route: `/hr/ai`
- Viewport and density: both inputs are user-provided cropped screenshots with unknown device scale factors; each was proportionally normalized to `890px` width and padded to `290px` height for the side-by-side comparison

## Full-view comparison evidence

The source reference was opened and inspected. It shows a rounded command panel with compact rows composed of a cube icon, bold Skill name, one-line description, and a muted source label aligned to the right. The first row uses a soft neutral highlight.

The pre-fix implementation screenshot was opened beside the source in one normalized comparison image. Its row structure, neutral highlight, icon, name, description, and source placement follow the reference. User follow-up clarified that the full composer width is intentional; only the individual Skill row density should be reduced.

The compact-row fix has been implemented while preserving the full-width menu, but it cannot be captured in the required browser surface because no in-app browser or other browser backend is available in the current environment. A code or build inspection is not a valid substitute for post-fix rendered evidence.

## Focused region comparison evidence

The side-by-side comparison focuses on the Skill menu and its surrounding composer context. User feedback established row density and repeated name/description as the intended fixes while retaining the current menu width. Post-fix focused evidence remains blocked pending a new screenshot.

## Findings

- [P1] Post-fix rendered visual fidelity cannot be verified
  - Location: HR AI chat `/` Skill menu.
  - Evidence: the implementation now preserves full width, reduces row height, and suppresses duplicate descriptions, but no revised screenshot is available.
  - Impact: the corrected vertical density and responsive rendering cannot yet be confirmed visually.
  - Fix: open `/hr/ai`, enter `/`, capture the revised Skill menu at the same desktop/light state, and compare it with the source.

## Required fidelity surfaces

- Fonts and typography: pre-fix hierarchy follows the reference; post-fix size and truncation remain blocked pending capture.
- Spacing and layout rhythm: the menu remains full width by product decision; code now uses a `42px` minimum row height with reduced internal padding, pending rendered confirmation.
- Colors and visual tokens: pre-fix neutral highlight and surface treatment are aligned with the reference; post-fix remains pending capture.
- Image and icon fidelity: the implementation uses the existing Element Plus `Box` icon; its pre-fix rendered weight is slightly heavier than the source, and the revised smaller size remains pending capture.
- Copy and content: Skill name and `smart-recruit` source are correct; duplicated description text is now omitted.

## Comparison history

- Initial pass: source opened; implementation capture blocked because no browser backend is available.
- User evidence pass: received and opened the `1780 × 580` implementation screenshot.
- Comparison pass: normalized source and implementation into `/private/tmp/skill-menu-comparison.png`; identified row density and duplicated name/description for follow-up.
- User correction: confirmed the menu must remain full width.
- Fix iteration: restored full-width layout, reduced panel padding/radius/shadow, reduced row height from `52px` to `42px`, reduced icon and type sizes, and hid descriptions identical to the Skill name.
- Post-fix pass: blocked because no revised screenshot or browser backend is available.

## Implementation checklist

- Capture the revised open Skill menu in the HR AI chat page.
- Confirm the full-width menu and compact row density in desktop and narrow layouts.
- Compare the revised crop against the source and close any remaining P0/P1/P2 differences.

final result: blocked
