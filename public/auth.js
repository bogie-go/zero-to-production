// account() is the header's signed-in state, shared by every page: ask
// GET /me who the saved bearer token is for. A guest's token (or none) is
// "not signed in", the same answer the payment page acts on. Loaded before
// Alpine, which finds the function when x-data="account()" is evaluated.
function account() {
  return {
    email: '',
    ready: false,
    get loginUrl() {
      return '/login.html?next=' + encodeURIComponent(location.pathname + location.search);
    },
    async load() {
      try {
        const token = localStorage.getItem('cartToken') || '';
        if (token) {
          const res = await fetch('/me', { headers: { Authorization: 'Bearer ' + token } });
          if (res.ok) this.email = (await res.json()).email;
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
