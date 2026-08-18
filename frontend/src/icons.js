// Conjunto mínimo de ícones inline (stroke, 24x24), sem dependências
// externas — o app não faz nenhuma requisição de rede para renderizar a UI.
const ICONS = {
  broom: '<path d="M19 20 9.5 10.5"/><path d="m11.5 4.5 5 5-8 8-6-1.5 1.5-6 7.5-5.5Z"/><path d="M6 19 4 21"/>',
  gauge: '<path d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z"/><path d="M12 2a10 10 0 1 0 8.66 15"/><path d="m13.4 10.6 4.6-5.6"/>',
  trash: '<path d="M3 6h18"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/>',
  undo: '<path d="M9 14 4 9l5-5"/><path d="M4 9h10.5a5.5 5.5 0 0 1 0 11H11"/>',
  checkCircle: '<circle cx="12" cy="12" r="9"/><path d="m8.5 12.3 2.4 2.4 4.6-5.4"/>',
  circle: '<circle cx="12" cy="12" r="9"/>',
  alertTriangle: '<path d="M10.3 3.9 1.9 18a2 2 0 0 0 1.7 3h16.8a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z"/><path d="M12 9v4"/><path d="M12 17h.01"/>',
  refresh: '<path d="M21 12a9 9 0 0 1-15.4 6.4L3 16"/><path d="M3 12a9 9 0 0 1 15.4-6.4L21 8"/><path d="M3 21v-5h5"/><path d="M21 3v5h-5"/>',
  folder: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z"/>',
  loader: '<path d="M12 2v4"/><path d="m16.2 7.8 2.9-2.9"/><path d="M18 12h4"/><path d="m16.2 16.2 2.9 2.9"/><path d="M12 18v4"/><path d="m4.9 19.1 2.9-2.9"/><path d="M2 12h4"/><path d="m4.9 4.9 2.9 2.9"/>',
};

/**
 * Retorna o markup de um ícone SVG inline.
 * @param {keyof typeof ICONS} name
 * @param {{ size?: number, className?: string }} [opts]
 */
export function icon(name, opts = {}) {
  const size = opts.size ?? 18;
  const cls = opts.className ? ` class="${opts.className}"` : '';
  const body = ICONS[name];
  if (!body) return '';
  return `<svg${cls} width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${body}</svg>`;
}
