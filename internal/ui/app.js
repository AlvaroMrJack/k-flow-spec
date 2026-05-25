(function () {
  'use strict';

  const REFRESH_INTERVAL = 30000;
  let refreshTimer = null;

  const els = {
    resultsBody: document.getElementById('results-body'),
    specsList: document.getElementById('specs-list'),
    specsCount: document.getElementById('specs-count'),
    resultsCount: document.getElementById('results-count'),
    statTotal: document.getElementById('stat-total'),
    statPassed: document.getElementById('stat-passed'),
    statFailed: document.getElementById('stat-failed'),
    statRate: document.getElementById('stat-rate'),
    statDuration: document.getElementById('stat-duration'),
    footerTimestamp: document.getElementById('footer-timestamp'),
    connectionStatus: document.getElementById('connection-status'),
    autoRefresh: document.getElementById('auto-refresh'),
  };

  let passRateChart = null;

  function fmtDuration(ms) {
    if (ms < 1000) return ms + ' ms';
    if (ms < 60000) return (ms / 1000).toFixed(2) + ' s';
    return (ms / 60000).toFixed(2) + ' min';
  }

  function fmtTimestamp(iso) {
    try {
      const d = new Date(iso);
      return d.toLocaleString();
    } catch {
      return iso;
    }
  }

  function escapeHtml(text) {
    if (text == null) return '';
    const div = document.createElement('div');
    div.appendChild(document.createTextNode(String(text)));
    return div.innerHTML;
  }

  function updateStats(results) {
    const total = results.length;
    const passed = results.filter(function (r) { return r.passed; }).length;
    const failed = total - passed;
    const totalMs = results.reduce(function (sum, r) { return sum + r.duration_ms; }, 0);
    const rate = total > 0 ? ((passed / total) * 100).toFixed(1) + '%' : '--';

    els.statTotal.textContent = total;
    els.statPassed.textContent = passed;
    els.statFailed.textContent = failed;
    els.statRate.textContent = rate;
    els.statDuration.textContent = fmtDuration(totalMs);
    els.resultsCount.textContent = total;

    updateChart(passed, failed);
  }

  function renderResults(results) {
    if (results.length === 0) {
      els.resultsBody.innerHTML =
        '<tr class="empty-row"><td colspan="6">No results yet. Run some specs to see them here.</td></tr>';
      return;
    }

    var html = '';
    for (var i = 0; i < results.length; i++) {
      var r = results[i];
      var statusClass = r.passed ? 'status-passed' : 'status-failed';
      var statusText = r.passed ? 'PASS' : 'FAIL';
      var errors = r.errors || [];
      var errorSummary = '';
      if (errors.length > 0) {
        errorSummary =
          '<span class="error-count">' +
          errors.length +
          '</span> ' +
          escapeHtml(errors[0].message || errors[0].type || 'error').substring(0, 60);
        if (errors[0].message && errors[0].message.length > 60) errorSummary += '&hellip;';
      } else {
        errorSummary = '<span class="text-muted">&mdash;</span>';
      }

      html +=
        '<tr class="result-row ' + statusClass + '-row">' +
        '<td class="cell-spec">' + escapeHtml(r.name) + '</td>' +
        '<td class="cell-workflow">' + escapeHtml(r.workflow_id) + '</td>' +
        '<td><span class="status-badge ' + statusClass + '">' + statusText + '</span></td>' +
        '<td class="cell-duration">' + fmtDuration(r.duration_ms) + '</td>' +
        '<td class="cell-errors">' + errorSummary + '</td>' +
        '<td class="cell-timestamp">' + fmtTimestamp(r.timestamp) + '</td>' +
        '</tr>';
    }
    els.resultsBody.innerHTML = html;
  }

  function renderSpecs(specs) {
    els.specsCount.textContent = specs.length;

    if (specs.length === 0) {
      els.specsList.innerHTML = '<li class="empty-item">No spec files found.</li>';
      return;
    }

    var html = '';
    for (var i = 0; i < specs.length; i++) {
      var s = specs[i];
      html +=
        '<li class="spec-item">' +
        '<span class="spec-name">' + escapeHtml(s.name) + '</span>' +
        '<span class="spec-workflow">' + escapeHtml(s.workflow) + '</span>' +
        '<span class="spec-file">' + escapeHtml(s.file) + '</span>' +
        '</li>';
    }
    els.specsList.innerHTML = html;
  }

  function updateChart(passed, failed) {
    var canvas = document.getElementById('pass-rate-chart');
    if (!canvas) return;

    if (passed === 0 && failed === 0) {
      if (passRateChart) {
        passRateChart.destroy();
        passRateChart = null;
      }
      return;
    }

    var ctx = canvas.getContext('2d');

    if (passRateChart) {
      passRateChart.data.datasets[0].data = [passed, failed];
      passRateChart.update();
      return;
    }

    passRateChart = new Chart(ctx, {
      type: 'doughnut',
      data: {
        labels: ['Passed', 'Failed'],
        datasets: [
          {
            data: [passed, failed],
            backgroundColor: ['#2ecc71', '#e74c3c'],
            borderColor: ['#1a1a2e', '#1a1a2e'],
            borderWidth: 2,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: true,
        cutout: '65%',
        plugins: {
          legend: {
            position: 'bottom',
            labels: {
              color: '#e0e0e0',
              padding: 12,
              font: { family: 'system-ui, sans-serif', size: 12 },
            },
          },
        },
      },
    });
  }

  function updateTimestamp() {
    els.footerTimestamp.textContent = new Date().toLocaleString();
  }

  function setConnectionStatus(ok) {
    var el = els.connectionStatus;
    el.textContent = ok ? 'Connected' : 'Disconnected';
    el.className = 'status-badge ' + (ok ? 'status-passed' : 'status-failed');
  }

  function fetchJSON(url) {
    return fetch(url)
      .then(function (res) {
        if (!res.ok) throw new Error('HTTP ' + res.status);
        return res.json();
      })
      .then(function (body) { return body.data || []; });
  }

  function loadData() {
    var failed = false;

    return Promise.all([fetchJSON('/api/results'), fetchJSON('/api/specs')])
      .then(function (results) {
        var resData = results[0];
        var specData = results[1];

        renderResults(resData);
        renderSpecs(specData);
        updateStats(resData);
        updateTimestamp();
        setConnectionStatus(true);
      })
      .catch(function (err) {
        console.error('Dashboard load error:', err);
        setConnectionStatus(false);
      });
  }

  function startAutoRefresh() {
    stopAutoRefresh();
    if (els.autoRefresh.checked) {
      refreshTimer = setInterval(loadData, REFRESH_INTERVAL);
    }
  }

  function stopAutoRefresh() {
    if (refreshTimer) {
      clearInterval(refreshTimer);
      refreshTimer = null;
    }
  }

  els.autoRefresh.addEventListener('change', function () {
    if (els.autoRefresh.checked) {
      startAutoRefresh();
    } else {
      stopAutoRefresh();
    }
  });

  loadData().then(startAutoRefresh);
})();
