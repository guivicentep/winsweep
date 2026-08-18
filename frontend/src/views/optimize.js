import { ListTweaks, DetectTweak, ApplyTweak, RevertTweak } from '../../wailsjs/go/main/App';
import { escapeHtml } from '../format.js';
import { icon } from '../icons.js';

const CATEGORY_LABEL = {
  energia: 'Energia',
  visual: 'Visual',
  jogos: 'Jogos',
};

const CATEGORY_BADGE_CLASS = {
  energia: 'badge-power',
  visual: 'badge-visual',
  jogos: 'badge-gaming',
};

const STATUS_LABEL = {
  unknown: 'Não verificado',
  checking: 'Verificando...',
  applied: 'Aplicado',
  not_applied: 'Não aplicado',
  error: 'Erro ao verificar',
};

/**
 * Monta o módulo de Otimização dentro de `container`. Nenhuma chamada ao
 * backend acontece automaticamente: o estado de cada ajuste só é consultado
 * quando o usuário clica em "Verificar estado atual", e nada é alterado na
 * máquina sem um clique explícito em "Aplicar" ou "Reverter".
 */
export function mountOptimizeView(container) {
  const state = {
    tweaks: [], // { dto, status: 'unknown'|'checking'|'applied'|'not_applied'|'error', errorMsg, busy: 'apply'|'revert'|null }
    loading: true,
  };

  container.innerHTML = `
    <div class="module-header">
      <h1>Otimização</h1>
      <p>Ajustes de desempenho para computadores mais lentos ou antigos: plano de energia e efeitos visuais do Windows. Cada ajuste é individual e pode ser revertido a qualquer momento.</p>
    </div>
    <div id="optimize-body"></div>
  `;

  const body = container.querySelector('#optimize-body');

  function renderLoading() {
    body.innerHTML = `<div class="empty-state">${icon('loader', { size: 28, className: 'spin' })}<span>Carregando catálogo de ajustes...</span></div>`;
  }

  function statusPillHtml(entry) {
    const cls = {
      unknown: 'pill',
      checking: 'pill',
      applied: 'pill pill-applied',
      not_applied: 'pill pill-not-applied',
      error: 'pill pill-error',
    }[entry.status];

    const iconName = {
      unknown: 'circle',
      checking: 'loader',
      applied: 'checkCircle',
      not_applied: 'circle',
      error: 'alertTriangle',
    }[entry.status];

    const spin = entry.status === 'checking' ? ' spin' : '';
    return `<span class="${cls}">${icon(iconName, { size: 12, className: spin.trim() })}${STATUS_LABEL[entry.status]}</span>`;
  }

  function renderCard(entry) {
    const { dto, status, errorMsg, busy } = entry;
    const card = document.createElement('div');
    card.className = 'card';

    card.innerHTML = `
      <div class="card-head">
        <div>
          <div class="card-title-group">
            <span class="card-title">${escapeHtml(dto.name)}</span>
            <span class="badge ${CATEGORY_BADGE_CLASS[dto.category] ?? ''}">${escapeHtml(CATEGORY_LABEL[dto.category] ?? dto.category)}</span>
          </div>
        </div>
        <div>${statusPillHtml(entry)}</div>
      </div>
      <p class="card-desc">${escapeHtml(dto.description)}</p>
      <div class="card-note is-neutral">${icon('alertTriangle', { size: 15 })}<span>${escapeHtml(dto.impact)}</span></div>
      <div class="card-actions"></div>
    `;

    const actions = card.querySelector('.card-actions');

    const detectBtn = document.createElement('button');
    detectBtn.className = 'btn btn-ghost btn-sm';
    detectBtn.innerHTML = status === 'checking'
      ? `${icon('loader', { size: 14, className: 'spin' })}<span>Verificando...</span>`
      : `${icon('refresh', { size: 14 })}<span>Verificar estado atual</span>`;
    detectBtn.disabled = status === 'checking' || !!busy;
    detectBtn.addEventListener('click', () => handleDetect(dto.id));
    actions.appendChild(detectBtn);

    const applyBtn = document.createElement('button');
    applyBtn.className = 'btn btn-primary btn-sm';
    applyBtn.innerHTML = busy === 'apply'
      ? `${icon('loader', { size: 14, className: 'spin' })}<span>Aplicando...</span>`
      : `${icon('checkCircle', { size: 14 })}<span>Aplicar</span>`;
    applyBtn.disabled = !!busy || status === 'checking';
    applyBtn.addEventListener('click', () => handleApply(dto.id));
    actions.appendChild(applyBtn);

    const revertBtn = document.createElement('button');
    revertBtn.className = 'btn btn-ghost btn-sm';
    revertBtn.innerHTML = busy === 'revert'
      ? `${icon('loader', { size: 14, className: 'spin' })}<span>Revertendo...</span>`
      : `${icon('undo', { size: 14 })}<span>Reverter</span>`;
    revertBtn.disabled = !!busy || status === 'checking';
    revertBtn.addEventListener('click', () => handleRevert(dto.id));
    actions.appendChild(revertBtn);

    if (status === 'error' && errorMsg) {
      actions.insertAdjacentHTML('beforeend', `<span class="card-result error">${icon('alertTriangle', { size: 14 })}${escapeHtml(errorMsg)}</span>`);
    }

    return card;
  }

  function render() {
    if (state.loading) {
      renderLoading();
      return;
    }

    const groups = new Map();
    for (const entry of state.tweaks) {
      const key = entry.dto.category;
      if (!groups.has(key)) groups.set(key, []);
      groups.get(key).push(entry);
    }

    body.innerHTML = '';
    for (const [category, entries] of groups) {
      const section = document.createElement('section');
      section.style.marginBottom = 'var(--space-6)';
      const heading = document.createElement('h2');
      heading.textContent = CATEGORY_LABEL[category] ?? category;
      heading.style.fontSize = 'var(--text-sm)';
      heading.style.textTransform = 'uppercase';
      heading.style.letterSpacing = '0.04em';
      heading.style.color = 'var(--text-tertiary)';
      heading.style.margin = '0 0 var(--space-3)';
      section.appendChild(heading);

      const list = document.createElement('div');
      list.className = 'card-list';
      for (const entry of entries) list.appendChild(renderCard(entry));
      section.appendChild(list);

      body.appendChild(section);
    }
  }

  function findEntry(id) {
    return state.tweaks.find((e) => e.dto.id === id);
  }

  async function handleDetect(id) {
    const entry = findEntry(id);
    if (!entry) return;
    entry.status = 'checking';
    entry.errorMsg = undefined;
    render();
    try {
      const result = await DetectTweak(id);
      entry.status = result;
    } catch (err) {
      entry.status = 'error';
      entry.errorMsg = String(err);
    }
    render();
  }

  async function handleApply(id) {
    const entry = findEntry(id);
    if (!entry) return;
    entry.busy = 'apply';
    entry.errorMsg = undefined;
    render();
    try {
      await ApplyTweak(id);
      await handleDetect(id);
    } catch (err) {
      entry.status = 'error';
      entry.errorMsg = String(err);
    } finally {
      entry.busy = null;
      render();
    }
  }

  async function handleRevert(id) {
    const entry = findEntry(id);
    if (!entry) return;
    entry.busy = 'revert';
    entry.errorMsg = undefined;
    render();
    try {
      await RevertTweak(id);
      await handleDetect(id);
    } catch (err) {
      entry.status = 'error';
      entry.errorMsg = String(err);
    } finally {
      entry.busy = null;
      render();
    }
  }

  render();

  ListTweaks()
    .then((list) => {
      state.tweaks = list.map((dto) => ({ dto, status: 'unknown', busy: null }));
      state.loading = false;
      render();
    })
    .catch((err) => {
      body.innerHTML = `<div class="empty-state">${icon('alertTriangle', { size: 28 })}<span>Falha ao carregar catálogo de ajustes: ${escapeHtml(String(err))}</span></div>`;
    });

  return {
    unmount() {
      // Não há listeners de evento neste módulo (as chamadas são request/response).
    },
  };
}
