---
name: cubik-animation
description: Animation authoring for the Cubik LED cube as TypeScript code. Use when the user wants a new animation or scene on the cube, wants to tweak one written earlier, or wants one saved to the device's animation library.
---

# Cubik animation authoring

Animations are code: a file in `scripts/anims/animations/<name>.ts` default-exporting `{ name, frames }` (`satisfies Animation`), built with the helpers in `scripts/anims/lib.ts` and driven through `mise run anim -- <name> <command>`. Read `lib.ts` for the drawing API and the header of `scripts/anims/cli.ts` for commands and env vars.

`animations/` is gitignored — every animation is a local draft. Only helper changes (`lib.ts`, `cli.ts`) are ever committed.

## The canvas

- 20×5 pixels, `(0,0)` top-left. Every element is a mosaic of 1–3 pixels, so a scene is a pixel budget, not a picture.
- Playback is fixed at **1 FPS** by the device rate limit — each frame is a one-second keyframe. There is no per-animation speed.
- Frames loop. The **seam** (last frame → first) must look like any other step.

## Steps

1. **Sketch.** Before writing code, show the user an ASCII mockup of the 20×5 grid with a legend, and name the trade-offs the pixel budget forces (what overlaps, what gets cut). Ask about each open choice — background, element count, placement, what moves. Done when the user has answered every open choice.
2. **Write** the animation file. Pick the loop length as the least common multiple of every element's cycle period so the seam is clean.
3. **Check** with `mise run anim -- <name> preview --ascii` (letters per colour plus a legend — readable without seeing colour). Walk every frame, including the seam, for: unintended overlaps, elements vanishing, off-by-one clipping at edges, and motion the user did not ask for. Done when every frame is accounted for; then show the user the key frames in ASCII.
4. **Play** on the cube: the backend must be running (`mise run back-dev` in the background; `curl localhost:9080/api/devices` answers when it is up). `mise run anim -- <name> play` shows it without saving. The user's eyes are the only judge of how it looks — ask what they see, iterate on their feedback, and replay after every change.
5. **Import** only when the user says so: `mise run anim -- <name> import --mode cancel` for a first save (fails loudly on a name clash instead of silently creating a "(2)" copy); `--mode overwrite` to update an animation already in the library.

## Design lessons

- **Draw order is z-order.** Draw what sits behind first (sea → fish → island), so occlusion is free. `Frame.set` clips out-of-bounds writes, so elements can slide off edges.
- **Stillness reads better than motion.** At 1 FPS a 1-px shift every second is loud; users preferred a static palm over a swaying one and a gull flapping in place over one crossing the sky. Default to animating few elements, with small cycles (flap, pulse, shimmer).
- **Ping-pong beats wraparound** for travel: move across, then turn in place by swapping head/tail colours on the same pixels — no teleport at the seam.
- **Two movers on one row collide.** Opposite-direction movers on a loop meet twice per cycle; resolve visible crossings explicitly (e.g. hop one to another row) or avoid them.
- **LED colour is not screen colour.** Dim values turn muddy; brown reads as dim orange; pure white looks like a glitch (use grey). Keep a black background and high contrast between neighbours of similar hue (sun vs sand). Expect colour tweaks after the first play.
