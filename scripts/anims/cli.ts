// Usage: bun scripts/anims/cli.ts <animation> <command> [--mode rename|overwrite|cancel] [--out file] [--ascii]
//
// <animation> is a file name (without .ts) in scripts/anims/animations/; run
// with no arguments to list them.
//
// Commands:
//   preview  print frames to the terminal (no server needed); --ascii prints
//            one letter per distinct colour plus a legend instead of ANSI colours
//   export   write Sparse JSON to --out (default: <animation>.json)
//   play     start the animation on the device without saving it
//   stop     stop whatever is playing on the device
//   import   save the animation to the device's library via /api/animation/import
//
// Env: CUBIK_URL (default http://localhost:9080), CUBIK_DEVICE (device id, required
// only when discovery finds more than one device).

import { readdir } from "node:fs/promises";
import { parseArgs } from "node:util";
import { WIDTH, toSparse, type Animation, type Frame, type RGB } from "./lib";

const ANIMATIONS_DIR = `${import.meta.dir}/animations`;

const BASE_URL = process.env.CUBIK_URL ?? "http://localhost:9080";

type Device = { id: string; name: string; location: string };

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json" },
  });
  const body = await res.text();
  if (!res.ok) throw new Error(`${init?.method ?? "GET"} ${path}: ${res.status} ${body}`);
  return JSON.parse(body) as T;
}

async function activeDevice(): Promise<Device> {
  const { devices } = await api<{ devices: Device[] }>("/api/devices");
  const wanted = process.env.CUBIK_DEVICE;
  if (wanted) {
    const device = devices.find((d) => d.id === wanted);
    if (!device) throw new Error(`device ${wanted} not found; discovered: ${devices.map((d) => d.id).join(", ")}`);
    return device;
  }
  if (devices.length !== 1) {
    throw new Error(`expected exactly one device, found ${devices.length}; set CUBIK_DEVICE`);
  }
  return devices[0];
}

const toHex = ({ r, g, b }: RGB) => `#${((r << 16) | (g << 8) | b).toString(16).padStart(6, "0")}`;

// Black is '.', every other colour gets a letter in order of first appearance.
function previewAscii(frames: Frame[]) {
  const letters = new Map<string, string>();
  const glyph = (p: RGB) => {
    const key = toHex(p);
    if (key === "#000000") return ".";
    if (!letters.has(key)) letters.set(key, String.fromCharCode(97 + (letters.size % 26)));
    return letters.get(key)!;
  };
  frames.forEach((frame, i) => {
    console.log(`frame ${i}`);
    for (let row = 0; row < frame.pixels.length; row += WIDTH) {
      console.log(frame.pixels.slice(row, row + WIDTH).map(glyph).join(""));
    }
  });
  console.log("legend");
  for (const [hex, letter] of letters) console.log(`${letter} = ${hex}`);
}

function preview(frames: Frame[]) {
  frames.forEach((frame, i) => {
    console.log(`frame ${i}`);
    for (let row = 0; row < frame.pixels.length; row += WIDTH) {
      const line = frame.pixels
        .slice(row, row + WIDTH)
        .map(({ r, g, b }) => `\x1b[48;2;${r};${g};${b}m  `)
        .join("");
      console.log(`${line}\x1b[0m`);
    }
  });
}

async function main() {
  const { positionals, values } = parseArgs({
    args: Bun.argv.slice(2),
    allowPositionals: true,
    options: {
      mode: { type: "string", default: "rename" },
      out: { type: "string" },
      ascii: { type: "boolean", default: false },
    },
  });
  const [animName, command] = positionals;
  if (!animName || !command) {
    const names = (await readdir(ANIMATIONS_DIR)).filter((f) => f.endsWith(".ts")).map((f) => f.slice(0, -3));
    console.error("usage: bun scripts/anims/cli.ts <animation> <preview|export|play|stop|import>");
    console.error(`animations: ${names.join(", ") || "(none yet)"}`);
    process.exit(2);
  }

  const anim: Animation = (await import(`${ANIMATIONS_DIR}/${animName}.ts`)).default;
  const frames = anim.frames();

  switch (command) {
    case "preview":
      if (values.ascii) previewAscii(frames);
      else preview(frames);
      break;
    case "export": {
      const out = values.out ?? `${animName}.json`;
      await Bun.write(out, JSON.stringify(toSparse(anim.name, frames), null, 2));
      console.log(`wrote ${frames.length} frames to ${out}`);
      break;
    }
    case "play": {
      const device = await activeDevice();
      const res = await api<{ frame_count: number }>("/api/animation/start", {
        method: "POST",
        body: JSON.stringify({ device_location: device.location, frames: frames.map((f) => f.pixels) }),
      });
      console.log(`playing "${anim.name}" (${res.frame_count} frames) on ${device.id}`);
      break;
    }
    case "stop": {
      const device = await activeDevice();
      await api("/api/animation/stop", {
        method: "POST",
        body: JSON.stringify({ device_location: device.location }),
      });
      console.log(`stopped ${device.id}`);
      break;
    }
    case "import": {
      const device = await activeDevice();
      const res = await api<{ animation: { id: string; name: string }; renamed_from?: string }>(
        `/api/animation/import?mode=${encodeURIComponent(values.mode!)}`,
        {
          method: "POST",
          body: JSON.stringify({ device_id: device.id, animation: toSparse(anim.name, frames) }),
        },
      );
      const renamed = res.renamed_from ? ` (renamed from "${res.renamed_from}")` : "";
      console.log(`imported "${res.animation.name}"${renamed} as ${res.animation.id} for ${device.id}`);
      break;
    }
    default:
      console.error(`unknown command: ${command}`);
      process.exit(2);
  }
}

main().catch((err) => {
  console.error(err instanceof Error ? err.message : err);
  process.exit(1);
});
