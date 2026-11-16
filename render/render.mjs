// Renders the animated stats card (stats.html) with live numbers to a looping GIF.
// Usage (repo root): node render/render.mjs
import puppeteer from "puppeteer-core";
import { readFileSync, writeFileSync, mkdtempSync, rmSync, existsSync, readdirSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { tmpdir } from "node:os";
import { join, dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = dirname(fileURLToPath(import.meta.url)), root = resolve(here, "..");
const FPS = 12, data = JSON.parse(readFileSync(join(root, "assets/stats-data.json"), "utf8"));
const chrome = process.env.CHROME_PATH || ["/usr/bin/google-chrome", "/usr/bin/google-chrome-stable", "/usr/bin/chromium"].find(existsSync);
const browser = await puppeteer.launch({ executablePath: chrome, args: ["--no-sandbox", "--allow-file-access-from-files"] });
const page = await browser.newPage();
await page.setViewport({ width: 880, height: 680 });
await page.evaluateOnNewDocument(d => { window.DATA = d; }, data);
await page.goto(pathToFileURL(join(here, "stats.html")).href);
await page.evaluate(() => window.ready);
const loop = await page.evaluate(() => window.LOOP);
const dir = mkdtempSync(join(tmpdir(), "stats-"));
for (let i = 0; i < loop * FPS; i++) {
  await page.evaluate(t => window.draw(t), i / FPS);
  await page.screenshot({ path: join(dir, String(i).padStart(4, "0") + ".png") });
}
await browser.close();
const tmp = join(dir, "out.gif");
execFileSync("ffmpeg", ["-y", "-loglevel", "error", "-framerate", String(FPS), "-i", join(dir, "%04d.png"),
  "-vf", "split[a][b];[a]palettegen=max_colors=192:stats_mode=diff[p];[b][p]paletteuse=dither=bayer:bayer_scale=4:diff_mode=rectangle", "-loop", "0", tmp]);
// unique file name per render so GitHub's image cache never shows old numbers
const gif = readFileSync(tmp), name = `stats-${createHash("sha1").update(gif).digest("hex").slice(0, 8)}.gif`;
for (const f of readdirSync(join(root, "assets"))) if (/^stats-[0-9a-f]{8}\.gif$/.test(f) && f !== name) rmSync(join(root, "assets", f));
writeFileSync(join(root, "assets", name), gif);
rmSync(dir, { recursive: true, force: true });
const readme = join(root, "README.md");
writeFileSync(readme, readFileSync(readme, "utf8").replace(/\.\/assets\/stats(\.svg|-[0-9a-f]{8}\.gif)/g, `./assets/${name}`));
console.log(`wrote assets/${name}`);
