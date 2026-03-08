const API_KEY = "UNLIMITED_KEY"; // Chave fornecida no seu exemplo
const container = document.getElementById('profiles-container');
const searchBtn = document.getElementById('search-btn');
const searchInput = document.getElementById('player-search');

// Objeto para traduzir as chaves da API para nomes amigáveis
const skillNames = {
    "FORAGING_SKILL": "Forrageamento",
    "COMBAT_SKILL": "Combate",
    "MINING_SKILL": "Mineração",
    "FISHING_SKILL": "Pescaria",
    "FARMING_SKILL": "Herbalismo",
    "ENCHANTING_SKILL": "Encantamento"
};

const maxSkillLevel = 25; // Definindo um nível máximo para as barras de progresso

// Detecta se estás a testar localmente ou no Render
const PROXY_URL = window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1"
    ? "http://localhost:8080"
    : "https://skyblock-stats.onrender.com";

async function fetchPlayerProfile(username) {
    if (!username) return;
    container.innerHTML = '<div class="loader">Buscando...</div>';

    try {
        const response = await fetch(`${PROXY_URL}/profile?id=${username.toLowerCase()}`);
        if (!response.ok) throw new Error("Não encontrado");

        const data = await response.json();
        const profile = data.profiles[data.activeProfile];
        
        renderProfileCard(profile, username);
    } catch (error) {
        container.innerHTML = `<div class="error">${error.message}</div>`;
    }
}

function renderProfileCard(p, username) {
    // Cálculo simples de nível (Skills + Coleções)
    const skills = p.data_model?.skills_model?.level ?? {};
    const cols = p.data_model?.collections_model?.collection_level ?? {};
    
    const totalLevel = Object.values(skills).reduce((a, b) => a + b, 0) + 
                       Object.values(cols).reduce((a, b) => a + b, 0);
    
    const sbLevel = Math.floor(totalLevel / 10);

    // Atualiza o Rank no servidor
    updateLeaderboardInGo(username, sbLevel);

    container.innerHTML = `
        <div class="profile-card-detail">
            <h2>${username}</h2>
            <p class="sb-level-number">Nível SB: ${sbLevel}</p>
            <p>Dinheiro: ${p.data_model?.stats_model?.purse?.toLocaleString() ?? 0}</p>
        </div>
    `;
}

searchBtn.addEventListener('click', () => fetchPlayerProfile(searchInput.value));

// Permitir busca ao apertar "Enter"
searchInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') fetchPlayerProfile(searchInput.value);
});

// Transforma "POTATO_ITEM" ou "EMERALD" em "Potato" e "Emerald"
function formatCollectionName(name) {
    return name.toLowerCase()
               .replace(/_/g, ' ')
               .replace(/\b\w/g, letra => letra.toUpperCase())
               .replace(' Item', '') // Remove o sufixo "Item" de alguns itens
               .replace(' Raw', ''); // Remove o sufixo "Raw"
}

// // 1. Função para salvar no histórico (sem duplicatas e limite de 5 nomes)
// function saveToHistory(username) {
//     let history = JSON.parse(localStorage.getItem('searchHistory')) || [];
    
//     // Remove o nome se já existir (para movê-lo para o topo)
//     history = history.filter(name => name !== username);
    
//     // Adiciona no início da lista
//     history.unshift(username);
    
//     // Mantém apenas os 5 últimos
//     if (history.length > 5) history.pop();
    
//     localStorage.setItem('searchHistory', JSON.stringify(history));
//     renderHistory();
// }

// // 2. Função para exibir os botões de histórico na tela
// function renderHistory() {
//     const historyContainer = document.getElementById('recent-searches'); // Você precisará deste ID no HTML
//     const history = JSON.parse(localStorage.getItem('searchHistory')) || [];
    
//     if (history.length === 0) {
//         historyContainer.innerHTML = '';
//         return;
//     }

//     historyContainer.innerHTML = `
//         <span style="color: #888; font-size: 0.8rem;">Recentes:</span>
//         ${history.map(name => `
//             <button class="history-btn" onclick="fetchPlayerProfile('${name}')">${name}</button>
//         `).join('')}
//     `;
// }

// // 3. Modifique sua função fetchPlayerProfile para chamar o saveToHistory no sucesso
// async function fetchPlayerProfile(username) {
//     const cleanUsername = username.trim().toLowerCase();
//     if (!cleanUsername) return;

//     // ... (seu código de fetch existente) ...

//     try {
//         const response = await fetch(`https://skyblock-stats.onrender.com/profile?id=${cleanUsername}`);
//         // ...
        
//         // Se a busca deu certo, salvamos no histórico
//         saveToHistory(cleanUsername);
        
//         renderProfileCard(profile, cleanUsername);
//     } catch (error) {
//         // ...
//     }
// }

// // Chame a renderização inicial ao carregar a página
// document.addEventListener('DOMContentLoaded', renderHistory);

// 1. Busca e desenha o Leaderboard na tela
// --- LEADERBOARD ---

async function fetchLeaderboard() {
    const list = document.getElementById('leaderboard-list');
    if (!list) return;

    try {
        const res = await fetch(`${PROXY_URL}/top`);
        const topPlayers = await res.json();
        
        if (!topPlayers || topPlayers.length === 0) {
            list.innerHTML = '<li class="lb-item">Nenhum jogador no Rank.</li>';
            return;
        }

        list.innerHTML = topPlayers.map((p, index) => {
            const medal = index === 0 ? "🥇" : index === 1 ? "🥈" : index === 2 ? "🥉" : `#${index + 1}`;
            return `
                <li class="lb-item">
                    <span>${medal} <b>${p.username}</b></span>
                    <strong>Lvl ${p.level}</strong>
                </li>
            `;
        }).join('');
    } catch (e) {
        list.innerHTML = '<li style="color:red">Erro ao carregar Rank</li>';
    }
}

// 2. Envia o nível calculado para o servidor Go
async function updateLeaderboardInGo(username, level) {
    try {
        await fetch(`${PROXY_URL}/update-top`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, level })
        });
        fetchLeaderboard(); // Recarrega a lista após atualizar
    } catch (e) { console.error("Erro ao atualizar rank:", e); }
}
// 3. Atualize a inicialização para carregar o rank quando a página abrir
document.addEventListener('DOMContentLoaded', () => {
    renderHistory();
    fetchLeaderboard(); // <-- CHAMA O RANKING AQUI
    // ... resto dos seus botões ...
});

// Inicialização
document.addEventListener('DOMContentLoaded', () => {
    fetchLeaderboard();
    
    document.getElementById('search-btn').addEventListener('click', () => {
        fetchPlayerProfile(document.getElementById('player-search').value);
    });
});