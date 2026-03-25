package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sagerlabs/awesome/tft/knowledge/models"
	"gopkg.in/yaml.v3"
)

// Loader 数据加载器
type Loader struct {
	dataDir string
}

// NewLoader 创建数据加载器
func NewLoader(dataDir string) *Loader {
	return &Loader{dataDir: dataDir}
}

// LoadAll 加载所有数据
func (l *Loader) LoadAll() (*Store, error) {
	store := NewStore()

	// 加载英雄数据
	if err := l.loadChampions(store); err != nil {
		return nil, fmt.Errorf("load champions: %w", err)
	}

	// 加载装备数据
	if err := l.loadItems(store); err != nil {
		return nil, fmt.Errorf("load items: %w", err)
	}

	// 加载羁绊数据
	if err := l.loadTraits(store); err != nil {
		return nil, fmt.Errorf("load traits: %w", err)
	}

	// 加载阵容数据
	if err := l.loadTeamComps(store); err != nil {
		return nil, fmt.Errorf("load team comps: %w", err)
	}

	// 加载知识文档
	if err := l.loadKnowledgeDocs(store); err != nil {
		return nil, fmt.Errorf("load knowledge docs: %w", err)
	}

	return store, nil
}

// loadChampions 加载英雄数据
func (l *Loader) loadChampions(store *Store) error {
	championsDir := filepath.Join(l.dataDir, "champions")
	files, err := os.ReadDir(championsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，跳过
		}
		return err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		path := filepath.Join(championsDir, file.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		// 先尝试加载Meta格式
		var mc models.MetaChampion
		if err := json.Unmarshal(b, &mc); err == nil && mc.Name != "" {
			store.AddMetaChampion(&mc)
			continue
		}

		// 如果不是Meta格式，尝试加载原有格式
		var champ models.Champion
		if err := json.Unmarshal(b, &champ); err != nil {
			return fmt.Errorf("unmarshal %s: %w", path, err)
		}
		store.AddChampion(&champ)
	}

	return nil
}

// loadItems 加载装备数据
func (l *Loader) loadItems(store *Store) error {
	itemsDir := filepath.Join(l.dataDir, "items")
	files, err := os.ReadDir(itemsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，跳过
		}
		return err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		path := filepath.Join(itemsDir, file.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		// 先尝试加载Meta格式
		var mi models.MetaItem
		if err := json.Unmarshal(b, &mi); err == nil && mi.Name != "" {
			store.AddMetaItem(&mi)
			continue
		}

		// 如果不是Meta格式，尝试加载原有格式
		var item models.Item
		if err := json.Unmarshal(b, &item); err != nil {
			return fmt.Errorf("unmarshal %s: %w", path, err)
		}
		store.AddItem(&item)
	}

	return nil
}

// loadTraits 加载羁绊数据
func (l *Loader) loadTraits(store *Store) error {
	traitsDir := filepath.Join(l.dataDir, "traits")
	files, err := os.ReadDir(traitsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，跳过
		}
		return err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		path := filepath.Join(traitsDir, file.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		var trait models.Trait
		if err := json.Unmarshal(b, &trait); err != nil {
			return fmt.Errorf("unmarshal %s: %w", path, err)
		}

		store.AddTrait(&trait)
	}

	return nil
}

// loadTeamComps 加载阵容数据
func (l *Loader) loadTeamComps(store *Store) error {
	teamCompsDir := filepath.Join(l.dataDir, "team_comps")
	files, err := os.ReadDir(teamCompsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，跳过
		}
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		path := filepath.Join(teamCompsDir, file.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		// 先尝试加载YAML格式（原有格式）
		if strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml") {
			var tc models.TeamComp
			if err := yaml.Unmarshal(b, &tc); err != nil {
				return fmt.Errorf("unmarshal yaml %s: %w", path, err)
			}
			store.AddTeamComp(&tc)
			continue
		}

		// 尝试加载JSON格式（MetaTFT格式）
		if strings.HasSuffix(file.Name(), ".json") {
			var mc models.MetaComp
			if err := json.Unmarshal(b, &mc); err != nil {
				return fmt.Errorf("unmarshal json %s: %w", path, err)
			}
			store.AddMetaComp(&mc)
			continue
		}
	}

	return nil
}

// loadKnowledgeDocs 加载知识文档
func (l *Loader) loadKnowledgeDocs(store *Store) error {
	knowledgeDir := filepath.Join(l.dataDir, "knowledge")
	files, err := os.ReadDir(knowledgeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，跳过
		}
		return err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		path := filepath.Join(knowledgeDir, file.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		// 简单的 Markdown 文档解析：用文件名作为 ID 和标题
		id := strings.TrimSuffix(file.Name(), ".md")
		doc := &models.KnowledgeDoc{
			ID:       id,
			Title:    id,
			Category: "strategy",
			Content:  string(b),
			Version:  "1.0",
		}

		store.AddKnowledgeDoc(doc)
	}

	return nil
}
