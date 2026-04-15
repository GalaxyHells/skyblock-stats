// Carrega o header compartilhado em todas as páginas
(async function loadSharedHeader() {
    const placeholder = document.getElementById('site-header');
    if (!placeholder) return;

    try {
        const res = await fetch('../html/header.html', { cache: 'no-cache' });
        if (!res.ok) throw new Error('Erro ao carregar header');
        const html = await res.text();
        placeholder.innerHTML = html;
    } catch (e) {
        console.error('Falha ao carregar header compartilhado:', e);
    }
})();
