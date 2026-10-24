// Checks that the theme colors meet WCAG 2.2 AA contrast: 4.5:1 for text,
// 3:1 for icons and other shapes people need to see. Run: npm run check:contrast
import { entryColors, entryInk, palette } from '../src/theme/colors.ts';

const TEXT = 4.5;
const GRAPHIC = 3;

// [foreground, background, minimum, where it shows up]
const pairs = [
  ['text', 'background', TEXT, 'body text'],
  ['text', 'surface', TEXT, 'list rows, sheets'],
  ['text', 'fill', TEXT, 'secondary buttons'],
  ['textSecondary', 'background', TEXT, 'subtitles, section titles'],
  ['textSecondary', 'surface', TEXT, 'row details and sublabels'],
  ['textSecondary', 'fill', TEXT, 'unselected segments, banners'],
  ['textTertiary', 'background', TEXT, 'footnotes, weekday names'],
  ['textTertiary', 'surface', TEXT, 'placeholders, quiet row text'],
  ['accent', 'background', TEXT, 'links, back buttons, today'],
  ['accent', 'surface', TEXT, 'action rows, plain buttons'],
  ['onAccent', 'accent', TEXT, 'primary buttons, selected today'],
  ['onAccent', 'danger', TEXT, 'destructive primary buttons'],
  ['danger', 'background', TEXT, 'error messages'],
  ['danger', 'surface', TEXT, 'destructive rows'],
  ['danger', 'fill', TEXT, 'destructive secondary buttons'],
  ['success', 'background', TEXT, '"Available" hints'],
  ['success', 'surface', TEXT, 'server status'],
  ['background', 'text', TEXT, 'selected day in the month grid'],
  ['accent', 'accentSoft', GRAPHIC, 'icons on accent badges'],
  ['accent', 'background', GRAPHIC, 'keyboard focus ring'],
  ['textTertiary', 'background', GRAPHIC, 'inactive tab icons'],
  ['textTertiary', 'surface', GRAPHIC, 'switch track when off'],
];

function luminance(hex) {
  const [r, g, b] = [1, 3, 5].map((i) => {
    const c = parseInt(hex.slice(i, i + 2), 16) / 255;
    return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(a, b) {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

const failures = [];
const check = (label, fg, bg, min) => {
  const ratio = contrast(fg, bg);
  if (ratio < min) failures.push(`${label}: ${ratio.toFixed(2)}:1, needs ${min}:1`);
};

for (const [scheme, colors] of Object.entries(palette)) {
  for (const [fg, bg, min, where] of pairs) {
    check(`${scheme} ${fg} on ${bg} (${where})`, colors[fg], colors[bg], min);
  }
}
for (const { id, value } of entryColors) {
  check(`initials and checkmarks on ${id}`, entryInk, value, TEXT);
}

if (failures.length > 0) {
  console.error(`Contrast too low:\n  ${failures.join('\n  ')}`);
  process.exit(1);
}
console.log(`Contrast OK: ${pairs.length * Object.keys(palette).length + entryColors.length} pairs pass WCAG AA.`);
