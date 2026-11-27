package game

// Outcome is empty in sandbox scenes. Otherwise it changes once per round.
type Outcome string

const (
	Playing Outcome = "Playing"
	Won     Outcome = "Won"
	Lost    Outcome = "Lost"
)

// ObjectiveStatus is a snapshot for the HUD. Tick is the last evaluated tick.
type ObjectiveStatus struct {
	Text, Signal    string
	Tick, TickLimit uint64
	Outcome         Outcome
}

// Finished reports whether this round reached its goal or deadline.
func (o ObjectiveStatus) Finished() bool { return o.Outcome == Won || o.Outcome == Lost }

// Objective returns the session's current goal and result.
func (s *Session) Objective() ObjectiveStatus { return s.objective }

// FinishRound records a terminal result after a successful scripted tick.
func (s *Session) FinishRound(won bool) {
	if s.objective.Finished() {
		return
	}
	s.objective.Outcome = Lost
	if won {
		s.objective.Outcome = Won
	}
	s.objective.Tick = s.World.Stats().Tick
	s.queueResultSound()
	s.publishSounds()
}
