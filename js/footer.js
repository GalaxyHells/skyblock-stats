// footer.js - carrega o footer em todas as páginas

fetch('footer.html')
  .then(response => response.text())
  .then(html => {
    const placeholder = document.getElementById('site-footer');
    if (placeholder) {
      placeholder.innerHTML = html;
    }
  })
  .catch(err => console.error('Erro ao carregar footer:', err));