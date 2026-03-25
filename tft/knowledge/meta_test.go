package knowledge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadMetaData(t *testing.T) {
	// 创建loader
	loader := NewLoader("data")
	
	// 加载所有数据
	store, err := loader.LoadAll()
	assert.NoError(t, err)
	assert.NotNil(t, store)
	
	// 检查Meta阵容数据
	metaComps := store.GetAllMetaComps()
	t.Logf("Loaded %d MetaComps", len(metaComps))
	assert.Greater(t, len(metaComps), 0, "应该至少有一个Meta阵容")
	
	// 检查Meta英雄数据
	metaChampions := store.GetAllMetaChampions()
	t.Logf("Loaded %d MetaChampions", len(metaChampions))
	assert.Greater(t, len(metaChampions), 0, "应该至少有一个Meta英雄")
	
	// 检查Meta装备数据
	metaItems := store.GetAllMetaItems()
	t.Logf("Loaded %d MetaItems", len(metaItems))
	assert.Greater(t, len(metaItems), 0, "应该至少有一个Meta装备")
	
	// 测试通过ID查询Meta阵容
	if len(metaComps) > 0 {
		comp := metaComps[0]
		found, ok := store.GetMetaCompByID(comp.ClusterID)
		assert.True(t, ok)
		assert.Equal(t, comp.ClusterID, found.ClusterID)
	}
	
	// 测试通过名称查询Meta英雄
	if len(metaChampions) > 0 {
		champ := metaChampions[0]
		found, ok := store.GetMetaChampionByName(champ.Name)
		assert.True(t, ok)
		assert.Equal(t, champ.Name, found.Name)
	}
	
	// 测试通过名称查询Meta装备
	if len(metaItems) > 0 {
		item := metaItems[0]
		found, ok := store.GetMetaItemByName(item.Name)
		assert.True(t, ok)
		assert.Equal(t, item.Name, found.Name)
	}
	
	// 测试搜索Meta阵容
	if len(metaComps) > 0 {
		// 使用第一个阵容的名称进行搜索
		compName := metaComps[0].DisplayNames[0].Name
		results := store.SearchMetaComps(compName)
		assert.Greater(t, len(results), 0, "搜索应该返回结果")
	}
	
	t.Log("All Meta data tests passed!")
}
