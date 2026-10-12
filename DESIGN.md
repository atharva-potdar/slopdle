# Design Guide: Fixing Vibe-Coded Design

Turn AI-generated code into interfaces that feel designed, not generated.

AI tools excel at producing something that works. They are far worse at
producing something with a point of view. The result is a recognizable
"default look": purple gradients, rounded cards, glowing buttons, a rainbow of
colors. Once you can spot the tells, you can fix them. Five original rules,
sharpened, plus five more that have become common.

---

## 1. Tasteless Text Gradients

**Problem:** AI sprinkles blue-to-green and pink-to-purple gradients over
headlines, buttons, badges for no reason. Gradient text tanks legibility at
small sizes — color shifts under letters, weakens contrast.

**Fix:** Default to solid colors. Reserve gradients for one intentional moment,
like a single hero word or one key call to action.

**Rules:**

- Never gradient text on body copy, labels, anything under ~24px.
- Hues close together (analogous), never opposite ends of wheel.
- One gradient per screen max.
- Check contrast at gradient's *lightest* point, not average.

---

## 2. Too Many Colors

**Problem:** `blue-500`, `green-500`, `pink-500`, `amber-500`. AI pulls from
entire Tailwind palette, every section gets own personality, page feels chaotic.

**Fix:** Pick tight system, stay inside it.

**Palette structure:**

- **Neutrals (workhorse):** 90% of UI. Background, surface, border, text, muted.
- **Primary:** brand color, main actions only.
- **Accent:** rare pop for highlights, key moments.
- **Semantic:** success, warning, error — status *only*.

Define as tokens once, reference everywhere:

```css
:root {
  --bg: #fafaf9;
  --surface: #ffffff;
  --text: #1c1917;
  --muted: #78716c;
  --primary: #e11d48;
  --accent: #0f766e;
}
```

AI writes `bg-purple-500` → bug. Tokens only.

---

## 3. Typography Ignored

**Problem:** Default system fonts, uniform size/weight, cramped or floaty line
spacing. Eye has nowhere to land first.

**Fix:** Real typefaces, clear scale, mind details.

- **Two fonts max.** Expressive serif headings + clean sans body is classic.
  Single versatile family in multiple weights also works.
- **Real size contrast.** Headings noticeably bigger, not 10% bigger. Ratio
  ~1.25–1.5 between steps.
- **Line height:** 1.5–1.7 body, 1.1–1.25 large headings.
- **Tighten large headings:** slight negative letter-spacing (e.g. `-0.02em`).
- **Line length:** 60–75 chars for comfortable reading.
- **Fluid sizing** via `clamp()` across screens.

---

## 4. Spacing Without Rhythm

**Problem:** Identical padding everywhere. Same gaps → nothing groups, nothing
stands out.

**Fix:** Spacing communicates relationships. Related items close, separate
sections generous.

- **Scale only:** 4, 8, 12, 16, 24, 32, 48, 64, 96. Never invent values.
- **Proximity = relationship:** label/input 4–8px apart; form sections 32–48px.
- **Real breathing room:** 80–128px vertical padding between major sections
  feels more premium than expected.
- **Align to grid**, keep consistent left edges.
- Doubt → more whitespace, not more decoration.

---

## 5. Effects Everywhere

**Problem:** Shadows, blurs, glassmorphism, glows, borders on every element.
Visual noise; every component fights for attention.

**Fix:** Strip static effects, invest in interaction feel.

- **One shadow style**, sparingly (elevated elements: modals, dropdowns only).
- Prefer **borders + background shifts** over heavy shadows for surfaces.
- **Microinteractions:** subtle hover, focus rings, pressed states, 150–250ms
  ease-out transitions.
- Respect **`prefers-reduced-motion`**.
- Animate `transform` + `opacity`, never layout properties.

---

## 6. Everything Is a Rounded Card

**Problem:** Every content chunk wrapped in rounded box with border + shadow.
Cards inside cards inside cards. Layout = stack of identical containers.

**Fix:** Let content sit directly on page. Cards only for truly distinct,
self-contained units.

- Dividers, whitespace, alignment group content instead of boxes.
- Vary layouts: full-bleed sections, split columns, large type blocks,
  asymmetry.
- One corner-radius philosophy (sharp, subtle, very round), applied
  consistently.

---

## 7. Generic Hero Syndrome

**Problem:** Centered headline, vague subtitle ("Supercharge your workflow"),
two buttons, glowing blob background. Every AI landing page = same page.

**Fix:** Say something specific, show real product.

- Concrete claim a competitor couldn't copy. No buzzwords.
- Actual screenshot, demo, photo — not decorative blobs.
- Left-aligned or asymmetric hero for character.
- One primary CTA, not two competing ones.

---

## 8. Emoji and Icon Overload

**Problem:** Feature lists with 🚀 ✨ 🎯, every heading preceded by icon in
colored circle.

**Fix:** Icons only when they add meaning.

- **Single icon set**, consistent stroke weight + size.
- Remove decorative icons. Text clear → icon unneeded.
- Replace emoji with custom marks, numbers, simple typographic details.

---

## 9. Copy That Sounds Generated

**Problem:** Design isn't just visual. "Seamless," "elevate," "unlock the power
of," "in today's fast-paced world" signal AI-written content, undermine all
polish.

**Fix:** Write like a person who knows the product.

- Specific: numbers, names, real outcomes.
- Cut adjectives, keep verbs.
- Read aloud. Wouldn't say it to a friend → rewrite.

---

## 10. Accessibility and States Forgotten

**Problem:** Low-contrast gray text, missing focus indicators, no empty /
loading / error states, tiny tap targets. Happy path fine, breaks everywhere
else.

**Fix:** Part of design, not extras.

- **WCAG AA contrast** (4.5:1 body text).
- **Focus states** visible + attractive, never removed.
- Design **empty, loading, error, success states** deliberately.
- Touch targets ≥ **44×44px**.

---

## Bonus: Prompt the AI Better

Most slop comes from vague prompts. Constrain up front:

> "Use only these design tokens: [paste palette]. Use [Font A] for headings
> and [Font B] for body. No gradients, no glassmorphism, no shadows except on
> modals. Spacing scale 4/8/16/24/48/96. Avoid nested cards. Specific,
> plain-language copy, no marketing buzzwords."

More habits:

- **Start with reference.** Point to admired site, describe *why* it works.
- **Iterate in small passes.** Typography → color → spacing → motion.
- **Review with checklist** before shipping.

---

## Quick Audit Checklist

- [ ] ≤3 brand colors, defined as tokens?
- [ ] ≤1 gradient on page?
- [ ] ≤2 fonts, clear size hierarchy?
- [ ] Spacing follows scale, tight groups + wide sections?
- [ ] Shadows/blurs rare + consistent?
- [ ] Fewer cards than first instinct?
- [ ] Hero specific, copy human?
- [ ] Hover, focus, empty, error states exist?
- [ ] Contrast checks pass?

---

**Core idea:** AI gives you *more*. Good design is mostly *restraint* — fewer
colors, fewer effects, fewer boxes, every remaining choice made on purpose.
