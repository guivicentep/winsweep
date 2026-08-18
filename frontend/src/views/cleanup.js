import { StartScan, DeleteFinding } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { formatBytes, escapeHtml } from '../format.js';
import { icon } from '../icons.js';

/**
 * Monta o módulo de Limpeza dentro de `container`. Retorna um objeto com
 * `unmount()` para remover os listeners de evento ao trocar de módulo.
 */
export function mountCleanupView(container) {
  const state = {
    order: [],
    findings: new Map(), // id -> { dto, status: 'pending'|'deleting'|'deleted'|'skipped'|'error', errorMsg }
    scanning: false,
    freedBytes: 0,
  };

  container.innerHTML = `
    <div class="module-header">
      <h1>Limpeza</h1>
      <p>Analisa locais conhecidos de arquivos desnecessários do Windows e só exclui o que você confirmar, um item de cada vez.</p>
    </div>
    <div class="toolbar">
      <button id="scan-btn" class="btn btn-primary">${icon('refresh')}<span>Analisar meu computador</span></button>
      <span id="status-line" class="status-line"></span>
    </div>
    <div id="summary" class="summary" hidden>
      <div class="summary-item"><strong id="summary-count">0</strong><span>itens encontrados</span></div>
      <div class="summary-item"><strong id="summary-size">0 B</strong><span>ocupados</span></div>
      <div class="summary-item"><strong id="summary-freed">0 B</strong><span>liberados</span></div>
    </div>
    <div id="findings" class="card-list"></div>
    <div id="empty-state" class="empty-state">
      ${icon('broom', { size: 32 })}
      <span>Clique em "Analisar meu computador" para começar.</span>
    </div>
  `;

  const scanBtn = container.querySelector('#scan-btn');
  const statusLine = container.querySelector('#status-line');
  const findingsEl = container.querySelector('#findings');
  const emptyState = container.querySelector('#empty-state');
  const summaryEl = container.querySelector('#summary');
  const summaryCount = container.querySelector('#summary-count');
  const summarySize = container.querySelector('#summary-size');
  const summaryFreed = container.querySelector('#summary-freed');

  function totalSizeBytes() {
    let total = 0;
    for (const { dto } of state.findings.values()) total += dto.sizeBytes;
    return total;
  }

  function renderSummary() {
    const hasFindings = state.findings.size > 0;
    summaryEl.hidden = !hasFindings;
    summaryCount.textContent = String(state.findings.size);
    summarySize.textContent = formatBytes(totalSizeBytes());
    summaryFreed.textContent = formatBytes(state.freedBytes);
  }

  function renderFindingCard(entry) {
    const { dto, status, errorMsg } = entry;
    const card = document.createElement('div');
    card.className = 'card';
    if (status === 'deleted' || status === 'skipped') card.classList.add('is-resolved');
    if (dto.permanent) card.classList.add('is-warning');

    card.innerHTML = `
      <div class="card-head">
        <div>
          <div class="card-title">${escapeHtml(dto.categoryName)}</div>
          <div class="card-meta">${icon('folder', { size: 13 })}<span>${escapeHtml(dto.path)} · ${dto.fileCount} item(ns)</span></div>
        </div>
        <div class="card-size">${formatBytes(dto.sizeBytes)}</div>
      </div>
      <p class="card-desc">${escapeHtml(dto.description)}</p>
      ${dto.permanent ? `<div class="card-note">${icon('alertTriangle', { size: 15 })}<span>Esta ação é definitiva e não pode ser desfeita.</span></div>` : ''}
      <div class="card-actions"></div>
    `;

    const actions = card.querySelector('.card-actions');

    if (status === 'pending' || status === 'deleting' || status === 'error') {
      const deleteBtn = document.createElement('button');
      deleteBtn.className = 'btn btn-danger btn-sm';
      deleteBtn.innerHTML = status === 'deleting'
        ? `${icon('loader', { size: 14, className: 'spin' })}<span>Excluindo...</span>`
        : `${icon('trash', { size: 14 })}<span>${dto.permanent ? 'Esvaziar definitivamente' : 'Excluir (enviar p/ Lixeira)'}</span>`;
      deleteBtn.disabled = status === 'deleting';
      deleteBtn.addEventListener('click', () => handleDelete(dto.id));
      actions.appendChild(deleteBtn);

      const skipBtn = document.createElement('button');
      skipBtn.className = 'btn btn-ghost btn-sm';
      skipBtn.textContent = 'Manter';
      skipBtn.disabled = status === 'deleting';
      skipBtn.addEventListener('click', () => handleSkip(dto.id));
      actions.appendChild(skipBtn);
    }

    if (status === 'deleted') {
      actions.insertAdjacentHTML('beforeend', `<span class="card-result ok">${icon('checkCircle', { size: 14 })}Excluído</span>`);
    } else if (status === 'skipped') {
      actions.insertAdjacentHTML('beforeend', '<span class="card-result">Mantido</span>');
    } else if (status === 'error') {
      actions.insertAdjacentHTML('beforeend', `<span class="card-result error">${icon('alertTriangle', { size: 14 })}Falha: ${escapeHtml(errorMsg || 'erro desconhecido')}</span>`);
    }

    return card;
  }

  function render() {
    findingsEl.innerHTML = '';
    emptyState.hidden = state.findings.size > 0 || state.scanning;
    for (const id of state.order) {
      const entry = state.findings.get(id);
      if (entry) findingsEl.appendChild(renderFindingCard(entry));
    }
    renderSummary();
  }

  async function handleDelete(id) {
    const entry = state.findings.get(id);
    if (!entry) return;
    entry.status = 'deleting';
    render();
    try {
      await DeleteFinding(id);
      state.freedBytes += entry.dto.sizeBytes;
      entry.status = 'deleted';
    } catch (err) {
      entry.status = 'error';
      entry.errorMsg = String(err);
    }
    render();
  }

  function handleSkip(id) {
    const entry = state.findings.get(id);
    if (!entry) return;
    entry.status = 'skipped';
    render();
  }

  function handleScanClick() {
    state.scanning = true;
    state.order = [];
    state.findings.clear();
    state.freedBytes = 0;
    scanBtn.disabled = true;
    scanBtn.innerHTML = `${icon('loader', { size: 16, className: 'spin' })}<span>Analisando...</span>`;
    statusLine.textContent = 'Iniciando análise...';
    render();
    StartScan();
  }

  const offFinding = EventsOn('scan:finding', (dto) => {
    if (!state.findings.has(dto.id)) {
      state.order.push(dto.id);
    }
    state.findings.set(dto.id, { dto, status: 'pending' });
    statusLine.textContent = `Analisando... ${state.findings.size} local(is) encontrado(s) até agora.`;
    render();
  });

  const offDone = EventsOn('scan:done', () => {
    state.scanning = false;
    scanBtn.disabled = false;
    scanBtn.innerHTML = `${icon('refresh')}<span>Analisar novamente</span>`;
    statusLine.textContent = state.findings.size
      ? `Análise concluída: ${state.findings.size} local(is) encontrado(s).`
      : 'Análise concluída: nenhum item desnecessário encontrado.';
    render();
  });

  scanBtn.addEventListener('click', handleScanClick);
  render();

  return {
    unmount() {
      offFinding();
      offDone();
    },
  };
}
