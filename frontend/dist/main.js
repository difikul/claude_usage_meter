import { Call, Events, Window } from '/wails/runtime.js';

document.addEventListener('DOMContentLoaded', async () => {
  // DOM refs
  const els = {
    bar5h: document.getElementById('bar-5h'),
    barWeekly: document.getElementById('bar-weekly'),
    barSonnet: document.getElementById('bar-sonnet'),
    percent5h: document.getElementById('percent-5h'),
    percentWeekly: document.getElementById('percent-weekly'),
    percentSonnet: document.getElementById('percent-sonnet'),
    cost5h: document.getElementById('cost-5h'),
    costWeekly: document.getElementById('cost-weekly'),
    costSonnet: document.getElementById('cost-sonnet'),
    reset5h: document.getElementById('reset-5h'),
    resetWeekly: document.getElementById('reset-weekly'),
    resetSonnet: document.getElementById('reset-sonnet'),
    tokens5h: document.getElementById('tokens-5h'),
    tokensWeekly: document.getElementById('tokens-weekly'),
    tierInfo: document.getElementById('tier-info'),
    lastUpdate: document.getElementById('last-update'),
  };

  let lastData = null;

  function formatTokens(n) {
    if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
    if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
    return n.toString();
  }

  function formatCost(usd) {
    if (usd >= 100) return '$' + usd.toFixed(0);
    if (usd >= 10) return '$' + usd.toFixed(1);
    return '$' + usd.toFixed(2);
  }

  function formatBudget(usd) {
    if (usd >= 100) return '$' + usd.toFixed(0);
    return '$' + usd.toFixed(0);
  }

  function thresholdClass(percent) {
    if (percent >= 90) return 'danger';
    if (percent >= 70) return 'warn';
    return '';
  }

  function formatResetTime(isoStr) {
    if (!isoStr) return 'No active window';

    const resetDate = new Date(isoStr);
    const now = new Date();
    const diffMs = resetDate - now;

    if (diffMs <= 0) return 'Resetting soon...';

    const diffMin = Math.floor(diffMs / 60000);
    const diffH = Math.floor(diffMin / 60);
    const remMin = diffMin % 60;

    if (diffH < 24) {
      if (diffH === 0) return 'Resets in ' + remMin + 'm';
      return 'Resets in ' + diffH + 'h ' + remMin + 'm';
    }

    const opts = { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' };
    return 'Resets ' + resetDate.toLocaleDateString(undefined, opts);
  }

  function renderBar(barEl, percentEl, percent) {
    const cls = thresholdClass(percent);
    barEl.style.width = Math.min(percent, 100) + '%';
    barEl.className = 'progress-bar' + (cls ? ' ' + cls : '');
    percentEl.className = 'percent' + (cls ? ' ' + cls : '');
  }

  function renderTokens(el, win) {
    el.innerHTML =
      '<span class="token-item">In: ' + formatTokens(win.input_tokens) + '</span>' +
      '<span class="token-item">Out: ' + formatTokens(win.output_tokens) + '</span>' +
      '<span class="token-item">Cache R: ' + formatTokens(win.cache_read_tokens) + '</span>' +
      '<span class="token-item">Cache W: ' + formatTokens(win.cache_create_tokens) + '</span>';
  }

  function renderSection(info, barEl, percentEl, costEl, resetEl, tokensEl, estimated) {
    const pct = info.percent;
    const overBudget = info.cost_usd > info.budget_usd;
    const prefix = estimated ? '~' : '';

    renderBar(barEl, percentEl, pct);
    percentEl.textContent = prefix + (overBudget ? '100%+' : Math.round(pct) + '%') + ' used';
    costEl.textContent = formatCost(info.cost_usd) + ' / ' + formatBudget(info.budget_usd);
    resetEl.textContent = formatResetTime(info.reset_ts);

    if (tokensEl) {
      renderTokens(tokensEl, info.window);
    }
  }

  function renderAll(data) {
    const noApi = !data.api_available;
    renderSection(data.five_hour, els.bar5h, els.percent5h, els.cost5h, els.reset5h, els.tokens5h, noApi);
    renderSection(data.weekly, els.barWeekly, els.percentWeekly, els.costWeekly, els.resetWeekly, els.tokensWeekly, noApi);
    renderSection(data.weekly_sonnet, els.barSonnet, els.percentSonnet, els.costSonnet, els.resetSonnet, null, noApi);

    const tierDisplay = data.tier_name.replace('default_claude_', '');
    const statusSuffix = data.rate_limit_status === 'rate_limited' ? ' [RATE LIMITED]' : '';
    els.tierInfo.textContent = 'Tier: ' + tierDisplay + statusSuffix;
  }

  function updateResetTimes() {
    if (!lastData) return;
    els.reset5h.textContent = formatResetTime(lastData.five_hour.reset_ts);
    els.resetWeekly.textContent = formatResetTime(lastData.weekly.reset_ts);
    els.resetSonnet.textContent = formatResetTime(lastData.weekly_sonnet.reset_ts);
  }

  async function refresh() {
    document.querySelector('.widget').classList.add('loading');
    try {
      const data = await Call.ByName("main.UsageService.GetUsage");
      lastData = data;
      renderAll(data);
      els.lastUpdate.textContent = 'Updated ' + new Date().toLocaleTimeString();

      // Resize window to fit content
      const widget = document.querySelector('.widget');
      const height = widget.offsetHeight;
      await Window.SetSize(340, height);
    } catch (e) {
      els.lastUpdate.textContent = 'Error: ' + e;
    }
    document.querySelector('.widget').classList.remove('loading');
  }

  // Initial load
  await refresh();

  // Listen for auto-refresh events from backend
  Events.On('usage-updated', () => {
    refresh();
  });

  // Countdown timer — update reset times every 30s
  setInterval(updateResetTimes, 30000);

  // Refresh button
  document.getElementById('btn-refresh').addEventListener('click', () => {
    refresh();
  });

  // Hide to tray button
  document.getElementById('btn-hide').addEventListener('click', async () => {
    await Window.Hide();
  });
});
