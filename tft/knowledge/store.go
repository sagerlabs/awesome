package knowledge

import (
	"strings"

	"github.com/sagerlabs/awesome/tft/knowledge/models"
)

// Store 知识库内存存储和索引
type Store struct {
	champions       map[string]*models.Champion     // id -> Champion
	championsByName map[string]*models.Champion     // name -> Champion
	items           map[string]*models.Item         // id -> Item
	itemsByName     map[string]*models.Item         // name -> Item
	traits          map[string]*models.Trait        // id -> Trait
	traitsByName    map[string]*models.Trait        // name -> Trait
	teamComps       map[string]*models.TeamComp     // id -> TeamComp
	teamCompsByName map[string]*models.TeamComp     // name -> TeamComp
	knowledgeDocs   map[string]*models.KnowledgeDoc // id -> KnowledgeDoc
	patchNotes      map[string]*models.PatchNote    // patch -> PatchNote
	aliases         map[string]map[string]string    // type -> raw lower -> normalized

	// Meta数据
	metaComps        map[string]*models.MetaComp     // cluster_id -> MetaComp
	metaCompsByName  map[string]*models.MetaComp     // name -> MetaComp
	metaChampions    map[string]*models.MetaChampion // name -> MetaChampion
	metaItems        map[string]*models.MetaItem     // name -> MetaItem
	championProfiles map[string]*models.ChampionProfile
}

// NewStore 创建空的 Store
func NewStore() *Store {
	return &Store{
		champions:        make(map[string]*models.Champion),
		championsByName:  make(map[string]*models.Champion),
		items:            make(map[string]*models.Item),
		itemsByName:      make(map[string]*models.Item),
		traits:           make(map[string]*models.Trait),
		traitsByName:     make(map[string]*models.Trait),
		teamComps:        make(map[string]*models.TeamComp),
		teamCompsByName:  make(map[string]*models.TeamComp),
		knowledgeDocs:    make(map[string]*models.KnowledgeDoc),
		patchNotes:       make(map[string]*models.PatchNote),
		aliases:          make(map[string]map[string]string),
		metaComps:        make(map[string]*models.MetaComp),
		metaCompsByName:  make(map[string]*models.MetaComp),
		metaChampions:    make(map[string]*models.MetaChampion),
		metaItems:        make(map[string]*models.MetaItem),
		championProfiles: make(map[string]*models.ChampionProfile),
	}
}

// AddAliases merges player slang aliases into the store.
func (s *Store) AddAliases(file models.AliasesFile) {
	s.addAliasGroup("champion", file.Heroes)
	s.addAliasGroup("item", file.Items)
	s.addAliasGroup("trait", file.Traits)
}

func (s *Store) addAliasGroup(kind string, entries map[string]string) {
	if len(entries) == 0 {
		return
	}
	group := s.aliases[kind]
	if group == nil {
		group = make(map[string]string, len(entries))
		s.aliases[kind] = group
	}
	for raw, normalized := range entries {
		raw = normalizeAliasKey(raw)
		normalized = strings.TrimSpace(normalized)
		if raw == "" || normalized == "" {
			continue
		}
		group[raw] = normalized
	}
}

// ResolveAlias returns a normalized knowledge term for a player slang term.
func (s *Store) ResolveAlias(kind string, raw string) (string, bool) {
	if s == nil {
		return "", false
	}
	group := s.aliases[kind]
	if len(group) == 0 {
		return "", false
	}
	normalized, ok := group[normalizeAliasKey(raw)]
	return normalized, ok
}

func normalizeAliasKey(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// AddChampion 添加英雄到 Store
func (s *Store) AddChampion(champ *models.Champion) {
	s.champions[champ.ID] = champ
	s.championsByName[strings.ToLower(champ.Name)] = champ
}

// AddItem 添加装备到 Store
func (s *Store) AddItem(item *models.Item) {
	s.items[item.ID] = item
	s.itemsByName[strings.ToLower(item.Name)] = item
}

// AddTrait 添加羁绊到 Store
func (s *Store) AddTrait(trait *models.Trait) {
	s.traits[trait.ID] = trait
	s.traitsByName[strings.ToLower(trait.Name)] = trait
}

// AddTeamComp 添加阵容到 Store
func (s *Store) AddTeamComp(tc *models.TeamComp) {
	s.teamComps[tc.ID] = tc
	s.teamCompsByName[strings.ToLower(tc.Name)] = tc
}

// AddKnowledgeDoc 添加知识文档到 Store
func (s *Store) AddKnowledgeDoc(doc *models.KnowledgeDoc) {
	s.knowledgeDocs[doc.ID] = doc
}

// AddPatchNote 添加版本公告到 Store。
func (s *Store) AddPatchNote(note *models.PatchNote) {
	if note == nil || strings.TrimSpace(note.Patch) == "" {
		return
	}
	s.patchNotes[strings.ToLower(strings.TrimSpace(note.Patch))] = note
}

// GetChampionByID 通过 ID 查询英雄
func (s *Store) GetChampionByID(id string) (*models.Champion, bool) {
	champ, ok := s.champions[id]
	return champ, ok
}

// GetChampionByName 通过名称查询英雄（不区分大小写）
func (s *Store) GetChampionByName(name string) (*models.Champion, bool) {
	champ, ok := s.championsByName[strings.ToLower(name)]
	return champ, ok
}

// GetAllChampions 获取所有英雄
func (s *Store) GetAllChampions() []*models.Champion {
	champs := make([]*models.Champion, 0, len(s.champions))
	for _, champ := range s.champions {
		champs = append(champs, champ)
	}
	return champs
}

// GetItemByID 通过 ID 查询装备
func (s *Store) GetItemByID(id string) (*models.Item, bool) {
	item, ok := s.items[id]
	return item, ok
}

// GetItemByName 通过名称查询装备（不区分大小写）
func (s *Store) GetItemByName(name string) (*models.Item, bool) {
	item, ok := s.itemsByName[strings.ToLower(name)]
	return item, ok
}

// GetAllItems 获取所有装备
func (s *Store) GetAllItems() []*models.Item {
	items := make([]*models.Item, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items
}

// GetTraitByID 通过 ID 查询羁绊
func (s *Store) GetTraitByID(id string) (*models.Trait, bool) {
	trait, ok := s.traits[id]
	return trait, ok
}

// GetTraitByName 通过名称查询羁绊（不区分大小写）
func (s *Store) GetTraitByName(name string) (*models.Trait, bool) {
	trait, ok := s.traitsByName[strings.ToLower(name)]
	return trait, ok
}

// GetAllTraits 获取所有羁绊
func (s *Store) GetAllTraits() []*models.Trait {
	traits := make([]*models.Trait, 0, len(s.traits))
	for _, trait := range s.traits {
		traits = append(traits, trait)
	}
	return traits
}

// GetTeamCompByID 通过 ID 查询阵容
func (s *Store) GetTeamCompByID(id string) (*models.TeamComp, bool) {
	tc, ok := s.teamComps[id]
	return tc, ok
}

// GetTeamCompByName 通过名称查询阵容（不区分大小写）
func (s *Store) GetTeamCompByName(name string) (*models.TeamComp, bool) {
	tc, ok := s.teamCompsByName[strings.ToLower(name)]
	return tc, ok
}

// GetAllTeamComps 获取所有阵容
func (s *Store) GetAllTeamComps() []*models.TeamComp {
	tcs := make([]*models.TeamComp, 0, len(s.teamComps))
	for _, tc := range s.teamComps {
		tcs = append(tcs, tc)
	}
	return tcs
}

// GetKnowledgeDocByID 通过 ID 查询知识文档
func (s *Store) GetKnowledgeDocByID(id string) (*models.KnowledgeDoc, bool) {
	doc, ok := s.knowledgeDocs[id]
	return doc, ok
}

// GetAllKnowledgeDocs 获取所有知识文档
func (s *Store) GetAllKnowledgeDocs() []*models.KnowledgeDoc {
	docs := make([]*models.KnowledgeDoc, 0, len(s.knowledgeDocs))
	for _, doc := range s.knowledgeDocs {
		docs = append(docs, doc)
	}
	return docs
}

// GetAllPatchNotes 获取所有版本公告。
func (s *Store) GetAllPatchNotes() []*models.PatchNote {
	notes := make([]*models.PatchNote, 0, len(s.patchNotes))
	for _, note := range s.patchNotes {
		notes = append(notes, note)
	}
	return notes
}

// SearchChampions 搜索英雄（简单的关键词搜索）
func (s *Store) SearchChampions(query string) []*models.Champion {
	var results []*models.Champion
	queryLower := strings.ToLower(query)

	for _, champ := range s.champions {
		if strings.Contains(strings.ToLower(champ.Name), queryLower) ||
			strings.Contains(strings.ToLower(champ.ID), queryLower) {
			results = append(results, champ)
			continue
		}

		// 搜索羁绊
		for _, trait := range champ.Traits {
			if strings.Contains(strings.ToLower(trait), queryLower) {
				results = append(results, champ)
				break
			}
		}
	}

	return results
}

// SearchItems 搜索装备（简单的关键词搜索）
func (s *Store) SearchItems(query string) []*models.Item {
	var results []*models.Item
	queryLower := strings.ToLower(query)

	for _, item := range s.items {
		if strings.Contains(strings.ToLower(item.Name), queryLower) ||
			strings.Contains(strings.ToLower(item.ID), queryLower) {
			results = append(results, item)
		}
	}

	return results
}

// AddMetaComp 添加Meta阵容到 Store
func (s *Store) AddMetaComp(mc *models.MetaComp) {
	s.metaComps[mc.ClusterID] = mc
	// 使用第一个显示名称作为索引
	if len(mc.DisplayNames) > 0 {
		s.metaCompsByName[strings.ToLower(mc.DisplayNames[0].Name)] = mc
	}
	// 也使用name_string作为索引
	if mc.NameString != "" {
		s.metaCompsByName[strings.ToLower(mc.NameString)] = mc
	}
}

// AddMetaChampion 添加Meta英雄到 Store
func (s *Store) AddMetaChampion(mc *models.MetaChampion) {
	s.metaChampions[strings.ToLower(mc.Name)] = mc
}

// AddMetaItem 添加Meta装备到 Store
func (s *Store) AddMetaItem(mi *models.MetaItem) {
	s.metaItems[strings.ToLower(mi.Name)] = mi
}

// AddChampionProfile 添加英雄画像到 Store。
func (s *Store) AddChampionProfile(name string, profile *models.ChampionProfile) {
	if profile == nil {
		return
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = strings.TrimSpace(profile.Name)
	}
	if trimmed == "" {
		return
	}
	if profile.Name == "" {
		profile.Name = trimmed
	}
	s.championProfiles[strings.ToLower(trimmed)] = profile
}

// GetChampionProfileByName 通过名称查询英雄画像。
func (s *Store) GetChampionProfileByName(name string) (*models.ChampionProfile, bool) {
	profile, ok := s.championProfiles[strings.ToLower(strings.TrimSpace(name))]
	return profile, ok
}

// GetAllChampionProfiles 获取所有英雄画像。
func (s *Store) GetAllChampionProfiles() []*models.ChampionProfile {
	profiles := make([]*models.ChampionProfile, 0, len(s.championProfiles))
	for _, profile := range s.championProfiles {
		profiles = append(profiles, profile)
	}
	return profiles
}

// GetMetaCompByID 通过ClusterID查询Meta阵容
func (s *Store) GetMetaCompByID(clusterID string) (*models.MetaComp, bool) {
	mc, ok := s.metaComps[clusterID]
	return mc, ok
}

// GetMetaCompByName 通过名称查询Meta阵容（不区分大小写）
func (s *Store) GetMetaCompByName(name string) (*models.MetaComp, bool) {
	mc, ok := s.metaCompsByName[strings.ToLower(name)]
	return mc, ok
}

// GetAllMetaComps 获取所有Meta阵容
func (s *Store) GetAllMetaComps() []*models.MetaComp {
	mcs := make([]*models.MetaComp, 0, len(s.metaComps))
	for _, mc := range s.metaComps {
		mcs = append(mcs, mc)
	}
	return mcs
}

// GetMetaChampionByName 通过名称查询Meta英雄（不区分大小写）
func (s *Store) GetMetaChampionByName(name string) (*models.MetaChampion, bool) {
	mc, ok := s.metaChampions[strings.ToLower(name)]
	return mc, ok
}

// GetAllMetaChampions 获取所有Meta英雄
func (s *Store) GetAllMetaChampions() []*models.MetaChampion {
	mcs := make([]*models.MetaChampion, 0, len(s.metaChampions))
	for _, mc := range s.metaChampions {
		mcs = append(mcs, mc)
	}
	return mcs
}

// GetMetaItemByName 通过名称查询Meta装备（不区分大小写）
func (s *Store) GetMetaItemByName(name string) (*models.MetaItem, bool) {
	mi, ok := s.metaItems[strings.ToLower(name)]
	return mi, ok
}

// GetAllMetaItems 获取所有Meta装备
func (s *Store) GetAllMetaItems() []*models.MetaItem {
	mis := make([]*models.MetaItem, 0, len(s.metaItems))
	for _, mi := range s.metaItems {
		mis = append(mis, mi)
	}
	return mis
}

// SearchMetaComps 搜索Meta阵容（简单的关键词搜索）
func (s *Store) SearchMetaComps(query string) []*models.MetaComp {
	var results []*models.MetaComp
	queryLower := strings.ToLower(query)

	for _, mc := range s.metaComps {
		// 搜索显示名称
		found := false
		for _, dn := range mc.DisplayNames {
			if strings.Contains(strings.ToLower(dn.Name), queryLower) {
				results = append(results, mc)
				found = true
				break
			}
		}
		if found {
			continue
		}

		// 搜索name_string
		if strings.Contains(strings.ToLower(mc.NameString), queryLower) {
			results = append(results, mc)
			continue
		}

		// 搜索英雄
		for _, unit := range mc.Units {
			if strings.Contains(strings.ToLower(unit), queryLower) {
				results = append(results, mc)
				found = true
				break
			}
		}
		if found {
			continue
		}

		// 搜索羁绊
		for _, trait := range mc.Traits {
			if strings.Contains(strings.ToLower(trait), queryLower) {
				results = append(results, mc)
				break
			}
		}
	}

	return results
}
