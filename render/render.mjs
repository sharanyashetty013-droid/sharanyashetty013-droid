// Renders game.html frame by frame with headless Chrome and stitches a looping GIF.
// Usage (from repo root): node render/render.mjs
import puppeteer from "puppeteer-core";
import { readFileSync, writeFileSync, mkdtempSync, rmSync, existsSync, readdirSync, renameSync } from "node:fs";
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { tmpdir } from "node:os";
import { join, dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, "..");
const FPS = Number(process.env.FPS || 12);
const data = JSON.parse(readFileSync(join(root, "assets/game-data.json"), "utf8"));

const chrome = process.env.CHROME_PATH ||
  ["/usr/bin/google-chrome", "/usr/bin/google-chrome-stable", "/usr/bin/chromium", "/usr/bin/chromium-browser"].find(existsSync);
const browser = await puppeteer.launch({
  executablePath: chrome,
  args: ["--no-sandbox", "--allow-file-access-from-files", "--disable-gpu", "--font-render-hinting=none"],
});
const page = await browser.newPage();
await page.setViewport({ width: 880, height: 440, deviceScaleFactor: 1 });
await page.evaluateOnNewDocument(d => { window.DATA = d; }, data);
await page.goto(pathToFileURL(join(here, "game.html")).href);
await page.evaluate(() => window.ready);
const loop = await page.evaluate(() => window.LOOP);

const dir = mkdtempSync(join(tmpdir(), "frames-"));
const n = Math.round(loop * FPS);
for (let i = 0; i < n; i++) {
  await page.evaluate(t => window.draw(t), i / FPS);
  await page.screenshot({ path: join(dir, String(i).padStart(4, "0") + ".png"), clip: { x: 0, y: 0, width: 880, height: 440 } });
}
await browser.close();

const out = join(root, "assets/banner.gif");
execFileSync("ffmpeg", ["-y", "-loglevel", "error", "-framerate", String(FPS), "-i", join(dir, "%04d.png"),
  "-vf", "split[a][b];[a]palettegen=max_colors=192:stats_mode=diff[p];[b][p]paletteuse=dither=bayer:bayer_scale=4:diff_mode=rectangle",
  "-loop", "0", out]);
rmSync(dir, { recursive: true, force: true });

// unique name per render so GitHub's image cache can never show an old banner
const hash = createHash("sha1").update(readFileSync(out)).digest("hex").slice(0, 8);
const name = `banner-${hash}.gif`;
for (const f of readdirSync(join(root, "assets"))) if (/^banner-[0-9a-f]{8}\.gif$/.test(f) && f !== name) rmSync(join(root, "assets", f));
renameSync(out, join(root, "assets", name));
const readme = join(root, "README.md");
writeFileSync(readme, readFileSync(readme, "utf8").replace(/\.\/assets\/banner(-[0-9a-f]{8})?\.gif/g, `./assets/${name}`));
console.log(`wrote assets/${name} (${n} frames)`);
