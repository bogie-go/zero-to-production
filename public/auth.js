// account() is the header's signed-in state, shared by every page: ask
// GET /me who the saved bearer token is for. A guest's token (or none) is
// "not signed in", the same answer the payment page acts on. Loaded before
// Alpine, which finds the function when x-data="account()" is evaluated.
function account() {
  return {
    email: '',
    role: '',
    get isStaff() { return this.role === 'staff'; },
    ready: false,
    get loginUrl() {
      return '/login.html?next=' + encodeURIComponent(location.pathname + location.search);
    },
    async load() {
      try {
        const token = localStorage.getItem('cartToken') || '';
        if (token) {
          const res = await fetch('/me', { headers: { Authorization: 'Bearer ' + token } });
          if (res.ok) {
            const me = await res.json();
            this.email = me.email;
            this.role = me.role;
          }
        }
      } catch (e) {}
      this.ready = true;
    },
    logout() {
      try { localStorage.removeItem('cartToken'); } catch (e) {}
      location.href = '/';
    },
  };
}
