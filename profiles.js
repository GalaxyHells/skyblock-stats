// Configurações Globais
const PROXY_URL = "https://skyblock-stats.onrender.com"; 
const container = document.getElementById('profiles-container');

// Dicionário de tradução para as Habilidades
const skillNames = {
    "FORAGING_SKILL": "Forrageamento",
    "COMBAT_SKILL": "Combate",
    "MINING_SKILL": "Mineração",
    "FISHING_SKILL": "Pescaria",
    "FARMING_SKILL": "Herbalismo",
    "ENCHANTING_SKILL": "Encantamento"
};

// --- FUNÇÕES DE UTILIDADE ---

// Deixa os nomes das coleções bonitos (ex: POTATO_ITEM -> Potato)
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
    
    // Remove o nome se já existir para não duplicar e move para o topo
    history = history.filter(name => name.toLowerCase() !== username.toLowerCase());
    history.unshift(username);
    
    // Mantém apenas as 5 últimas buscas
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

// --- BUSCA E RENDERIZAÇÃO ---

async function fetchPlayerProfile(username) {
    if (!username || username.trim() === "") return;
    
    const cleanUsername = username.trim().toLowerCase();
    container.innerHTML = '<div class="loader">Consultando SkyBlock API...</div>';

    try {
        // Chamada para o seu proxy em Go no Render
        const response = await fetch(`${PROXY_URL}/profile?id=${cleanUsername}`);
        
        if (!response.ok) {
            throw new Error("Jogador não encontrado ou servidor offline.");
        }

        const data = await response.json();
        const activeIdx = data.activeProfile;
        const profile = data.profiles[activeIdx];

        // Se a busca deu certo, salvamos no histórico e mostramos o card
        saveToHistory(cleanUsername);
        renderProfileCard(profile, cleanUsername);
        
    } catch (error) {
        container.innerHTML = `
            <div class="error">
                <p>⚠️ ${error.message}</p>
                <small>Verifique se o nome está correto ou tente novamente em instantes.</small>
            </div>`;
    }
}

function renderProfileCard(p, username) {
    // Extraindo dados do JSON
    const skills = p.data_model.skills_model.level;
    const colLvl = p.data_model.collections_model.collection_level;
    const colExp = p.data_model.collections_model.collection_exp;
    const purse = p.data_model.stats_model.purse.toLocaleString('pt-BR');

    // CÁLCULO NÍVEL SKYBLOCK (10 XP por nível de Skill/Coleção)
    const totalSkillLevels = Object.values(skills).reduce((a, b) => a + b, 0);
    const totalColLevels = Object.values(colLvl).reduce((a, b) => a + b, 0);
    const totalXP = (totalSkillLevels + totalColLevels) * 10;
    
    const sbLevel = Math.floor(totalXP / 100);
    const progressXP = totalXP % 100;

    // Filtra apenas coleções que o jogador já começou
    const activeCollections = Object.keys(colLvl).filter(key => colLvl[key] > 0);

    container.innerHTML = `
        <div class="profile-card-detail">
            <div class="card-header">
                <div class="header-left">
                    <img src="https://mc-heads.net/body/${username}/right" alt="${username}" class="player-body">
                    <div class="player-meta">
                        <h2>${username} <span>(${p.profile_name})</span></h2>
                        <p class="purse-text">💰 ${purse}</p>
                        <p class="xp-text">XP Vanilla: ${p.experience}</p>
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
                                <span class="lvl">Lvl ${lvl}</span>
                            </div>
                            <div class="bar-container">
                                <div class="bar-fill" style="width: ${(lvl / 25) * 100}%"></div>
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>

            <div class="collections-section">
                <h4>Coleções Desbloqueadas</h4>
                <div class="collections-grid">
                    ${activeCollections.map(key => `
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