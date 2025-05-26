package main

import (
	"fmt"
)

func NewGame() Game {
	return Game{
		Map:      make(map[string]Area),
		ItemList: make(map[string]Item),
		NPCs:     make(map[string]Character),
	}
}

type Game struct {
	Player   Character
	Map      map[string]Area
	ItemList map[string]Item
	NPCs     map[string]Character
}

// AddItem adds an item to the game's ItemList
func (g *Game) AddItem(item Item) error {
	if item.Name == "" {
		return fmt.Errorf("item must have a name")
	}
	g.ItemList[item.Name] = item
	return nil
}

// RemoveItem removes an item from the game's ItemList
func (g *Game) RemoveItem(itemName string) error {
	if _, exists := g.ItemList[itemName]; !exists {
		return fmt.Errorf("item %s not found", itemName)
	}
	delete(g.ItemList, itemName)
	return nil
}

// GetItem retrieves an item from the game's ItemList
func (g *Game) GetItem(itemName string) (Item, error) {
	item, exists := g.ItemList[itemName]
	if !exists {
		return Item{}, fmt.Errorf("item %s not found", itemName)
	}
	return item, nil
}

// AddArea adds an area to the game's Map
func (g *Game) AddArea(areaID string, area Area) error {
	if areaID == "" {
		return fmt.Errorf("area ID cannot be empty")
	}
	g.Map[areaID] = area
	return nil
}

// RemoveArea removes an area from the game's Map
func (g *Game) RemoveArea(areaID string) error {
	if _, exists := g.Map[areaID]; !exists {
		return fmt.Errorf("area %s not found", areaID)
	}
	delete(g.Map, areaID)
	return nil
}

// GetArea retrieves an area from the game's Map
func (g *Game) GetArea(areaID string) (Area, error) {
	area, exists := g.Map[areaID]
	if !exists {
		return Area{}, fmt.Errorf("area %s not found", areaID)
	}
	return area, nil
}

// AddNPC adds an NPC to the game's NPCs map
func (g *Game) AddNPC(npc Character) error {
	if npc.Name == "" {
		return fmt.Errorf("NPC must have a name")
	}
	g.NPCs[npc.Name] = npc
	return nil
}

// RemoveNPC removes an NPC from the game's NPCs map
func (g *Game) RemoveNPC(npcName string) error {
	if _, exists := g.NPCs[npcName]; !exists {
		return fmt.Errorf("NPC %s not found", npcName)
	}
	delete(g.NPCs, npcName)
	return nil
}

// GetNPC retrieves an NPC from the game's NPCs map
func (g *Game) GetNPC(npcName string) (Character, error) {
	npc, exists := g.NPCs[npcName]
	if !exists {
		return Character{}, fmt.Errorf("NPC %s not found", npcName)
	}
	return npc, nil
}

// AddItemToArea adds an item to a specific area
func (g *Game) AddItemToArea(areaID string, item Item) error {
	area, err := g.GetArea(areaID)
	if err != nil {
		return err
	}
	area.Items = append(area.Items, item)
	g.Map[areaID] = area
	return nil
}

// RemoveItemFromArea removes an item from a specific area
func (g *Game) RemoveItemFromArea(areaID string, itemName string) error {
	area, err := g.GetArea(areaID)
	if err != nil {
		return err
	}

	for i, item := range area.Items {
		if item.Name == itemName {
			area.Items = append(area.Items[:i], area.Items[i+1:]...)
			g.Map[areaID] = area
			return nil
		}
	}
	return fmt.Errorf("item %s not found in area %s", itemName, areaID)
}

// AddNPCToArea adds an NPC to a specific area
func (g *Game) AddNPCToArea(areaID string, npcName string) error {
	area, err := g.GetArea(areaID)
	if err != nil {
		return err
	}

	npc, err := g.GetNPC(npcName)
	if err != nil {
		return err
	}

	area.Occupants = append(area.Occupants, npcName)
	npc.Location = area
	g.Map[areaID] = area
	g.NPCs[npcName] = npc
	return nil
}

// RemoveNPCFromArea removes an NPC from a specific area
func (g *Game) RemoveNPCFromArea(areaID string, npcName string) error {
	area, err := g.GetArea(areaID)
	if err != nil {
		return err
	}

	for i, occupant := range area.Occupants {
		if occupant == npcName {
			area.Occupants = append(area.Occupants[:i], area.Occupants[i+1:]...)
			g.Map[areaID] = area
			return nil
		}
	}
	return fmt.Errorf("NPC %s not found in area %s", npcName, areaID)
}

// GetAllAreas returns all areas in the game
func (g *Game) GetAllAreas() []*Area {
	areas := make([]*Area, 0, len(g.Map))
	for _, area := range g.Map {
		areas = append(areas, &area)
	}
	return areas
}
