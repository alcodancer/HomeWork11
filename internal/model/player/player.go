// internal/model/player/player.go
package player

type Player struct {
	name string
	id   int
}

func NewPlayer(name string, id int) *Player {
	return &Player{
		name: name,
		id:   id,
	}
}

func (p *Player) GetName() string {
	return p.name
}

func (p *Player) SetName(name string) {
	if name != "" {
		p.name = name
	}
}

func (p *Player) GetID() int {
	return p.id
}

func (p *Player) SetID(id int) {
	p.id = id
}

func (p *Player) String() string {
	return p.name
}