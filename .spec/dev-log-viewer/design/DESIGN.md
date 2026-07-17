---
name: Kinetic Terminal
colors:
  surface: '#10131a'
  surface-dim: '#10131a'
  surface-bright: '#363941'
  surface-container-lowest: '#0b0e15'
  surface-container-low: '#191b23'
  surface-container: '#1d2027'
  surface-container-high: '#272a31'
  surface-container-highest: '#32353c'
  on-surface: '#e1e2ec'
  on-surface-variant: '#c2c6d6'
  inverse-surface: '#e1e2ec'
  inverse-on-surface: '#2e3038'
  outline: '#8c909f'
  outline-variant: '#424754'
  surface-tint: '#adc6ff'
  primary: '#adc6ff'
  on-primary: '#002e6a'
  primary-container: '#4d8eff'
  on-primary-container: '#00285d'
  inverse-primary: '#005ac2'
  secondary: '#4ae176'
  on-secondary: '#003915'
  secondary-container: '#00b954'
  on-secondary-container: '#004119'
  tertiary: '#ffb786'
  on-tertiary: '#502400'
  tertiary-container: '#df7412'
  on-tertiary-container: '#461f00'
  error: '#ffb4ab'
  on-error: '#690005'
  error-container: '#93000a'
  on-error-container: '#ffdad6'
  primary-fixed: '#d8e2ff'
  primary-fixed-dim: '#adc6ff'
  on-primary-fixed: '#001a42'
  on-primary-fixed-variant: '#004395'
  secondary-fixed: '#6bff8f'
  secondary-fixed-dim: '#4ae176'
  on-secondary-fixed: '#002109'
  on-secondary-fixed-variant: '#005321'
  tertiary-fixed: '#ffdcc6'
  tertiary-fixed-dim: '#ffb786'
  on-tertiary-fixed: '#311400'
  on-tertiary-fixed-variant: '#723600'
  background: '#10131a'
  on-background: '#e1e2ec'
  surface-variant: '#32353c'
typography:
  headline-lg:
    fontFamily: Inter
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
    letterSpacing: -0.01em
  headline-md:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: '600'
    lineHeight: 24px
    letterSpacing: -0.01em
  body-md:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  body-sm:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 18px
  code-md:
    fontFamily: JetBrains Mono
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 20px
  code-sm:
    fontFamily: JetBrains Mono
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 16px
  label-xs:
    fontFamily: Inter
    fontSize: 11px
    fontWeight: '700'
    lineHeight: 16px
    letterSpacing: 0.05em
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  base: 4px
  xs: 4px
  sm: 8px
  md: 12px
  lg: 16px
  xl: 24px
  gutter: 8px
  sidebar_width: 240px
---

## Brand & Style

The design system is engineered for high-density information environments where speed of diagnosis and precision are paramount. It targets software engineers and DevOps professionals who require a UI that stays out of the way while surfacing critical data points through structural hierarchy rather than decorative flourishes.

The style is **Professional Minimalist** with a **Technical** lean. It draws inspiration from IDEs and monitoring dashboards, prioritizing utility and data density. The aesthetic is defined by flat surfaces, rigid grid alignments, and a strict "function over form" philosophy. Visual noise is aggressively reduced by eliminating gradients, shadows, and unnecessary transitions, ensuring that the user's cognitive load is reserved entirely for the data being analyzed.

## Colors

This design system utilizes a deep charcoal and slate palette to minimize eye strain during long-form technical analysis. The color architecture is strictly functional:

- **Neutral Scale:** Backgrounds use a tiered monochromatic approach to establish depth without shadows. The base (#0f1115) is for the workspace, while surfaces and elevated areas distinguish panels and modals.
- **Status Colors:** These are the primary vehicles for communication. They are high-chroma to ensure immediate recognition against the dark background.
- **Action Colors:** Primary blue is reserved for meaningful interactions and focus states.
- **Borders:** Low-contrast borders (#2d333b) are used instead of shadows to define containers, maintaining a flat, architectural feel.

## Typography

Typography is split into two distinct functional roles:
1.  **Interface (Inter):** Used for navigation, labels, and UI controls. It is optimized for legibility at small sizes.
2.  **Data (JetBrains Mono):** Used for log entries, code snippets, and technical metadata. The monospaced nature ensures vertical alignment of log timestamps and levels, making it easier to scan lists.

**Hierarchy Rules:**
- Use `headline-lg` sparingly for page-level titles.
- `body-sm` is the default for most interface text to maximize density.
- `code-sm` is the standard for log stream rendering.
- All uppercase `label-xs` should be used for table headers and section grouping labels.

## Layout & Spacing

This system employs a **Compact Fluid Grid** based on a 4px baseline. The goal is to maximize the "above the fold" data visibility.

- **Grid:** A 12-column system is used for dashboard layouts, while log explorers utilize a sidebar-main configuration.
- **Density:** Padding is kept to a minimum (8px default for containers). Horizontal spacing between related elements (like an icon and its label) is 4px.
- **Breakpoints:**
  - **Desktop (1440px+):** Full multi-pane view with persistent sidebars.
  - **Tablet (768px - 1439px):** Collapsible sidebars; main log stream fills the width.
  - **Mobile:** Not a primary target; use a single-column stack with simplified log views.

## Elevation & Depth

Depth is communicated through **Tonal Layering** rather than physical light simulation.

- **Level 0 (Base):** #0f1115. Used for the main application background.
- **Level 1 (Surface):** #1a1d23. Used for sidebars, activity bars, and header sections.
- **Level 2 (Elevated):** #252932. Used for active cards, code blocks, or floating tooltips.
- **Borders:** All levels are separated by a 1px solid border (#2d333b).
- **Shadows:** Avoid shadows entirely. If a modal requires separation, use a 1px border with a slightly lighter hex than the surface color to create a "glow" edge or a simple dark overlay behind the modal.

## Shapes

The design system uses a **Soft Square** language.
- **Standard Corners:** 4px (0.25rem) is the default for buttons, inputs, and panels. This provides a professional, modern feel without being as aggressive as sharp 0px corners.
- **Small Elements:** Badges and tags also follow the 4px rule.
- **Exceptions:** No pill-shaped or fully rounded elements should be used, as they waste horizontal space and conflict with the technical aesthetic.

## Components

### Buttons
- **Primary:** Solid #3b82f6 background with white text. 4px radius.
- **Secondary/Ghost:** No background, 1px border (#2d333b). On hover, background becomes #252932.
- **Size:** Compact height (28px or 32px).

### Badges & Status Tokens
- **Log Levels:** Small rectangles with a subtle 10% opacity background of the status color and a 100% opacity side-border (left, 2px) of the status color. Text is white or the status color itself.
- **Clickable Tokens:** Tags for metadata (e.g., `service:auth`) should have a #252932 background and a subtle hover state highlighting the border.

### Input Fields
- **Search/Filter:** Background #0f1115, border #2d333b. Focus state changes border to #3b82f6. Use JetBrains Mono for the text within filter inputs.

### Lists & Tables
- **Log Stream:** Each row has a height of 20px-24px. Alternate row striping is not used; instead, use a subtle #252932 background on hover to indicate row selection.
- **Vertical Alignment:** Timestamps, Levels, and Messages must be strictly aligned in columns.

### Cards
- Cards are used for high-level metrics (e.g., "Error Rate"). They should have no shadow, a #1a1d23 background, and a 1px border. Titles should be `label-xs` style.