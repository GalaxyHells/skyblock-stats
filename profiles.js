// Configurações
const PROXY_URL = "https://skyblock-stats.onrender.com"; 
const container = document.getElementById('profiles-container');

// Dicionário de tradução para as Skills
const skillNames = {
    "FORAGING_SKILL": "Forrageamento",
    "COMBAT_SKILL": "Combate",
    "MINING_SKILL": "Mineração",
    "FISHING_SKILL": "Pescaria",
    "FARMING_SKILL": "Herbalismo",
    "ENCHANTING_SKILL": "Encantamento"
};

// --- FUNÇÕES DE HISTÓRICO ---

function saveToHistory(username) {
    let history = JSON.parse(localStorage.getItem('searchHistory')) || [];
    // Remove duplicatas e coloca o mais recente no topo
    history = history.filter(name => name.toLowerCase() !== username.toLowerCase());
    history.unshift(username);
    // Mantém apenas os 5 últimos
    if (history.length > 5) history.pop();
    
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
        <div class="history-list">
            ${history.map(name => `
                <button class="history-btn" onclick="fetchPlayerProfile('${name}')">${name}</button>
            `).join('')}
        </div>
    `;
}

// --- BUSCA DE DADOS ---

async function fetchPlayerProfile(username) {
    if (!username || username.trim() === "") return;
    
    const cleanUsername = username.trim().toLowerCase();
    container.innerHTML = '<div class="loader">Buscando informações no servidor...</div>';

    try {
        // Chamada para o seu servidor Go (o proxy já cuida da API KEY internamente)
        const response = await fetch(`${PROXY_URL}/profile?id=${cleanUsername}`);
        
        if (!response.ok) throw new Error("Jogador não encontrado ou erro no servidor.");

        const data = await response.json();
        const activeIdx = data.activeProfile;
        const profile = data.profiles[activeIdx];

        // Salva no histórico após o sucesso
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
    const purse = p.data_model.stats_model.purse.toLocaleString('pt-BR');

    // CÁLCULO DO NÍVEL SKYBLOCK PERSONALIZADO
    // 1 nível (skill ou coleção) = 10 XP. 100 XP = 1 Nível SB.
    const totalSkillLevels = Object.values(skills).reduce((a, b) => a + b, 0);
    const totalColLevels = Object.values(colLvl).reduce((a, b) => a + b, 0);
    
    const totalXP = (totalSkillLevels + totalColLevels) * 10;
    const sbLevel = Math.floor(totalXP / 100);
    const currentXP = totalXP % 100; // O que sobra para a barra

    container.innerHTML = `
        <div class="profile-card-detail">
            <div class="card-header">
                <div class="header-left">
                    <img src="https://mc-heads.net/body/${username}/right" class="player-body">
                    <div class="player-meta">
                        <h2>${username} <span>(${p.profile_name})</span></h2>
                        <p class="purse-text">💰 ${purse}</p>
                        <p class="xp-text">Nível Vanilla: ${p.experience}</p>
                    </div>
                </div>

                <div class="sb-level-box">
                    <div class="sb-level-title">Nível SkyBlock</div>
                    <div class="sb-level-number">${sbLevel}</div>
                    <div class="sb-level-bar-container">
                        <div class="sb-level-bar-fill" style="width: ${currentXP}%"></div>
                    </div>
                    <div class="sb-level-text">${currentXP} / 100 XP</div>
                </div>
            </div>

            <div class="stats-grid">
                <div class="stat-item">
                    <h4>Habilidades</h4>
                    <div class="skills-list-mini">
                        ${Object.entries(skills).map(([name, lvl]) => `
                            <div class="skill-mini">
                                <span>${skillNames[name] || name}</span>
                                <strong>${lvl}</strong>
                            </div>
                        `).join('')}
                    </div>
                </div>
                
                <div class="stat-item">
                    <h4>Coleções (Top)</h4>
                    <div class="skills-list-mini">
                        ${Object.entries(colLvl).slice(0, 6).map(([name, lvl]) => `
                            <div class="skill-mini">
                                <span>${name.replace('_', ' ')}</span>
                                <strong>${lvl}</strong>
                            </div>
                        `).join('')}
                    </div>
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