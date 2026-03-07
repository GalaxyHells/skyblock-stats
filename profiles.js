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

async function fetchPlayerProfile(username) {
    const API_KEY = "UNLIMITED_KEY";
    container.innerHTML = '<div class="loader">Consultando proxy local...</div>';

    try {
        // Agora apontamos para o seu servidor Go
        const response = await fetch(`https://skyblock-stats.onrender.com/profile?id=${username}&key=${API_KEY}`);
        
        if (!response.ok) throw new Error("Jogador não encontrado ou erro no proxy");

        const data = await response.json();
        
        // Seguindo a lógica do seu JSON:
        const activeIdx = data.activeProfile;
        const profile = data.profiles[activeIdx];

        renderProfileCard(profile, username);
    } catch (error) {
        container.innerHTML = `<div class="error">${error.message}</div>`;
    }
}

function renderProfileCard(p, username) {
    // Extraindo dados do JSON
    const purse = p.data_model.bank_model.purse.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' }).replace('R$', '$');
    const skills = p.data_model.skills_model.level;
    const activePet = p.data_model.pet_model.active_pet;

    // Encontrando a maior skill para destaque
    const topSkill = Object.entries(skills).reduce((a, b) => a[1] > b[1] ? a : b);

    // Lógica das Coleções
    const colLvl = p.data_model.collections_model.collection_level;
    const colExp = p.data_model.collections_model.collection_exp;

    // Filtra apenas as coleções maiores que nível 0
    const activeCollections = Object.keys(colLvl).filter(key => colLvl[key] > 0);

    // --- CÁLCULO DO NÍVEL SKYBLOCK ---
    // 1. Soma todos os níveis de habilidades
    const totalSkillLevels = Object.values(skills).reduce((acc, curr) => acc + curr, 0);

    // 2. Soma todos os níveis de coleções
    const totalColLevels = Object.values(colLvl).reduce((acc, curr) => acc + curr, 0);

    // 3. Aplica a regra: cada nível = 10 XP
    const totalXP = (totalSkillLevels + totalColLevels) * 10;

    // 4. Calcula o nível atual e o XP restante para o próximo nível
    const skyblockLevel = Math.floor(totalXP / 100);
    const currentLevelXP = totalXP % 100; 
    // ---------------------------------

    // Cria o HTML das coleções
    const collectionsHTML = `
        <div class="collections-section">
            <h4>Coleções Desbloqueadas</h4>
            <div class="collections-grid">
                ${activeCollections.map(key => {
                    const name = formatCollectionName(key);
                    const lvl = colLvl[key];
                    // Formata o número (ex: 56333277 vira 56.333.277)
                    const exp = colExp[key].toLocaleString('pt-BR'); 
                    
                    return `
                        <div class="collection-card">
                            <div class="col-icon">${name.charAt(0)}</div>
                            <div class="col-info">
                                <h5>${name} <span class="col-lvl">${lvl}</span></h5>
                                <span class="col-amount">${exp} coletados</span>
                            </div>
                        </div>
                    `;
                }).join('')}
            </div>
        </div>
    `;

    container.innerHTML = `
        <div class="profile-card-detail">
            <div class="card-header">
                <div class="header-left">
                    <img src="https://mc-heads.net/body/${username}/right" alt="${username}" class="player-body">
                    <div class="player-meta">
                        <h2>${username} <span>(${p.profile_name})</span></h2>
                        <p class="purse-text">💰 ${purse}</p>
                        <p class="xp-text">Nível de XP (Vanilla): ${p.experience}</p>
                    </div>
                </div>

                <div class="sb-level-box">
                    <div class="sb-level-title">Nível SkyBlock</div>
                    <div class="sb-level-number">${skyblockLevel}</div>
                    <div class="sb-level-bar-container">
                        <div class="sb-level-bar-fill" style="width: ${currentLevelXP}%"></div>
                    </div>
                    <div class="sb-level-text">${currentLevelXP} / 100 XP</div>
                </div>
            </div>

            <div class="stats-grid">
                <div class="stat-item">
                    <span class="label">Melhor Skill</span>
                    <span class="value">${topSkill[0].replace('_SKILL', '')} ${topSkill[1]}</span>
                </div>
                <div class="stat-item">
                    <span class="label">Pet Ativo</span>
                    <span class="value">${activePet ? activePet.petType : 'Nenhum'} (Lvl ${activePet ? activePet.level : 0})</span>
                </div>
                <div class="stat-item">
                    <span class="label">Região Atual</span>
                    <span class="value">${p.data_model.regions_model.region}</span>
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
                                <div class="bar-fill" style="width: ${lvl * (100 / maxSkillLevel)}%"></div>
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>

            ${collectionsHTML}
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