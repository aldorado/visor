---
name: ascii-art
description: Use when asked to create ASCII art, decorative text borders, styled headers, or visual embellishments using text characters. Triggered by "ascii art", "mach mir ascii art", "text art", "styled text", or when decorative ASCII elements would enhance a document.
user-invocable: true
argument-hint: "[description of what to create]"
---

# ASCII Art Skill

Create ASCII art for documents, cheatsheets, or messages using text characters.

## Style Guide

Use box-drawing characters for clean, printable results:

### Single-line boxes
```
┌──────────────────┐
│  Content here    │
└──────────────────┘
```

### Double-line boxes (for headers/emphasis)
```
╔══════════════════╗
║  IMPORTANT       ║
╠══════════════════╣
║  Details here    ║
╚══════════════════╝
```

### Decorative headers
```
═══════════════════════════
  ⚔  SECTION TITLE  ⚔
═══════════════════════════
```

### Banners
```
╔═══════════════════════════════╗
║  ███  TITLE  ███              ║
╚═══════════════════════════════╝
```

## Character Palette

Box drawing: `┌ ┐ └ ┘ ─ │ ├ ┤ ┬ ┴ ┼`
Double box: `╔ ╗ ╚ ╝ ═ ║ ╠ ╣ ╦ ╩ ╬`
Blocks: `█ ▓ ▒ ░ ▄ ▀ ▌ ▐`
Arrows: `► ◄ ▲ ▼ → ← ↑ ↓ ↗ ↘`
Symbols: `★ ✦ ⚔ ☠ ✠ ⚡ ♦ ● ○ ◆ ◇`
Checks: `✓ ✗ □ ☐ ☑`

## Rules

- Keep it readable — art should enhance, not obscure
- Use monospace-compatible characters only (these files get printed)
- For Warhammer 40K content: use ⚔ ☠ ✠ ⚡ for thematic flair
- Align columns cleanly
- Test that box widths match (count characters!)
- When used in markdown files, wrap in code blocks (```) so alignment is preserved
