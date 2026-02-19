---
name: ascii-diagram
description: Use when asked to create diagrams, flowcharts, tables, org charts, timelines, or any structured visual using ASCII/Unicode characters. Triggered by "ascii diagram", "mach mir ein diagramm", "flowchart", "zeichne mir", or when a visual diagram would help explain something.
user-invocable: true
argument-hint: "[description of diagram to create]"
---

# ASCII Diagram Skill

Create structured diagrams using text characters — flowcharts, decision trees, tables, timelines, org charts, and more.

## Diagram Types

### Flowcharts

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│  Start   │────►│ Process  │────►│   End    │
└──────────┘     └──────────┘     └──────────┘
                      │
                      ▼
                 ┌──────────┐
                 │ Decision │
                 └────┬─────┘
                   Ja │ Nein
              ┌───────┴───────┐
              ▼               ▼
         ┌────────┐     ┌────────┐
         │ Pfad A │     │ Pfad B │
         └────────┘     └────────┘
```

### Decision Trees

```
                    [Frage?]
                   /        \
                 Ja          Nein
                /              \
          [Ergebnis A]    [Frage 2?]
                          /        \
                        Ja          Nein
                       /              \
                 [Ergebnis B]   [Ergebnis C]
```

### Vertical Flow

```
  ┌──────────┐
  │ Phase 1  │
  └────┬─────┘
       │
  ┌────▼─────┐
  │ Phase 2  │
  └────┬─────┘
       │
  ┌────▼─────┐
  │ Phase 3  │
  └──────────┘
```

### Comparison Tables

```
╔═══════════╦══════════════╦══════════════╗
║           ║  Option A    ║  Option B    ║
╠═══════════╬══════════════╬══════════════╣
║ Speed     ║  ★★★★☆      ║  ★★☆☆☆      ║
║ Cost      ║  ★★☆☆☆      ║  ★★★★★      ║
║ Quality   ║  ★★★☆☆      ║  ★★★★☆      ║
╚═══════════╩══════════════╩══════════════╝
```

### Timelines

```
──●────────●────────●────────●────────●──
  R1       R2       R3       R4       R5
  Setup    Score    Key      Push     Final
           starts   turn            scoring
```

### Hierarchy / Org Charts

```
              ┌─────────┐
              │  Root   │
              └────┬────┘
           ┌───────┼───────┐
      ┌────▼───┐ ┌▼────┐ ┌▼────────┐
      │ Child1 │ │ C2  │ │ Child3  │
      └────────┘ └─────┘ └─────────┘
```

## Character Palette

Connectors: `─ │ ┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼`
Double: `═ ║ ╔ ╗ ╚ ╝ ╠ ╣ ╦ ╩ ╬`
Arrows: `► ◄ ▲ ▼ → ← ↑ ↓ ▶ ◀ ────►`
Points: `● ○ ◆ ◇ ★ ☆`
Blocks: `█ ▓ ▒ ░`

## Rules

- Always wrap in code blocks (```) for alignment
- Count characters carefully — misaligned boxes look broken
- Keep line width under 60 chars when possible (printability)
- Use consistent box sizes within one diagram
- Label arrows and connections clearly
- For complex diagrams, build top-to-bottom or left-to-right (not both)
- Test alignment by counting chars in each line
