// Configurações
const PROXY_URL = "https://skyblock-stats.onrender.com"; 
const container = document.getElementById('profiles-container');

// Tradução das Skills
const skillNames = {
    "FORAGING_SKILL": "Forrageamento",
    "COMBAT_SKILL": "Combate",
    "MINING_SKILL": "Mineração",
    "FISHING_SKILL": "Pescaria",
    "FARMING_SKILL": "Herbalismo",
    "ENCHANTING_SKILL": "Encantamento"
};

// --- FUNÇÕES DE UTILIDADE ---

function formatCollectionName(name) {
    return name.toLowerCase()
               .replace(/_/g, ' ')
               .replace(/\b\w/g, l => l.toUpperCase())
               .replace(' Item', '')
               .replace(' Raw', '');
}

// --- SISTEMA DE HISTÓRICO ---

function saveToHistory(username) {
    let history = JSON.parse(localStorage.getItem('searchHistory')) || [];
    history = history.filter(name => name !== username); // Remove duplicata
    history.unshift(username); // Adiciona no topo
    if (history.length > 5) history.pop(); // Limite de 5
    localStorage.setItem('searchHistory', JSON.stringify(history));
    renderHistory();
}

function renderHistory() {
    const historyContainer = document.getElementById('recent-searches');
    if (!historyContainer) return;

    const history = JSON.parse(localStorage.getItem('searchHistory')) || [];
    if (history.length === 0) {
        historyContainer.innerHTML = '';
        return;
    }

    historyContainer.innerHTML = `
        <span style="color: #888; font-size: 0.8rem;">Recentes:</span>
        ${history.map(name => `
            <button class="history-btn" onclick="fetchPlayerProfile('${name}')">${name}</button>
        `).join('')}
    `;
}

// --- MOTOR DE BUSCA ---

async function fetchPlayerProfile(username) {
    if (!username) return;
    const cleanUsername = username.trim().toLowerCase();
    
    container.innerHTML = '<div class="loader">Consultando proxy...</div>';

    try {
        const response = await fetch(`${PROXY_URL}/profile?id=${cleanUsername}`);
        if (!response.ok) throw new Error("Jogador não encontrado");

        const data = await response.json();
        const activeIdx = data.activeProfile;
        const profile = data.profiles[activeIdx];

        // Se a busca deu certo, salva no histórico e renderiza
        saveToHistory(cleanUsername);
        renderProfileCard(profile, cleanUsername);
        
    } catch (error) {
        container.innerHTML = `<div class="error">Erro: ${error.message}</div>`;
    }
}

// --- RENDERIZAÇÃO DO CARD ---

function renderProfileCard(p, username) {
    const skills = p.data_model.skills_model.level;
    const colLvl = p.data_model.collections_model.collection_level;
    const colExp = p.data_model.collections_model.collection_exp;
    const purse = p.data_model.stats_model.purse.toLocaleString('pt-BR');

    // Cálculo do Nível SkyBlock (Sua regra: 10 XP por nível)
    const totalLevels = Object.values(skills).reduce((a, b) => a + b, 0) + 
                        Object.values(colLvl).reduce((a, b) => a + b, 0);
    const totalXP = totalLevels * 10;
    const sbLevel = Math.floor(totalXP / 100);
    const progressXP = totalXP % 100;

    // Filtro de Coleções Ativas
    const activeCols = Object.keys(colLvl).filter(key => colLvl[key] > 0);

    container.innerHTML = `
        <div class="profile-card-detail">
            <div class="card-header">
                <div class="header-left">
                    <img src="https://mc-heads.net/body/${username}/right" class="player-body">
                    <div class="player-meta">
                        <h2>${username} <span>(${p.profile_name})</span></h2>
                        <p class="purse-text">💰 ${purse}</p>
                        <p class="xp-text">Nível XP Vanilla: ${p.experience}</p>
                    </div>
                </div>

                <div class="sb-level-box">
                    <div class="sb-level-title">Nível SkyBlock</div>
                    <div class="sb-level-number">${sbLevel}</div>
                    <div class="sb-level-bar-container">
                        <div class="sb-level-bar-fill" style="width: ${progressXP}%"></div>
                    </div>
                    <div class="sb-level-text">${progressXP} / 100 XP</div>
                </div>
            </div>

            <div class="skills-section">
                <h4>Habilidades</h4>
                <div class="skills-grid-inner">
                    ${Object.entries(skills).map(([name, lvl]) => `
                        <div class="skill-item-full">
                            <div class="skill-label">
                                <span>${skillNames[name] || name}</span>
                                <span class="lvl">Nível ${lvl}</span>
                            </div>
                            <div class="bar-container">
                                <div class="bar-fill" style="width: ${(lvl/25)*100}%"></div>
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>

            <div class="collections-section">
                <h4>Coleções</h4>
                <div class="collections-grid">
                    ${activeCols.map(key => `
                        <div class="collection-card">
                            <div class="col-icon">${key.charAt(0)}</div>
                            <div class="col-info">
                                <h5>${formatCollectionName(key)} <span class="col-lvl">${colLvl[key]}</span></h5>
                                <span class="col-amount">${colExp[key].toLocaleString('pt-BR')} coletados</span>
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>
        </div>
    `;
}

// --- INICIALIZAÇÃO ---

document.addEventListener('DOMContentLoaded', () => {
    renderHistory();
    
    const btn = document.getElementById('search-btn');
    const input = document.getElementById('player-search');
    
    if (btn && input) {
        btn.addEventListener('click', () => fetchPlayerProfile(input.value));
        input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') fetchPlayerProfile(input.value);
        });
    }
});