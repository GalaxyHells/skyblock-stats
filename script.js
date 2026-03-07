// Função para copiar o IP
const ipSection = document.getElementById('copy-ip');
const ipText = "play.seuservidor.com.br";

ipSection.addEventListener('click', () => {
    navigator.clipboard.writeText(ipText).then(() => {
        const btn = ipSection.querySelector('button');
        btn.innerText = "Copiado!";
        btn.style.background = "#27ae60";
        
        setTimeout(() => {
            btn.innerText = "Copiar IP";
            btn.style.background = "#9b59b6";
        }, 2000);
    });
});

// Simulação de Player Count (Aqui você usaria uma API como mcapi.us ou mcstatus.io)
document.getElementById('player-count').innerText = "1,240";