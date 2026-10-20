import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const sourceDirectory = join(root, "node_modules/lucide-static/icons");
const outputDirectory = join(root, "assets/icons");
const packageInfo = JSON.parse(
  await readFile(join(root, "node_modules/lucide-static/package.json"), "utf8"),
);
const iconNames = [
  "activity",
  "arrow-down",
  "arrow-right",
  "arrow-up-right",
  "chevron-right",
  "circle-check",
  "circle-dot",
  "code-xml",
  "mouse-pointer-click",
  "server",
  "sparkles",
];
const page = await readFile(join(root, "index.html"), "utf8");
const referencedIcons = new Set(
  [...page.matchAll(/sprite\.svg#lucide-([a-z0-9-]+)/g)].map((match) => match[1]),
);
const missingIcons = [...referencedIcons].filter((name) => !iconNames.includes(name));
if (missingIcons.length > 0) {
  throw new Error(`Missing Lucide icons in sprite: ${missingIcons.join(", ")}`);
}

const readAttribute = (attributes, name) => {
  const match = attributes.match(new RegExp(`\\b${name}="([^"]+)"`));
  if (!match) throw new Error(`Lucide icon is missing ${name}.`);
  return match[1];
};

const symbols = await Promise.all(
  iconNames.map(async (name) => {
    const source = await readFile(join(sourceDirectory, `${name}.svg`), "utf8");
    const rootTag = source.match(/<svg\b([^>]*)>/s);
    const body = source.match(/<svg\b[^>]*>([\s\S]*?)<\/svg>/s);
    if (!rootTag || !body) throw new Error(`Could not parse Lucide icon: ${name}`);

    const attributes = rootTag[1];
    const symbolAttributes = [
      `id="lucide-${name}"`,
      `viewBox="${readAttribute(attributes, "viewBox")}"`,
      `fill="${readAttribute(attributes, "fill")}"`,
      `stroke="${readAttribute(attributes, "stroke")}"`,
      `stroke-width="${readAttribute(attributes, "stroke-width")}"`,
      `stroke-linecap="${readAttribute(attributes, "stroke-linecap")}"`,
      `stroke-linejoin="${readAttribute(attributes, "stroke-linejoin")}"`,
    ].join(" ");

    return `  <symbol ${symbolAttributes}>${body[1].trim()}</symbol>`;
  }),
);

await mkdir(outputDirectory, { recursive: true });
await writeFile(
  join(outputDirectory, "sprite.svg"),
  [
    `<!-- Lucide icons from lucide-static v${packageInfo.version}, ISC license. See LICENSE. -->`,
    '<svg xmlns="http://www.w3.org/2000/svg">',
    "<defs>",
    ...symbols,
    "</defs>",
    "</svg>",
    "",
  ].join("\n"),
);
await writeFile(
  join(outputDirectory, "LICENSE"),
  await readFile(join(root, "node_modules/lucide-static/LICENSE")),
);

console.log(`Generated ${iconNames.length} Lucide icons in assets/icons/sprite.svg`);
