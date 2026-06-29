package connection_deck

import "encoding/json"

var (
	_ json.Marshaler = (*Connection)(nil)
	_ json.Marshaler = (*Deck)(nil)
)

func (c *Connection) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"id":       c.ID().String(),
		"name":     c.Name(),
		"bucket":   c.Bucket(),
		"server":   c.Server(),
		"region":   c.Region(),
		"provider": c.Provider().String(),
		"readOnly": c.ReadOnly(),
		"revision": c.Revision(),
		"tls":      c.useTLS,
	})
}

func (d *Deck) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"connections": d.connections,
		"selectedId":  d.selectedID.String(),
	})
}
