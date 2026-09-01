package mods

type ModInfo struct {
	Name          string `json:"name"`
	ModID         string `json:"mod_id"`
	WorkshopID    string `json:"workshop_id"`
	Description   string `json:"description"`
	IsCollection  bool   `json:"is_collection"`
	CollectionURL string `json:"collection_url,omitempty"`
}
