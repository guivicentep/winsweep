import './style.css';
import './app.css';

import { icon } from './icons.js';
import { mountCleanupView } from './views/cleanup.js';
import { mountOptimizeView } from './views/optimize.js';

const MODULES = [
  {
    id: 'cleanup',
    label: 'Limpeza',
    description: 'Remove arquivos desnecessários',
    icon: 'broom',
    mount: mountCleanupView,
  },
  {
    id: 'optimize',
    label: 'Otimização',
    description: 'Deixa o Windows mais leve',
    icon: 'gauge',
    mount: mountOptimizeView,
  },
];

const app = document.querySelector('#app');

app.innerHTML = `
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">
        ${icon('broom', { size: 22, className: 'brand-icon' })}
        <span class="brand-name">winsweep</span>
      </div>
      <nav class="nav" id="nav"></nav>
    </aside>
    <main class="content"><div class="content-inner" id="content-inner"></div></main>
  </div>
`;

const nav = document.querySelector('#nav');
const contentInner = document.querySelector('#content-inner');

nav.innerHTML = MODULES.map(
  (mod) => `
    <button class="nav-item" data-module="${mod.id}">
      ${icon(mod.icon, { size: 18 })}
      <span>
        <span class="nav-item-title">${mod.label}</span>
        <span class="nav-item-desc">${mod.description}</span>
      </span>
    </button>
  `
).join('');

let activeView = null;

function activateModule(id) {
  const mod = MODULES.find((m) => m.id === id);
  if (!mod) return;

  nav.querySelectorAll('.nav-item').forEach((btn) => {
    btn.classList.toggle('is-active', btn.dataset.module === id);
  });

  if (activeView && typeof activeView.unmount === 'function') {
    activeView.unmount();
  }
  contentInner.innerHTML = '';
  activeView = mod.mount(contentInner);
}

nav.querySelectorAll('.nav-item').forEach((btn) => {
  btn.addEventListener('click', () => activateModule(btn.dataset.module));
});

activateModule(MODULES[0].id);
