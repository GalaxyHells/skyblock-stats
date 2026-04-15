// Carrega o header compartilhado em todas as páginas
(async function loadSharedHeader() {
    const placeholder = document.getElementById('site-header');
    if (!placeholder) return;

    try {
        const basePath = window.location.pathname.includes('/assets/html/') ? '' : 'assets/html/';
        const res = await fetch(basePath + 'header.html', { cache: 'no-cache' });
        if (!res.ok) throw new Error('Erro ao carregar header');
        const html = await res.text();
        placeholder.innerHTML = html;

        if (window.location.pathname.includes('/assets/html/')) {
            placeholder.querySelectorAll('a').forEach(link => {
                const href = link.getAttribute('href');
                if (!href || href.startsWith('http') || href.startsWith('#')) return;

                if (href === 'index.html') {
                    link.setAttribute('href', '../index.html');
                } else if (href === 'assets/html/profiles.html') {
                    link.setAttribute('href', 'profiles.html');
                } else if (href === 'assets/html/tops.html') {
                    link.setAttribute('href', 'tops.html');
                }
            });
        }
    } catch (e) {
        console.error('Falha ao carregar header compartilhado:', e);
    }
})();
