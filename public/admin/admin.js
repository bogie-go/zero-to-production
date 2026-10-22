// adminShell() is what every admin page shares: the sidebar's links (the
// one place they are listed), who is signed in, and the staff gate. A page
// spreads it into its own component:
//
//   function adminPage() { return { ...adminShell(), async load() { if (!(await this.guard())) return; ... } } }
//
// The gate is a convenience, not the security: every /admin API call is
// checked again by the server (RequireStaff), so skipping the redirect
// gets a visitor nothing.
function adminShell() {
  return {
    email: '',
    loading: true,
    error: null,
    links: [
      { label: 'Dashboard', href: '/admin/' },
      { label: 'Products', href: '/admin/products.html', soon: true },
      { label: 'Categories', href: '/admin/categories.html', soon: true },
      { label: 'Price rules', href: '/admin/price_rules.html', soon: true },
    ],
    isActive(link) {
      const here = location.pathname.replace(/index\.html$/, '');
      return here === link.href;
    },
    token() {
      try { return localStorage.getItem('cartToken') || ''; } catch (e) { return ''; }
    },
    api(url, options = {}) {
      const headers = new Headers(options.headers || {});
      if (this.token()) headers.set('Authorization', 'Bearer ' + this.token());
      return fetch(url, { ...options, headers });
    },
    // guard sends anyone who is not staff away and reports whether to go on:
    // not signed in goes to login (and back here); a customer goes home.
    async guard() {
      try {
        const me = await this.api('/me');
        if (me.status === 401 || me.status === 403) {
          location.href = '/login.html?next=' + encodeURIComponent(location.pathname + location.search);
          return false;
        }
        if (!me.ok) throw new Error('status ' + me.status);
        const account = await me.json();
        if (account.role !== 'staff') {
          location.href = '/';
          return false;
        }
        this.email = account.email;
        return true;
      } catch (e) {
        this.error = 'Could not load the admin area: ' + e.message;
        this.loading = false;
        return false;
      }
    },
    logout() {
      try { localStorage.removeItem('cartToken'); } catch (e) {}
      location.href = '/';
    },
  };
}
