export const icons = {
  close: '<path d="M18 6 6 18M6 6l12 12"/>',
  menu: '<path d="M4 6h16M4 12h16M4 18h16"/>',
  search: '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>',
  chevronDown: '<path d="m6 9 6 6 6-6"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  minus: '<path d="M5 12h14"/>',
  play: '<path d="m8 5 11 7-11 7z"/>',
  stop: '<rect width="12" height="12" x="6" y="6" rx="1"/>',
  file: '<path d="M6 2h8l4 4v16H6z"/><path d="M14 2v5h5"/>',
  folder: '<path d="M3 6h7l2 2h9v11H3z"/>',
} as const;

export type IconName = keyof typeof icons;

export function createIcon(name: IconName, label?: string): SVGSVGElement {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.classList.add("ui-icon");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("fill", "none");
  svg.setAttribute("stroke", "currentColor");
  svg.setAttribute("stroke-linecap", "round");
  svg.setAttribute("stroke-linejoin", "round");
  svg.innerHTML = icons[name];
  if (label) {
    svg.setAttribute("role", "img");
    svg.setAttribute("aria-label", label);
  } else {
    svg.setAttribute("aria-hidden", "true");
  }
  return svg;
}
