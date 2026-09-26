// Helpers for authoring Cubik animations as code.
//
// Playback is fixed at 1 FPS by the backend (animation.go), so every frame is
// shown for one second — design animations as a sequence of keyframes.

export const WIDTH = 20;
export const HEIGHT = 5;
export const FORMAT_VERSION = "1.0";

export type RGB = { r: number; g: number; b: number };

export const BLACK: RGB = { r: 0, g: 0, b: 0 };

export type Animation = {
  name: string;
  frames: () => Frame[];
};

// Frame is a row-major WIDTH x HEIGHT pixel buffer, (0,0) is top-left.
export class Frame {
  readonly pixels: RGB[];

  constructor(fill: RGB = BLACK) {
    this.pixels = Array.from({ length: WIDTH * HEIGHT }, () => ({ ...fill }));
  }

  // Out-of-bounds writes are ignored so shapes can slide off the edges.
  set(x: number, y: number, color: RGB): this {
    if (x < 0 || x >= WIDTH || y < 0 || y >= HEIGHT) return this;
    this.pixels[y * WIDTH + x] = { ...color };
    return this;
  }

  get(x: number, y: number): RGB {
    return this.pixels[y * WIDTH + x];
  }

  // Draws a bitmap where each string row uses '#' for lit pixels.
  blit(bitmap: string[], x: number, y: number, color: RGB): this {
    bitmap.forEach((row, dy) => {
      [...row].forEach((ch, dx) => {
        if (ch === "#") this.set(x + dx, y + dy, color);
      });
    });
    return this;
  }
}

export function rgb(r: number, g: number, b: number): RGB {
  const clamp = (v: number) => Math.max(0, Math.min(255, Math.round(v)));
  return { r: clamp(r), g: clamp(g), b: clamp(b) };
}

export function hex(value: string): RGB {
  const n = parseInt(value.replace(/^#/, ""), 16);
  return { r: (n >> 16) & 0xff, g: (n >> 8) & 0xff, b: n & 0xff };
}

// h in degrees [0, 360), s and v in [0, 1].
export function hsv(h: number, s = 1, v = 1): RGB {
  const hh = (((h % 360) + 360) % 360) / 60;
  const c = v * s;
  const x = c * (1 - Math.abs((hh % 2) - 1));
  const m = v - c;
  const [r, g, b] =
    hh < 1 ? [c, x, 0] : hh < 2 ? [x, c, 0] : hh < 3 ? [0, c, x] : hh < 4 ? [0, x, c] : hh < 5 ? [x, 0, c] : [c, 0, x];
  return rgb((r + m) * 255, (g + m) * 255, (b + m) * 255);
}

export function scale(color: RGB, factor: number): RGB {
  return rgb(color.r * factor, color.g * factor, color.b * factor);
}

// Sparse JSON wire format (spec.yml#/components/schemas/SparseAnimation).
export function toSparse(name: string, frames: Frame[]) {
  return {
    version: FORMAT_VERSION,
    name,
    width: WIDTH,
    height: HEIGHT,
    frames: frames.map((frame) =>
      frame.pixels.flatMap((p, i) =>
        p.r === 0 && p.g === 0 && p.b === 0
          ? []
          : [{ x: i % WIDTH, y: Math.floor(i / WIDTH), c: (p.r << 16) | (p.g << 8) | p.b }],
      ),
    ),
  };
}
