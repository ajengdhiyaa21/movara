(function () {
  var $ = function (s) { return document.querySelector(s); };
  var scanTimer = null;
  var filterTimer = null;
  var lastLookupValue = '';
  var pendingDeleteForm = null;

  function focusAfterSwap(container) {
    if (container.querySelector('[data-found]')) {
      var q = $('#qty');
      if (q) { q.focus(); q.select(); }
    } else {
      var b = $('#barcode');
      if (b) { b.focus(); b.select(); }
    }
  }

  function lookupBarcode(input) {
    var value = input.value.trim();
    if (value === '' || value === lastLookupValue) return;
    lastLookupValue = value;
    htmx.trigger(input, 'lookup');
  }

  function refreshDirectionToggle() {
    var input = $('#direction');
    if (!input) return;
    document.querySelectorAll('[data-direction]').forEach(function (button) {
      var active = button.dataset.direction === input.value;
      var blue = active && button.dataset.direction === 'counter_to_market';
      var green = active && button.dataset.direction === 'market_to_counter';
      button.style.borderColor = blue ? '#3b82f6' : green ? '#10b981' : '#e2e8f0';
      button.style.backgroundColor = blue ? '#eff6ff' : green ? '#ecfdf5' : '#ffffff';
      button.style.boxShadow = active ? '0 0 0 2px ' + (blue ? 'rgba(59,130,246,.18)' : 'rgba(16,185,129,.18)') : 'none';
      button.setAttribute('aria-pressed', active ? 'true' : 'false');
    });
  }

  document.addEventListener('keydown', function (e) {
    // Enter di kolom barkode = cari produk (bukan submit form).
    if (e.key === 'Enter' && e.target && e.target.id === 'barcode') {
      e.preventDefault();
      lookupBarcode(e.target);
    }
    if (e.key === 'Escape') {
      var reset = $('#btn-reset');
      if (reset) reset.click();
    }
  });

  document.addEventListener('click', function (e) {
    var modal = $('#delete-modal');
    if (e.target.closest('[data-delete-cancel]')) {
      if (modal) { modal.classList.add('hidden'); modal.classList.remove('flex'); }
      pendingDeleteForm = null;
      return;
    }
    if (e.target.closest('[data-delete-confirm]') && pendingDeleteForm) {
      pendingDeleteForm.submit();
      return;
    }
    var historyTab = e.target.closest('[data-history-tab]');
    if (historyTab) {
      var selected = historyTab.dataset.historyTab;
      document.querySelectorAll('[data-history-tab]').forEach(function (button) {
        var active = button === historyTab;
        button.classList.toggle('history-tab-active', active);
        button.classList.toggle('text-blue-700', active);
        button.classList.toggle('text-slate-500', !active);
        button.setAttribute('aria-selected', active ? 'true' : 'false');
      });
      document.querySelectorAll('[data-history-tab-panel]').forEach(function (panel) {
        panel.classList.toggle('hidden', panel.dataset.historyTabPanel !== selected);
      });
      return;
    }
    var direction = e.target.closest('[data-direction]');
    if (direction) {
      var input = $('#direction');
      if (input) input.value = direction.dataset.direction;
      refreshDirectionToggle();
      return;
    }
    var step = e.target.closest('[data-qty-step]');
    if (step) {
      var qty = $('#qty');
      var next = (parseInt(qty.value, 10) || 1) + parseInt(step.dataset.qtyStep, 10);
      qty.value = Math.min(999, Math.max(1, next));
      return;
    }
    var fill = e.target.closest('[data-fill-user]');
    if (fill) {
      $('#username').value = fill.dataset.fillUser;
      $('#password').value = fill.dataset.fillPass;
      return;
    }
    if (e.target.closest('[data-toggle-password]')) {
      var p = $('#password');
      p.type = p.type === 'password' ? 'text' : 'password';
    }
  });

  function openConfirmModal(form) {
    pendingDeleteForm = form;
    var modal = $('#delete-modal');
    var title = $('#delete-modal-title');
    var message = $('#delete-modal-message');
    var icon = $('#confirm-modal-icon');
    var confirmButton = $('#confirm-modal-submit');
    var action = (form.getAttribute('action') || '').toLowerCase();
    var isDelete = action.indexOf('/delete') !== -1;
    var isStatus = action.indexOf('/deactivate') !== -1 || action.indexOf('/toggle') !== -1;
    var isEdit = action.indexOf('/edit') !== -1 || action === '/profile' || action === '/password';
    if (title) title.textContent = form.dataset.confirmTitle || (isDelete ? 'Hapus data?' : isStatus ? 'Ubah status akun?' : isEdit ? 'Simpan perubahan?' : 'Konfirmasi tindakan');
    if (message) message.textContent = form.dataset.confirmMessage || (isDelete ? 'Data mutasi akan dihapus dan tidak dapat dikembalikan.' : isStatus ? (action.indexOf('/deactivate') !== -1 ? 'Akun atau data ini akan dinonaktifkan dan tidak dapat digunakan sementara.' : 'Akun atau data ini akan diaktifkan kembali.') : isEdit ? 'Pastikan data yang diubah sudah benar sebelum disimpan.' : 'Apakah Anda yakin ingin melanjutkan?');
    if (icon) {
      icon.className = 'grid h-11 w-11 shrink-0 place-items-center rounded-2xl ' + (isDelete ? 'bg-red-50 text-red-600' : isStatus ? 'bg-amber-50 text-amber-600' : 'bg-blue-50 text-blue-600');
    }
    if (confirmButton) {
      confirmButton.textContent = isDelete ? 'Ya, hapus' : isEdit ? 'Ya, simpan' : 'Ya, lanjutkan';
      confirmButton.className = 'rounded-xl px-4 py-2.5 text-sm font-semibold text-white ' + (isDelete ? 'bg-red-600 hover:bg-red-700' : isStatus ? 'bg-amber-500 hover:bg-amber-600' : 'bg-blue-600 hover:bg-blue-700');
    }
    if (modal) { modal.classList.remove('hidden'); modal.classList.add('flex'); }
  }

  // Capture dijalankan sebelum inline onsubmit lama, sehingga dialog browser
  // "localhost says" tidak lagi muncul pada halaman yang belum dimigrasikan.
  document.addEventListener('submit', function (e) {
    var form = e.target;
    if (!form.matches('[data-confirm-action], [data-confirm-delete], form[action*="/deactivate"], form[action*="/toggle"], form[action*="/edit"], form[action="/profile"], form[action="/password"]')) return;
    e.preventDefault();
    e.stopPropagation();
    openConfirmModal(form);
  }, true);

  document.addEventListener('submit', function (e) {
    if (!e.target.matches('[data-confirm-delete]')) return;
    e.preventDefault();
    openConfirmModal(e.target);
  });

  document.addEventListener('keydown', function (e) {
    if (e.key !== 'Escape' || !pendingDeleteForm) return;
    var modal = $('#delete-modal');
    if (modal) { modal.classList.add('hidden'); modal.classList.remove('flex'); }
    pendingDeleteForm = null;
  });

  document.addEventListener('input', function (e) {
    var filterInput = e.target && e.target.closest('form[method="get"] input[type="search"], form[method="get"] input[name="q"]');
    if (filterInput && !filterInput.hasAttribute('hx-get')) {
      clearTimeout(filterTimer);
      filterTimer = setTimeout(function () {
        var form = filterInput.closest('form');
        if (form) form.requestSubmit();
      }, 350);
    }
    if (!e.target || e.target.id !== 'barcode') return;
    clearTimeout(scanTimer);
    var delay = 180;
    scanTimer = setTimeout(function () { lookupBarcode(e.target); }, delay);
  });

  document.addEventListener('change', function (e) {
    var control = e.target && e.target.closest('form[method="get"] select, form[method="get"] input[type="date"]');
    if (!control || control.hasAttribute('hx-get')) return;
    var form = control.closest('form');
    if (form) form.requestSubmit();
  });

  document.body.addEventListener('htmx:afterSwap', function (e) {
    var id = e.detail.target && e.detail.target.id;
    if (id === 'scan-panel') {
      lastLookupValue = '';
      refreshDirectionToggle();
      focusAfterSwap(e.detail.target);
      return;
    }
    if (id === 'product-fields') {
      focusAfterSwap(e.detail.target);
    }
  });

  var barcode = $('#barcode');
  if (barcode) barcode.focus();
  refreshDirectionToggle();

  // Paksa validasi ulang saat browser mengembalikan halaman dari back-forward cache.
  // Ini mencegah tampilan akun lama muncul setelah logout.
  window.addEventListener('pageshow', function (event) {
    if (event.persisted) window.location.reload();
  });
})();

(function () {
  function modal(id, show) { var el = document.getElementById(id); if (!el) return; el.classList.toggle('hidden', !show); el.classList.toggle('flex', show); }
  document.addEventListener('click', function (e) {
    var edit = e.target.closest('[data-mutation-edit]');
    if (edit) { document.getElementById('mutation-edit-id').value = edit.dataset.id; document.getElementById('mutation-edit-qty').value = edit.dataset.qty; modal('mutation-edit-modal', true); return; }
    var history = e.target.closest('[data-mutation-history]');
    if (history) { modal('mutation-history-modal', true); var box = document.getElementById('mutation-history-content'); box.innerHTML = '<p class="text-sm text-slate-500">Memuat riwayat...</p>'; fetch('/mutasi/today/' + history.dataset.id + '/edit-history', { headers: { 'X-Requested-With': 'XMLHttpRequest' } }).then(function (r) { return r.text(); }).then(function (html) { box.innerHTML = html; }); return; }
    if (e.target.closest('[data-mutation-modal-close]')) { modal('mutation-edit-modal', false); modal('mutation-history-modal', false); }
  });
  document.body.addEventListener('htmx:afterRequest', function (e) { if (e.detail && e.detail.xhr && e.detail.xhr.getResponseHeader('HX-Trigger') === 'mutation-edit-saved') modal('mutation-edit-modal', false); });
})();
