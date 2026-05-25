package games

// GameState holds the current state for each game
type GameState struct {
	CurrentNextWinningNumber   int
	CurrentNextWinningNumberFT  int
	CurrentNextMultiplier      int
	CurrentNextWinningNumberTF string
	CurrentNextWinningNumberAB int
}

var State = &GameState{}

// GetCurrentWinningNumber returns the current next winning number for a game
func (gs *GameState) GetCurrentWinningNumber(game string) interface{} {
	switch game {
	case "roulette":
		return gs.CurrentNextWinningNumber
	case "funtarget":
		return gs.CurrentNextWinningNumberFT
	case "triplefun":
		return gs.CurrentNextWinningNumberTF
	case "andarbahar":
		return gs.CurrentNextWinningNumberAB
	default:
		return nil
	}
}

// GetMultiplier returns the current multiplier for FunTarget
func (gs *GameState) GetMultiplier() int {
	return gs.CurrentNextMultiplier
}

// SetCurrentWinningNumber sets the current next winning number for a game
func (gs *GameState) SetCurrentWinningNumber(game string, value interface{}) {
	switch game {
	case "roulette":
		if v, ok := value.(int); ok {
			gs.CurrentNextWinningNumber = v
		}
	case "funtarget":
		if v, ok := value.(int); ok {
			gs.CurrentNextWinningNumberFT = v
		}
	case "triplefun":
		if v, ok := value.(string); ok {
			gs.CurrentNextWinningNumberTF = v
		}
	case "andarbahar":
		if v, ok := value.(int); ok {
			gs.CurrentNextWinningNumberAB = v
		}
	}
}

// SetMultiplier sets the multiplier for FunTarget
func (gs *GameState) SetMultiplier(multiplier int) {
	gs.CurrentNextMultiplier = multiplier
}

