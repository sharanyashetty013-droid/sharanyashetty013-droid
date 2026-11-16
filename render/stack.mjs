// Builds assets/cards/stack.svg (tech stack with real logos). Edit `groups` below, then run:
//   cd render && npm install && node stack.mjs ../assets/cards/stack.svg
import { readFileSync, writeFileSync } from "node:fs";
const S = JSON.parse(readFileSync(new URL("node_modules/", import.meta.url).pathname + "@iconify-json/skill-icons/icons.json")), L = JSON.parse(readFileSync(new URL("node_modules/", import.meta.url).pathname + "@iconify-json/logos/icons.json")), SI = JSON.parse(readFileSync(new URL("node_modules/", import.meta.url).pathname + "@iconify-json/simple-icons/icons.json"));
const font = (name, file, w) => `@font-face{font-family:${name};font-weight:${w};src:url(data:font/woff2;base64,${readFileSync(file).toString("base64")}) format("woff2")}`;
const F = new URL("../assets/fonts/", import.meta.url).pathname;
let uid = 0;
function icon(set, name, color) {
  const ic = set.icons[name]; if (!ic) throw new Error("missing " + name);
  const w = ic.width || set.width || 24, h = ic.height || set.height || 24, p = "i" + (uid++) + "_";
  let body = ic.body.replace(/id="([^"]+)"/g, `id="${p}$1"`).replace(/url\(#([^)]+)\)/g, `url(#${p}$1)`).replace(/href="#([^"]+)"/g, `href="#${p}$1"`);
  if (color) body = body.replace(/currentColor/g, color);
  return { w, h, body };
}
// tile: skill-icons already include their rounded background; others get a matching soft tile
const T = (label, kind, name, color) => ({ label, kind, name, color });
const groups = [
  ["Languages", [T("Go","s","golang"),T("Solidity","s","solidity"),T("C++","s","cpp"),T("C","s","c"),T("Python","s","python-light"),T("JavaScript","s","javascript"),T("TypeScript","s","typescript"),T("HTML","s","html"),T("CSS","s","css")]],
  ["Blockchain", [T("Ethereum","l","ethereum"),T("Bitcoin","l","bitcoin"),T("Hardhat","l","hardhat-icon"),T("Foundry","x","F","#1a1a1a"),T("IPFS","s","ipfs-light")]],
  ["Backend & frontend", [T("Node.js","s","nodejs-light"),T("React","s","react-light"),T("Next.js","s","nextjs-light"),T("Vite","s","vite-light"),T("Tailwind","s","tailwindcss-light")]],
  ["AI & automation", [T("n8n","l","n8n-icon"),T("OpenAI","l","openai-icon"),T("Ollama","l","ollama")]],
  ["Tools & deploy", [T("Git","s","git"),T("Actions","s","githubactions-light"),T("Docker","s","docker"),T("Vercel","s","vercel-light"),T("Linux","s","linux-light"),T("Bash","s","bash-light"),T("Figma","s","figma-light")]],
  ["Problem solving", [T("LeetCode","si","leetcode","#FFA116"),T("DSA · C++","s","cpp")]],
];
const W = 1320, TS = 56, STEP = 68, PER = 9, COLS = [34, 690];
let out = "", colY = [30, 30];
// left column gets the first three groups, right column the rest
groups.forEach(([title, items], gi) => {
  const c = gi < 3 ? 0 : 1, X0 = COLS[c];
  let y = colY[c];
  out += `<text x="${X0}" y="${y + 16}" class="px" style="font-size:13px;fill:#9a6b0c">${title.toUpperCase().replace("&", "&amp;")}</text>`;
  y += 28;
  items.forEach((it, i) => {
    const col = i % PER, row = Math.floor(i / PER), x = X0 + col * STEP, ty = y + row * (TS + 34);
    if (it.kind === "s") { const ic = icon(S, it.name); out += `<svg x="${x}" y="${ty}" width="${TS}" height="${TS}" viewBox="0 0 ${ic.w} ${ic.h}">${ic.body}</svg>`; }
    else {
      out += `<rect x="${x}" y="${ty}" width="${TS}" height="${TS}" rx="13" fill="#F4F2ED"/>`;
      if (it.kind === "x") out += `<text x="${x + TS / 2}" y="${ty + 39}" text-anchor="middle" style="font-weight:800;font-size:30px;fill:${it.color}">${it.name}</text>`;
      else { const ic = it.kind === "l" ? icon(L, it.name) : icon(SI, it.name, it.color); const s = 34, sc = Math.min(s / ic.w, s / ic.h), iw = ic.w * sc, ih = ic.h * sc;
        out += `<svg x="${x + (TS - iw) / 2}" y="${ty + (TS - ih) / 2}" width="${iw}" height="${ih}" viewBox="0 0 ${ic.w} ${ic.h}">${ic.body}</svg>`; }
    }
    out += `<text x="${x + TS / 2}" y="${ty + TS + 19}" text-anchor="middle" style="font-weight:800;font-size:13px;fill:#14234a">${it.label}</text>`;
  });
  colY[c] = y + Math.ceil(items.length / PER) * (TS + 34) + 18;
});
const y = Math.max(...colY);
const H = y + 4;
const frame = `<rect x="6" y="6" width="${W - 6}" height="${H - 6}" fill="#c3cfe0"/><rect x="0" y="0" width="${W - 6}" height="${H - 6}" fill="#7d8a9e"/><rect x="4" y="4" width="${W - 14}" height="${H - 14}" fill="#f6f8fc"/><rect x="4" y="4" width="${W - 14}" height="10" fill="#e4ecf8"/>`;
const svg = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}" role="img" aria-label="Tech stack"><style>${font("Pixel", F + "pixel.woff2", 400)}${font("Nunito", F + "nunito-800.woff2", 800)}text{font-family:Nunito,Arial,sans-serif}.px{font-family:Pixel,monospace}</style>${frame}${out}</svg>`;
writeFileSync(process.argv[2], svg);
console.log("height", H);
