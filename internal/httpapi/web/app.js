const state = { sessions: [], selectedID: null, bundle: null, toastTimer: null };
const $ = (selector) => document.querySelector(selector);

function escapeHTML(value) {
  return String(value ?? '').replace(/[&<>'"]/g, (character) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[character]));
}

function formatDate(value) {
  if (!value) return '--';
  return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value));
}

function statusLabel(status) {
  return ({ draft: '草稿', measuring: '测量中', pending_review: '待复核', rework: '返工', ready_to_seal: '可封存', sealed: '已封存' })[status] || status;
}

function showToast(message, kind = 'success') {
  const toast = $('#toast');
  toast.textContent = message;
  toast.className = `toast visible ${kind}`;
  clearTimeout(state.toastTimer);
  state.toastTimer = setTimeout(() => { toast.className = 'toast'; }, 3600);
}

async function api(path, options = {}) {
  const response = await fetch(path, { headers: { 'Content-Type': 'application/json', ...(options.headers || {}) }, ...options });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(body.error?.message || '请求失败');
  return body;
}

async function loadSessions(selectID = state.selectedID) {
  const body = await api('/api/sessions');
  state.sessions = body.sessions || [];
  renderSessions();
  if (selectID && state.sessions.some((session) => session.id === selectID)) await selectSession(selectID);
  else if (state.sessions.length) await selectSession(state.sessions[0].id);
  else clearDetail();
}

function renderSessions() {
  const query = $('#session-search').value.trim().toLowerCase();
  const sessions = state.sessions.filter((session) => `${session.deviceID} ${session.deviceName} ${session.id}`.toLowerCase().includes(query));
  $('#session-count').textContent = state.sessions.length;
  $('#empty-sessions').classList.toggle('visible', sessions.length === 0);
  $('#session-list').innerHTML = sessions.map((session) => `<button class="session-item ${session.id === state.selectedID ? 'active' : ''}" data-session-id="${escapeHTML(session.id)}">
    <div class="session-item-top"><span class="session-item-name">${escapeHTML(session.deviceName)}</span><span class="status-dot ${escapeHTML(session.status)}"></span></div>
    <div class="session-item-id">${escapeHTML(session.deviceID)}</div>
    <div class="session-item-bottom"><span>${statusLabel(session.status)}</span><span>${formatDate(session.updatedAt)}</span></div>
  </button>`).join('');
  document.querySelectorAll('.session-item').forEach((item) => item.addEventListener('click', () => selectSession(item.dataset.sessionId)));
}

async function selectSession(id) {
  try {
    const body = await api(`/api/sessions/${encodeURIComponent(id)}`);
    state.selectedID = id;
    state.bundle = body;
    renderSessions();
    renderDetail();
  } catch (error) { showToast(error.message, 'error'); }
}

function clearDetail() {
  state.selectedID = null;
  state.bundle = null;
  $('#empty-detail').classList.remove('hidden');
  $('#detail-content').classList.add('hidden');
}

function renderDetail() {
  const data = state.bundle;
  if (!data) return clearDetail();
  $('#empty-detail').classList.add('hidden');
  $('#detail-content').classList.remove('hidden');
  const session = data.session;
  const samples = data.samples || [];
  const measurements = data.measurements || [];
  $('#detail-id').textContent = `SESSION / ${session.id}`;
  $('#detail-title').textContent = session.deviceName;
  $('#detail-meta').textContent = `${session.deviceID}  ·  ${session.observingBand}  ·  负责人 ${session.owner}`;
  $('#detail-status').textContent = statusLabel(session.status);
  $('#detail-status').className = `status-badge ${session.status}`;
  $('#detail-version').textContent = session.version;
  const progress = data.progress || {};
  const qualified = progress.qualifiedCount || 0;
  const measuredCount = progress.measuredCount || 0;
  $('#metric-samples').textContent = samples.length;
  $('#metric-samples-note').textContent = samples.length ? `${measuredCount} 个样本已测` : '待登记';
  $('#metric-qualified').textContent = samples.length ? `${Math.round(qualified / samples.length * 100)}%` : '0%';
  $('#metric-qualified-note').textContent = measuredCount ? `${qualified} 个最新读数合格` : '尚未开始';
  $('#metric-audits').textContent = (data.audit || []).length;
  $('#metric-audits-note').textContent = data.auditVerified ? '哈希链完整' : '完整性待校验';
  $('#metric-band').textContent = session.observingBand;
  $('#metric-owner').textContent = `负责人 ${session.owner}`;
  $('#measurement-progress').textContent = `${measuredCount} / ${samples.length}`;
  renderActionPanel(session, samples, measurements, progress.nextSampleID);
  renderSamples(samples, measurements);
  renderAudit(data.audit || [], data.auditVerified);
  renderCertificate(data.certificate);
}

function renderActionPanel(session, samples, measurements, nextSampleID) {
  const panel = $('#action-panel');
  if (session.status === 'draft') {
    panel.innerHTML = `<div class="action-header"><div><h4>登记标准样本</h4><p class="action-copy">建立本次校准的可追溯参考值和容许偏差。</p></div></div><form id="register-form" class="inline-form"></form>`;
    renderInlineSampleForm($('#register-form'));
    $('#register-form').addEventListener('submit', submitSamples);
    return;
  }
  if (session.status === 'measuring' || session.status === 'rework') {
    const next = samples.find((sample) => sample.id === nextSampleID)
      || samples.find((sample) => !measurements.some((measurement) => measurement.sampleID === sample.id))
      || samples.find((sample) => {
        const latest = measurements.filter((measurement) => measurement.sampleID === sample.id).reduce((current, measurement) => {
          if (!current || measurement.measurementSequence > current.measurementSequence) return measurement;
          return current;
        }, null);
        return latest && !latest.withinTolerance;
      })
      || samples[0];
    panel.innerHTML = `<div class="action-header"><div><h4>${session.status === 'rework' ? '返工补测' : '录入测量'}</h4><p class="action-copy">${next ? `当前顺序样本：${escapeHTML(next.sampleNumber)}，每次提交都必须携带当前版本。` : '请先登记标准样本。'}</p></div></div><form id="measurement-form" class="inline-form"><label>样本<select name="sampleID" required>${samples.map((sample) => `<option value="${escapeHTML(sample.id)}" ${sample.id === next?.id ? 'selected' : ''}>${escapeHTML(sample.sampleNumber)} / ${escapeHTML(sample.referenceValue)} ${escapeHTML(sample.unit)}</option>`).join('')}</select></label><label>实际测量值<input name="measuredValue" type="number" step="any" min="0" required inputmode="decimal"></label><label>操作员<input name="operator" required placeholder="姓名"></label><label>幂等键<input name="idempotencyKey" required placeholder="例如 shift-a-001"></label><label>备注<textarea name="note" placeholder="可选"></textarea></label><button class="button button-primary" type="submit">提交读数</button></form>`;
    $('#measurement-form').addEventListener('submit', submitMeasurement);
    return;
  }
  if (session.status === 'pending_review') {
    panel.innerHTML = `<div class="action-header"><div><h4>质量复核</h4><p class="action-copy">逐项偏差已计算，请由复核员给出通过或返工决定。</p></div></div><form id="review-form" class="inline-form review-form"><label>复核员<input name="reviewer" required placeholder="姓名"></label><label>结论<select name="conclusion"><option value="passed">通过</option><option value="rework">退回返工</option></select></label><label>返工原因<input name="reworkReason" placeholder="退回时必填"></label><button class="button button-primary" type="submit">提交复核</button></form>`;
    $('#review-form').addEventListener('submit', submitReview);
    return;
  }
  if (session.status === 'ready_to_seal') {
    panel.innerHTML = `<div class="action-header"><div><h4>封存校准会话</h4><p class="action-copy">复核已通过。封存后所有样本、测量与审计事件只读。</p></div></div><form id="seal-form" class="inline-form seal-form"><label>封存人<input name="sealedBy" required value="${escapeHTML(session.owner)}"></label><button class="button button-safe" type="submit">生成校准证书</button></form>`;
    $('#seal-form').addEventListener('submit', submitSeal);
    return;
  }
  panel.innerHTML = session.status === 'sealed' ? '<p class="action-copy">会话已封存，业务数据进入只读边界。</p>' : '';
}

function renderInlineSampleForm(form) {
  form.innerHTML = `<label>样本编号<input name="sampleNumber" required placeholder="例如 REF-01"></label><label>参考值<input name="referenceValue" type="number" step="any" min="0" required inputmode="decimal"></label><label>单位<input name="unit" required placeholder="例如 ADU"></label><label>允许偏差<input name="allowedDelta" type="number" step="any" min="0.000001" required inputmode="decimal"></label><label>登记人<input name="registeredBy" required placeholder="姓名"></label><button class="button button-primary" type="submit">登记样本</button>`;
}

function readForm(form) { return Object.fromEntries(new FormData(form).entries()); }

async function submitSamples(event) {
  event.preventDefault();
  const value = readForm(event.currentTarget);
  value.referenceValue = Number(value.referenceValue);
  value.allowedDelta = Number(value.allowedDelta);
  try { await api(`/api/sessions/${encodeURIComponent(state.selectedID)}/samples`, { method: 'POST', body: JSON.stringify({ expectedVersion: state.bundle.session.version, samples: [value] }) }); showToast('标准样本已登记'); await loadSessions(state.selectedID); } catch (error) { showToast(error.message, 'error'); }
}

async function submitMeasurement(event) {
  event.preventDefault();
  const value = readForm(event.currentTarget);
  value.measuredValue = Number(value.measuredValue);
  try { await api(`/api/sessions/${encodeURIComponent(state.selectedID)}/measurements`, { method: 'POST', body: JSON.stringify({ ...value, expectedVersion: state.bundle.session.version }) }); showToast('测量记录已写入账本'); await loadSessions(state.selectedID); } catch (error) { showToast(error.message, 'error'); }
}

async function submitReview(event) {
  event.preventDefault();
  const value = readForm(event.currentTarget);
  try { await api(`/api/sessions/${encodeURIComponent(state.selectedID)}/review`, { method: 'POST', body: JSON.stringify({ ...value, expectedVersion: state.bundle.session.version }) }); showToast(value.conclusion === 'passed' ? '复核通过，可以封存' : '已退回返工'); await loadSessions(state.selectedID); } catch (error) { showToast(error.message, 'error'); }
}

async function submitSeal(event) {
  event.preventDefault();
  const value = readForm(event.currentTarget);
  try { await api(`/api/sessions/${encodeURIComponent(state.selectedID)}/seal`, { method: 'POST', body: JSON.stringify({ ...value, expectedVersion: state.bundle.session.version }) }); showToast('校准证书已生成'); await loadSessions(state.selectedID); } catch (error) { showToast(error.message, 'error'); }
}

function renderSamples(samples, measurements) {
  const rows = samples.map((sample) => {
    const reading = [...measurements].reverse().find((measurement) => measurement.sampleID === sample.id);
    const stateClass = !reading ? 'pending' : reading.withinTolerance ? 'ok' : 'out';
    const status = !reading ? '待测量' : reading.withinTolerance ? '合格' : '超差';
    return `<div class="sample-row"><div><strong>${escapeHTML(sample.sampleNumber)}</strong><small>${escapeHTML(sample.id.slice(0, 16))}</small></div><div><span class="reading">${escapeHTML(sample.referenceValue)} ${escapeHTML(sample.unit)}</span><small>允许 ±${escapeHTML(sample.allowedDelta)}</small></div><div><span class="reading ${stateClass}">${reading ? escapeHTML(reading.measuredValue) : '--'}</span><small>${reading ? `偏差 ${escapeHTML(reading.deviation)}` : '等待录入'}</small></div><div><span class="reading ${stateClass}">${status}</span><small>${reading ? formatDate(reading.measuredAt) : '—'}</small></div></div>`;
  }).join('');
  $('#sample-table').innerHTML = `<div class="sample-row sample-row-head"><span>样本</span><span>参考值</span><span>最新读数</span><span>判定</span></div>${rows || '<p class="action-copy">尚未登记标准样本。</p>'}`;
}

function renderAudit(events, verified) {
  const list = $('#audit-list');
  $('#audit-status').textContent = verified ? 'HASH / VALID' : 'HASH / CHECK';
  $('#audit-status').className = `audit-status ${verified ? 'valid' : 'invalid'}`;
  list.innerHTML = events.length ? events.map((event) => `<div class="audit-event"><strong>${escapeHTML(actionLabel(event.action))}</strong><small>${formatDate(event.occurredAt)} · ${escapeHTML(event.actor)}</small><p>${escapeHTML(detailLabel(event.details))}</p></div>`).join('') : '<p class="action-copy">暂无审计事件。</p>';
}

function actionLabel(action) { return ({ 'session.created': '创建校准会话', 'samples.registered': '登记标准样本', 'measurement.submitted': '提交测量记录', 'quality.reviewed': '完成质量复核', 'session.sealed': '封存会话并生成证书' })[action] || action; }
function detailLabel(details) { return Object.entries(details || {}).map(([key, value]) => `${key}: ${value}`).join(' · '); }

function renderCertificate(certificate) {
  const section = $('#certificate-section');
  section.classList.toggle('hidden', !certificate);
  if (!certificate) return;
  const verification = certificate.verification || {};
  const verifiable = verification.verifiable === true && verification.status === 'verified';
  section.classList.toggle('invalid', !verifiable);
  $('#certificate-number').textContent = certificate.certificateNo;
  $('#certificate-hash').textContent = certificate.summaryHash;
  $('#certificate-sealed-by').textContent = certificate.sealedBy;
  $('#certificate-sealed-at').textContent = formatDate(certificate.sealedAt);
  $('#certificate-verification').textContent = verifiable ? '凭据可验证' : '完整性异常';
  $('#certificate-failure').textContent = verification.failureReason || '摘要或审计哈希链未通过校验';
  $('#certificate-failure').classList.toggle('hidden', verifiable);
}

function addSampleRow() {
  const row = document.createElement('div');
  row.className = 'new-sample-row';
  row.innerHTML = `<label>样本编号<input name="sampleNumber" required placeholder="REF-01"></label><label>参考值<input name="referenceValue" type="number" step="any" min="0" required></label><label>单位<input name="unit" required placeholder="ADU"></label><label>允许偏差<input name="allowedDelta" type="number" step="any" min="0.000001" required></label><label>登记人<input name="registeredBy" required placeholder="姓名"></label><button class="button remove-sample" type="button" aria-label="移除样本">×</button>`;
  row.querySelector('.remove-sample').addEventListener('click', () => row.remove());
  $('#new-sample-rows').appendChild(row);
}

async function submitNewSession(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const value = readForm(form);
  const samples = [...document.querySelectorAll('#new-sample-rows .new-sample-row')].map((row) => { const sample = readForm(row); return { ...sample, referenceValue: Number(sample.referenceValue), allowedDelta: Number(sample.allowedDelta) }; });
  delete value.sampleNumber;
  try { const body = await api('/api/sessions', { method: 'POST', body: JSON.stringify({ ...value, samples }) }); $('#session-dialog').close(); form.reset(); $('#new-sample-rows').innerHTML = ''; showToast('校准会话已创建'); await loadSessions(body.session.id); } catch (error) { showToast(error.message, 'error'); }
}

function setupDialog() {
  const dialog = $('#session-dialog');
  $('#new-session-button').addEventListener('click', () => { $('#new-sample-rows').innerHTML = ''; dialog.showModal(); });
  $('#close-dialog').addEventListener('click', () => dialog.close());
  $('#cancel-dialog').addEventListener('click', () => dialog.close());
  $('#add-sample-row').addEventListener('click', addSampleRow);
  $('#session-form').addEventListener('submit', submitNewSession);
}

function tickClock() { $('#clock').textContent = new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(new Date()); }

$('#session-search').addEventListener('input', renderSessions);
setupDialog();
tickClock();
setInterval(tickClock, 1000);
loadSessions().catch((error) => showToast(error.message, 'error'));
