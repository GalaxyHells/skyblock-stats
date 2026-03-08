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
    // 1. Uso de Optional Chaining (?.) e Nullish Coalescing (??) para evitar o erro de 'undefined'
    const skills = p?.data_model?.skills_model?.level ?? {};
    const colLvl = p?.data_model?.collections_model?.collection_level ?? {};
    const colExp = p?.data_model?.collections_model?.collection_exp ?? {};
    
    // Se não houver purse (dinheiro), define como 0
    const purseValue = p?.data_model?.stats_model?.purse ?? 0;
    const purse = purseValue.toLocaleString('pt-BR');

    // 2. Cálculo do Nível SkyBlock com segurança
    const totalSkillLevels = Object.values(skills).reduce((a, b) => a + b, 0);
    const totalColLevels = Object.values(colLvl).reduce((a, b) => a + b, 0);
    
    const totalXP = (totalSkillLevels + totalColLevels) * 10;
    const sbLevel = Math.floor(totalXP / 100);
    const currentXP = totalXP % 100;

    // Filtra apenas coleções que existem e são maiores que 0
    const activeCols = Object.keys(colLvl).filter(key => colLvl[key] > 0);

    container.innerHTML = `
        <div class="profile-card-detail">
            <div class="card-header">
                <div class="header-left">
                    <img src="https://mc-heads.net/body/${username}/right" class="player-body">
                    <div class="player-meta">
                        <h2>${username} <span>(${p.profile_name || 'Perfil'})</span></h2>
                        <p class="purse-text">💰 ${purse}</p>
                        <p class="xp-text">Nível Vanilla: ${p.experience ?? 0}</p>
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
                        ${Object.entries(skills).length > 0 ? 
                            Object.entries(skills).map(([name, lvl]) => `
                                <div class="skill-mini">
                                    <span>${skillNames[name] || name}</span>
                                    <strong>${lvl}</strong>
                                </div>
                            `).join('') : '<p>Nenhuma skill evoluída</p>'
                        }
                    </div>
                </div>
                
                <div class="stat-item">
                    <h4>Coleções</h4>
                    <div class="skills-list-mini">
                        ${activeCols.length > 0 ? 
                            activeCols.slice(0, 6).map(key => `
                                <div class="skill-mini">
                                    <span>${key.replace(/_/g, ' ')}</span>
                                    <strong>${colLvl[key]}</strong>
                                </div>
                            `).join('') : '<p>Nenhuma coleção</p>'
                        }
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